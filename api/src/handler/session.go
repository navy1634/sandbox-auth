package handler

import (
	"crypto/hmac"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-auth/src/infrastructure/session"
)

// OAuth callback の state が開始時に保存した値と一致するか確認する。
func (h *AuthHandler) validateState(c *gin.Context) error {
	// OAuth 開始時に保存した状態値とコールバックの状態値を比較する。
	expected, err := c.Cookie(h.cookieName(stateCookieName))
	if err != nil {
		return err
	}
	if expected == "" || !hmac.Equal([]byte(expected), []byte(c.Query("state"))) {
		return errors.New("state mismatch")
	}
	return nil
}

// ログイン Cookie を検証してセッションユーザーを返す。
func (h *AuthHandler) readSession(c *gin.Context) (session.User, error) {
	// ログイン Cookie を読み取り、署名検証済みのセッション情報へ戻す。
	value, err := c.Cookie(h.cookieName(sessionCookieName))
	if err != nil {
		return session.User{}, err
	}
	return h.sessions.VerifyContext(c.Request.Context(), value)
}

// アプリ共通の属性で認証系 Cookie を設定する。
func (h *AuthHandler) setCookie(c *gin.Context, name string, value string, maxAge int, httpOnly bool) {
	// フロントエンド URL が HTTPS のときだけ Secure 属性付き Cookie として送る。
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.cookieName(name), value, maxAge, "/", h.cookieDomain(name), h.secureCookies(), httpOnly)
}

// 認証系 Cookie をブラウザから削除する Set-Cookie を返す。
func (h *AuthHandler) clearCookie(c *gin.Context, name string) {
	// 同じ属性で期限切れ Cookie を返し、ブラウザ側の値を消す。
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.cookieName(name), "", -1, "/", h.cookieDomain(name), h.secureCookies(), true)
}

// 認証完了後の戻り先 URL を検証して Cookie に保存する。
func (h *AuthHandler) storeAuthRedirect(c *gin.Context) {
	// 認証完了後に戻す本体アプリの URL を検証して Cookie に保持する。
	h.setCookie(c, redirectCookieName, h.cfg.AuthRedirectURL(c.Query("redirect_to")), 1800, true)
}

// 保存済みの戻り先 URL を検証して返す。
func (h *AuthHandler) authRedirect(c *gin.Context) string {
	value, err := c.Cookie(h.cookieName(redirectCookieName))
	if err != nil {
		return h.cfg.DefaultRedirectURL
	}
	return h.cfg.AuthRedirectURL(value)
}

// 保存済み戻り先 URL を読み取って Cookie を削除する。
func (h *AuthHandler) consumeAuthRedirect(c *gin.Context) string {
	redirectURL := h.authRedirect(c)
	h.clearCookie(c, redirectCookieName)
	return redirectURL
}

// URL 文字列が https scheme で始まるかを返す。
func isHTTPS(rawURL string) bool {
	return strings.HasPrefix(rawURL, "https://")
}

// フロントエンド URL に合わせて Secure Cookie を使うかを返す。
func (h *AuthHandler) secureCookies() bool {
	return isHTTPS(h.cfg.FrontendURL)
}

// Secure Cookie で使う場合だけ __Host- 接頭辞付きの名前を返す。
func (h *AuthHandler) cookieName(name string) string {
	if !h.secureCookies() {
		return name
	}
	switch name {
	case sessionCookieName:
		if h.cfg.CookieDomain != "" {
			return sessionCookieName
		}
		return hostSessionCookieName
	case stateCookieName:
		return hostStateCookieName
	case redirectCookieName:
		return hostRedirectCookieName
	case passkeySessionCookieName:
		return hostPasskeySessionCookieName
	case oidcTransactionCookieName:
		if h.cfg.CookieDomain != "" {
			return oidcTransactionCookieName
		}
		return hostOIDCTransactionCookieName
	default:
		return name
	}
}

// ログインセッションだけを設定済みの親ドメインで共有する。
func (h *AuthHandler) cookieDomain(name string) string {
	if name == sessionCookieName {
		return h.cfg.CookieDomain
	}
	if name == oidcTransactionCookieName {
		return h.cfg.CookieDomain
	}
	return ""
}

package handler

import (
	"crypto/hmac"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-nextjs/src/infrastructure/session"
)

func (h *AuthHandler) validateState(c *gin.Context) error {
	// OAuth 開始時に保存した状態値とコールバックの状態値を比較する。
	expected, err := c.Cookie(stateCookieName)
	if err != nil {
		return err
	}
	if expected == "" || !hmac.Equal([]byte(expected), []byte(c.Query("state"))) {
		return errors.New("state mismatch")
	}
	return nil
}

func (h *AuthHandler) readSession(c *gin.Context) (session.User, error) {
	// ログイン Cookie を読み取り、署名検証済みのセッション情報へ戻す。
	value, err := c.Cookie(sessionCookieName)
	if err != nil {
		return session.User{}, err
	}
	return h.sessions.Verify(value)
}

func (h *AuthHandler) setCookie(c *gin.Context, name string, value string, maxAge int, httpOnly bool) {
	// フロントエンド URL が HTTPS のときだけ Secure 属性付き Cookie として送る。
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, value, maxAge, "/", "", isHTTPS(h.cfg.FrontendURL), httpOnly)
}

func (h *AuthHandler) clearCookie(c *gin.Context, name string) {
	// 同じ属性で期限切れ Cookie を返し、ブラウザ側の値を消す。
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, "", -1, "/", "", isHTTPS(h.cfg.FrontendURL), true)
}

func (h *AuthHandler) storeAuthRedirect(c *gin.Context) {
	// 認証完了後に戻す本体アプリの URL を検証して Cookie に保持する。
	h.setCookie(c, redirectCookieName, h.cfg.AuthRedirectURL(c.Query("redirect_to")), 1800, true)
}

func (h *AuthHandler) authRedirect(c *gin.Context) string {
	value, err := c.Cookie(redirectCookieName)
	if err != nil {
		return h.cfg.DefaultRedirectURL
	}
	return h.cfg.AuthRedirectURL(value)
}

func (h *AuthHandler) consumeAuthRedirect(c *gin.Context) string {
	redirectURL := h.authRedirect(c)
	h.clearCookie(c, redirectCookieName)
	return redirectURL
}

func isHTTPS(rawURL string) bool {
	return strings.HasPrefix(rawURL, "https://")
}

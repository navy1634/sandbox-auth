package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-nextjs/src/config"
	"github.com/sandbox-nextjs/src/infrastructure/session"
	"github.com/sandbox-nextjs/src/usecase"
)

const (
	sessionCookieName            = "app_session"
	stateCookieName              = "oauth_state"
	redirectCookieName           = "auth_redirect"
	passkeySessionCookieName     = "passkey_session"
	hostSessionCookieName        = "__Host-app_session"
	hostStateCookieName          = "__Host-oauth_state"
	hostRedirectCookieName       = "__Host-auth_redirect"
	hostPasskeySessionCookieName = "__Host-passkey_session"
)

type AuthHandler struct {
	cfg      config.Config
	accounts *usecase.AccountUsecase
	sessions *session.Manager
}

// 認証系 handler で共有する依存をまとめて初期化する。
func NewAuthHandler(cfg config.Config, accounts *usecase.AccountUsecase) *AuthHandler {
	// 認証系ハンドラで共有する設定、アカウント処理、Cookie セッション管理をまとめる。
	return &AuthHandler{
		cfg:      cfg,
		accounts: accounts,
		sessions: session.NewManager(cfg.SessionSecret),
	}
}

// API が応答できる状態かを返す。
func (h *AuthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ログインセッションを失効させて Cookie を削除する。
func (h *AuthHandler) Logout(c *gin.Context) {
	// ブラウザに残るログイン Cookie を削除してログアウト状態にする。
	if value, err := c.Cookie(h.cookieName(sessionCookieName)); err == nil {
		_ = h.sessions.Revoke(value)
	}
	h.clearCookie(c, sessionCookieName)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

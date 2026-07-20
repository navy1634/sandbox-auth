package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-nextjs/src/config"
	"github.com/sandbox-nextjs/src/infrastructure/session"
	"github.com/sandbox-nextjs/src/usecase"
)

const (
	sessionCookieName        = "app_session"
	stateCookieName          = "oauth_state"
	redirectCookieName       = "auth_redirect"
	passkeySessionCookieName = "passkey_session"
)

type AuthHandler struct {
	cfg      config.Config
	accounts *usecase.AccountUsecase
	sessions *session.Manager
}

func NewAuthHandler(cfg config.Config, accounts *usecase.AccountUsecase) *AuthHandler {
	// 認証系ハンドラで共有する設定、アカウント処理、Cookie セッション管理をまとめる。
	return &AuthHandler{
		cfg:      cfg,
		accounts: accounts,
		sessions: session.NewManager(cfg.SessionSecret),
	}
}

func (h *AuthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	// ブラウザに残るログイン Cookie を削除してログアウト状態にする。
	h.clearCookie(c, sessionCookieName)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

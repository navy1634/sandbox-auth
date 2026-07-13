package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-nextjs/src/config"
	"github.com/sandbox-nextjs/src/infrastructure/session"
	"github.com/sandbox-nextjs/src/repository"
)

const (
	sessionCookieName        = "app_session"
	stateCookieName          = "oauth_state"
	passkeySessionCookieName = "passkey_session"
	passkeyRegisterCeremony  = "passkey_register"
	passkeyLoginCeremony     = "passkey_login"
)

type AuthHandler struct {
	cfg      config.Config
	accounts repository.AccountRepository
	sessions *session.Manager
}

func NewAuthHandler(cfg config.Config, accounts repository.AccountRepository) *AuthHandler {
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
	h.clearCookie(c, sessionCookieName)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sandbox-nextjs/api/src/config"
	"sandbox-nextjs/api/src/infrastructure/auth"
	"sandbox-nextjs/api/src/infrastructure/session"
	"sandbox-nextjs/api/src/repository"
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
	accounts *repository.AccountRepository
	google   *auth.GoogleService
	passkey  *auth.PasskeyService
	sessions *session.Manager
}

func NewAuthHandler(cfg config.Config, accounts *repository.AccountRepository) (*AuthHandler, error) {
	passkey, err := auth.NewPasskeyService(cfg.PasskeyRPID, cfg.PasskeyRPOrigin)
	if err != nil {
		return nil, err
	}

	return &AuthHandler{
		cfg:      cfg,
		accounts: accounts,
		google:   auth.NewGoogleService(cfg.GoogleClientID, cfg.GoogleSecret, cfg.GoogleRedirectURL),
		passkey:  passkey,
		sessions: session.NewManager(cfg.SessionSecret),
	}, nil
}

func (h *AuthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	h.clearCookie(c, sessionCookieName)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/sandbox-nextjs/src/config"
	"github.com/sandbox-nextjs/src/domain"
	"github.com/sandbox-nextjs/src/infrastructure/auth"
	"github.com/sandbox-nextjs/src/infrastructure/session"
	"github.com/sandbox-nextjs/src/repository"
)

type PasskeyHandler struct {
	base     *AuthHandler
	passkeys repository.PasskeyRepository
	passkey  *auth.PasskeyService
}

func NewPasskeyHandler(cfg config.Config, base *AuthHandler, passkeys repository.PasskeyRepository) (*PasskeyHandler, error) {
	passkey, err := auth.NewPasskeyService(cfg.PasskeyRPID, cfg.PasskeyRPOrigin)
	if err != nil {
		return nil, err
	}

	return &PasskeyHandler{
		base:     base,
		passkeys: passkeys,
		passkey:  passkey,
	}, nil
}

func (h *PasskeyHandler) RegisterRoutes(routes gin.IRoutes) {
	routes.POST("/passkeys/register/options", h.BeginRegistration)
	routes.POST("/passkeys/register/verify", h.FinishRegistration)
	routes.POST("/passkeys/login/options", h.BeginLogin)
	routes.POST("/passkeys/login/verify", h.FinishLogin)
}

func (h *PasskeyHandler) BeginRegistration(c *gin.Context) {
	user, err := h.base.readSession(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	storedAccount, err := h.passkeys.FindWebAuthnUserByID(c.Request.Context(), user.AccountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load account"})
		return
	}

	creation, passkeySession, err := h.passkey.BeginRegistration(storedAccount)
	if err != nil {
		log.Printf("failed to begin passkey registration: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create passkey registration options"})
		return
	}

	sessionID, err := session.RandomString(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create passkey session"})
		return
	}

	accountID := storedAccount.ID
	if err := h.passkeys.SaveSession(c.Request.Context(), sessionID, &accountID, passkeyRegisterCeremony, passkeySession, 5*time.Minute); err != nil {
		log.Printf("failed to save passkey registration session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save passkey session"})
		return
	}

	h.base.setCookie(c, passkeySessionCookieName, sessionID, 300, true)
	c.JSON(http.StatusOK, creation)
}

func (h *PasskeyHandler) FinishRegistration(c *gin.Context) {
	user, err := h.base.readSession(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	sessionID, err := c.Cookie(passkeySessionCookieName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passkey session is missing"})
		return
	}

	passkeySession, err := h.passkeys.ConsumeSession(c.Request.Context(), sessionID, passkeyRegisterCeremony)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid passkey session"})
		return
	}
	if passkeySession.AccountID == nil || *passkeySession.AccountID != user.AccountID {
		c.JSON(http.StatusForbidden, gin.H{"error": "passkey session does not match account"})
		return
	}

	storedAccount, err := h.passkeys.FindWebAuthnUserByID(c.Request.Context(), user.AccountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load account"})
		return
	}

	credential, err := h.passkey.FinishRegistration(storedAccount, passkeySession.Session, c.Request)
	if err != nil {
		log.Printf("failed to verify passkey registration: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to verify passkey registration"})
		return
	}

	if err := h.passkeys.SaveCredential(c.Request.Context(), storedAccount.ID, credential); err != nil {
		log.Printf("failed to save passkey credential: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save passkey credential"})
		return
	}

	h.base.clearCookie(c, passkeySessionCookieName)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *PasskeyHandler) BeginLogin(c *gin.Context) {
	assertion, passkeySession, err := h.passkey.BeginLogin()
	if err != nil {
		log.Printf("failed to begin passkey login: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create passkey login options"})
		return
	}

	sessionID, err := session.RandomString(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create passkey session"})
		return
	}

	if err := h.passkeys.SaveSession(c.Request.Context(), sessionID, nil, passkeyLoginCeremony, passkeySession, 5*time.Minute); err != nil {
		log.Printf("failed to save passkey login session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save passkey session"})
		return
	}

	h.base.setCookie(c, passkeySessionCookieName, sessionID, 300, true)
	c.JSON(http.StatusOK, assertion)
}

func (h *PasskeyHandler) FinishLogin(c *gin.Context) {
	sessionID, err := c.Cookie(passkeySessionCookieName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passkey session is missing"})
		return
	}

	passkeySession, err := h.passkeys.ConsumeSession(c.Request.Context(), sessionID, passkeyLoginCeremony)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid passkey session"})
		return
	}

	validatedUser, validatedCredential, err := h.passkey.FinishLogin(func(rawID []byte, userHandle []byte) (webauthn.User, error) {
		return h.passkeys.FindWebAuthnUserByHandle(c.Request.Context(), userHandle)
	}, passkeySession.Session, c.Request)
	if err != nil {
		log.Printf("failed to verify passkey login: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to verify passkey login"})
		return
	}

	storedAccount, ok := validatedUser.(domain.Account)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load passkey account"})
		return
	}

	if err := h.passkeys.UpdateCredential(c.Request.Context(), storedAccount.ID, validatedCredential); err != nil {
		log.Printf("failed to update passkey credential: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update passkey credential"})
		return
	}

	sessionValue, err := h.base.sessions.Sign(session.FromAccount(storedAccount))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	h.base.clearCookie(c, passkeySessionCookieName)
	h.base.setCookie(c, sessionCookieName, sessionValue, 86400, true)
	h.base.respondAuthenticated(c, storedAccount)
}

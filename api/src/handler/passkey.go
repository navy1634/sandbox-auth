package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
	"sandbox-nextjs/api/src/domain"
	"sandbox-nextjs/api/src/infrastructure/session"
)

func (h *AuthHandler) BeginPasskeyRegistration(c *gin.Context) {
	user, err := h.readSession(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	storedAccount, err := h.accounts.FindWebAuthnUserByID(c.Request.Context(), user.AccountID)
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
	if err := h.accounts.SavePasskeySession(c.Request.Context(), sessionID, &accountID, passkeyRegisterCeremony, passkeySession, 5*time.Minute); err != nil {
		log.Printf("failed to save passkey registration session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save passkey session"})
		return
	}

	h.setCookie(c, passkeySessionCookieName, sessionID, 300, true)
	c.JSON(http.StatusOK, creation)
}

func (h *AuthHandler) FinishPasskeyRegistration(c *gin.Context) {
	user, err := h.readSession(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	sessionID, err := c.Cookie(passkeySessionCookieName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passkey session is missing"})
		return
	}

	passkeySession, err := h.accounts.ConsumePasskeySession(c.Request.Context(), sessionID, passkeyRegisterCeremony)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid passkey session"})
		return
	}
	if passkeySession.AccountID == nil || *passkeySession.AccountID != user.AccountID {
		c.JSON(http.StatusForbidden, gin.H{"error": "passkey session does not match account"})
		return
	}

	storedAccount, err := h.accounts.FindWebAuthnUserByID(c.Request.Context(), user.AccountID)
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

	if err := h.accounts.SaveCredential(c.Request.Context(), storedAccount.ID, credential); err != nil {
		log.Printf("failed to save passkey credential: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save passkey credential"})
		return
	}

	h.clearCookie(c, passkeySessionCookieName)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHandler) BeginPasskeyLogin(c *gin.Context) {
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

	if err := h.accounts.SavePasskeySession(c.Request.Context(), sessionID, nil, passkeyLoginCeremony, passkeySession, 5*time.Minute); err != nil {
		log.Printf("failed to save passkey login session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save passkey session"})
		return
	}

	h.setCookie(c, passkeySessionCookieName, sessionID, 300, true)
	c.JSON(http.StatusOK, assertion)
}

func (h *AuthHandler) FinishPasskeyLogin(c *gin.Context) {
	sessionID, err := c.Cookie(passkeySessionCookieName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passkey session is missing"})
		return
	}

	passkeySession, err := h.accounts.ConsumePasskeySession(c.Request.Context(), sessionID, passkeyLoginCeremony)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid passkey session"})
		return
	}

	validatedUser, validatedCredential, err := h.passkey.FinishLogin(func(rawID []byte, userHandle []byte) (webauthn.User, error) {
		return h.accounts.FindWebAuthnUserByHandle(c.Request.Context(), userHandle)
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

	if err := h.accounts.UpdateCredential(c.Request.Context(), storedAccount.ID, validatedCredential); err != nil {
		log.Printf("failed to update passkey credential: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update passkey credential"})
		return
	}

	sessionValue, err := h.sessions.Sign(session.FromAccount(storedAccount))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	h.clearCookie(c, passkeySessionCookieName)
	h.setCookie(c, sessionCookieName, sessionValue, 86400, true)
	h.respondAuthenticated(c, storedAccount)
}

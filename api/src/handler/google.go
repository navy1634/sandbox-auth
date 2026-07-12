package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"sandbox-nextjs/api/src/infrastructure/session"
)

func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	state, err := session.RandomString(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create oauth state"})
		return
	}

	h.setCookie(c, stateCookieName, state, 300, true)
	c.Redirect(http.StatusFound, h.google.AuthCodeURL(state))
}

func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	if oauthError := c.Query("error"); oauthError != "" {
		log.Printf("google oauth callback returned error: %s", oauthError)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "google oauth authorization failed"})
		return
	}

	if err := h.validateState(c); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth state"})
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "oauth code is missing"})
		return
	}

	googleUser, err := h.google.ExchangeAndValidate(c.Request.Context(), code)
	if err != nil {
		log.Printf("google oauth code exchange failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to exchange oauth code"})
		return
	}

	storedAccount, err := h.accounts.UpsertGoogleAccount(c.Request.Context(), googleUser)
	if err != nil {
		log.Printf("failed to save google account: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save account"})
		return
	}

	sessionValue, err := h.sessions.Sign(session.FromAccount(storedAccount))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	h.clearCookie(c, stateCookieName)
	h.setCookie(c, sessionCookieName, sessionValue, 86400, true)
	if storedAccount.RegisteredAt == nil {
		c.Redirect(http.StatusFound, h.cfg.FrontendURL+"/register")
		return
	}

	c.Redirect(http.StatusFound, h.cfg.FrontendURL+"/mypage")
}

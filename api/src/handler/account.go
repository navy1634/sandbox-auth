package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-nextjs/src/domain"
	"github.com/sandbox-nextjs/src/infrastructure/session"
	"github.com/sandbox-nextjs/src/repository"
)

func (h *AuthHandler) Me(c *gin.Context) {
	user, err := h.readSession(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"authenticated": false})
		return
	}

	storedAccount, err := h.accounts.Me(c.Request.Context(), user.AccountID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
			return
		}

		log.Printf("failed to load account: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load account"})
		return
	}

	h.respondAuthenticated(c, storedAccount)
}

func (h *AuthHandler) respondAuthenticated(c *gin.Context, storedAccount domain.Account) {
	c.JSON(http.StatusOK, gin.H{
		"account":       storedAccount,
		"authenticated": true,
		"redirectTo":    h.cfg.AuthRedirectURL(c.Query("redirect_to")),
		"user":          session.FromAccount(storedAccount),
	})
}

func (h *AuthHandler) respondAuthenticatedWithRedirect(c *gin.Context, storedAccount domain.Account) {
	c.JSON(http.StatusOK, gin.H{
		"account":       storedAccount,
		"authenticated": true,
		"redirectTo":    h.consumeAuthRedirect(c),
		"user":          session.FromAccount(storedAccount),
	})
}

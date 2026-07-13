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

	storedAccount, err := h.accounts.FindByID(c.Request.Context(), user.AccountID)
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

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	user, err := h.readSession(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var input domain.ProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	input = input.Normalize()
	if err := input.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	storedAccount, err := h.accounts.UpdateProfile(c.Request.Context(), user.AccountID, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.respondAuthenticated(c, storedAccount)
}

func (h *AuthHandler) respondAuthenticated(c *gin.Context, storedAccount domain.Account) {
	c.JSON(http.StatusOK, gin.H{
		"account":           storedAccount,
		"authenticated":     true,
		"needsRegistration": storedAccount.RegisteredAt == nil,
		"user":              session.FromAccount(storedAccount),
	})
}

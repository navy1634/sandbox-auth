package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-auth/src/domain"
	"github.com/sandbox-auth/src/infrastructure/session"
	"github.com/sandbox-auth/src/repository"
)

// 現在のログインセッションに対応するアカウント情報を返す。
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

	h.respondAuthenticated(c, storedAccount, h.cfg.AuthRedirectURL(c.Query("redirect_to")))
}

// 認証済みレスポンスを指定した戻り先 URL と合わせて返す。
func (h *AuthHandler) respondAuthenticated(c *gin.Context, storedAccount domain.Account, redirectTo string) {
	c.JSON(http.StatusOK, gin.H{
		"account":       storedAccount,
		"authenticated": true,
		"redirectTo":    redirectTo,
		"user":          session.FromAccount(storedAccount),
	})
}

// 保存済み戻り先を消費して認証済みレスポンスを返す。
func (h *AuthHandler) respondAuthenticatedWithRedirect(c *gin.Context, storedAccount domain.Account) {
	h.respondAuthenticated(c, storedAccount, h.consumeAuthRedirect(c))
}

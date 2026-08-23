package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-nextjs/src/infrastructure/session"
	"github.com/sandbox-nextjs/src/usecase"
)

type OAuthHandler struct {
	base  *AuthHandler
	oauth *usecase.OAuthUsecase
}

// OAuth 用の HTTP handler を作る。
func NewOAuthHandler(base *AuthHandler, oauth *usecase.OAuthUsecase) *OAuthHandler {
	return &OAuthHandler{
		base:  base,
		oauth: oauth,
	}
}

// OAuth login と callback のルートを登録する。
func (h *OAuthHandler) RegisterRoutes(routes gin.IRoutes) {
	routes.GET("/auth/:provider/login", h.Login)
	routes.GET("/auth/:provider/callback", h.Callback)
}

// OAuth 認可を開始して provider の認可 URL へリダイレクトする。
func (h *OAuthHandler) Login(c *gin.Context) {
	// OAuth 認可を開始し、状態値を Cookie に保持してから認証画面へ移動する。
	state, authURL, err := h.oauth.BeginLogin(c.Param("provider"))
	if errors.Is(err, usecase.ErrAuthProviderNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "auth provider not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create oauth state"})
		return
	}

	h.base.storeAuthRedirect(c)
	h.base.setCookie(c, stateCookieName, state, 300, true)
	c.Redirect(http.StatusFound, authURL)
}

// OAuth callback を検証してアプリのログインセッションを発行する。
func (h *OAuthHandler) Callback(c *gin.Context) {
	providerName := c.Param("provider")
	if oauthError := c.Query("error"); oauthError != "" {
		// OAuth error は quoted string としてログに残す。
		log.Printf("%q oauth callback returned error: %q", providerName, oauthError)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "oauth authorization failed"})
		return
	}

	if err := h.base.validateState(c); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth state"})
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "oauth code is missing"})
		return
	}

	// OAuth コールバックを検証し、アプリのログインセッションを発行する。
	storedAccount, err := h.oauth.Callback(c.Request.Context(), providerName, code)
	if errors.Is(err, usecase.ErrAuthProviderNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "auth provider not found"})
		return
	}
	if errors.Is(err, usecase.ErrOAuthExchangeFailed) {
		log.Printf("%s oauth code exchange failed: %v", providerName, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to exchange oauth code"})
		return
	}
	if err != nil {
		log.Printf("failed to save provider identity: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save account"})
		return
	}

	sessionValue, err := h.base.sessions.SignContext(c.Request.Context(), session.FromAccount(storedAccount))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	h.base.clearCookie(c, stateCookieName)
	h.base.setCookie(c, sessionCookieName, sessionValue, 86400, true)
	c.Redirect(http.StatusFound, h.base.consumeAuthRedirect(c))
}

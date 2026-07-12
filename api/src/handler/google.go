package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-nextjs/src/config"
	"github.com/sandbox-nextjs/src/infrastructure/auth"
	"github.com/sandbox-nextjs/src/infrastructure/session"
)

type OAuthHandler struct {
	base      *AuthHandler
	providers map[string]auth.OAuthProvider
}

func NewOAuthHandler(cfg config.Config, base *AuthHandler) *OAuthHandler {
	return &OAuthHandler{
		base: base,
		providers: map[string]auth.OAuthProvider{
			"google": auth.NewGoogleProvider(cfg.GoogleClientID, cfg.GoogleSecret, cfg.GoogleRedirectURL),
		},
	}
}

func (h *OAuthHandler) RegisterRoutes(routes gin.IRoutes) {
	routes.GET("/auth/:provider/login", h.Login)
	routes.GET("/auth/:provider/callback", h.Callback)
}

func (h *OAuthHandler) Login(c *gin.Context) {
	provider, ok := h.providers[c.Param("provider")]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "auth provider not found"})
		return
	}

	state, err := session.RandomString(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create oauth state"})
		return
	}

	h.base.setCookie(c, stateCookieName, state, 300, true)
	c.Redirect(http.StatusFound, provider.AuthCodeURL(state))
}

func (h *OAuthHandler) Callback(c *gin.Context) {
	providerName := c.Param("provider")
	provider, ok := h.providers[providerName]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "auth provider not found"})
		return
	}

	if oauthError := c.Query("error"); oauthError != "" {
		log.Printf("%s oauth callback returned error: %s", providerName, oauthError)
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

	identity, err := provider.ExchangeAndValidate(c.Request.Context(), code)
	if err != nil {
		log.Printf("%s oauth code exchange failed: %v", providerName, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to exchange oauth code"})
		return
	}

	storedAccount, err := h.base.accounts.UpsertProviderIdentity(c.Request.Context(), identity)
	if err != nil {
		log.Printf("failed to save provider identity: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save account"})
		return
	}

	sessionValue, err := h.base.sessions.Sign(session.FromAccount(storedAccount))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	h.base.clearCookie(c, stateCookieName)
	h.base.setCookie(c, sessionCookieName, sessionValue, 86400, true)
	if storedAccount.RegisteredAt == nil {
		c.Redirect(http.StatusFound, h.base.cfg.FrontendURL+"/register")
		return
	}

	c.Redirect(http.StatusFound, h.base.cfg.FrontendURL+"/mypage")
}

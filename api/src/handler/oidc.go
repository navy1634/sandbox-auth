package handler

import (
	"crypto/hmac"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-auth/src/config"
	"github.com/sandbox-auth/src/domain"
	"github.com/sandbox-auth/src/infrastructure/oidc"
	"github.com/sandbox-auth/src/infrastructure/session"
	"github.com/sandbox-auth/src/repository"
	"github.com/sandbox-auth/src/usecase"
)

type OIDCHandler struct {
	base     *AuthHandler
	accounts *usecase.AccountUsecase
	provider *oidc.Provider
	cfg      config.Config
}

// OIDC Provider用のHTTP handlerを作る。
func NewOIDCHandler(base *AuthHandler, accounts *usecase.AccountUsecase, provider *oidc.Provider) *OIDCHandler {
	return &OIDCHandler{
		base:     base,
		accounts: accounts,
		provider: provider,
		cfg:      base.cfg,
	}
}

// OIDC Providerの標準エンドポイントを登録する。
func (h *OIDCHandler) RegisterRoutes(routes gin.IRoutes) {
	routes.GET("/.well-known/openid-configuration", h.Discovery)
	routes.GET("/.well-known/jwks.json", h.JWKS)
	routes.GET("/oidc/authorize", h.Authorize)
	routes.GET("/oidc/authorize/resume", h.ResumeAuthorization)
	routes.POST("/oidc/token", h.Token)
	routes.GET("/oidc/userinfo", h.UserInfo)
	routes.POST("/oidc/userinfo", h.UserInfo)
}

// OIDC Discovery文書を返す。
func (h *OIDCHandler) Discovery(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=300")
	c.JSON(http.StatusOK, h.provider.Discovery())
}

// OIDC署名検証用の公開鍵を返す。
func (h *OIDCHandler) JWKS(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=300")
	c.JSON(http.StatusOK, h.provider.JWKS())
}

// OIDC認可要求を検証し、ログイン済みなら認可コードを返す。
func (h *OIDCHandler) Authorize(c *gin.Context) {
	request, err := h.provider.ParseAuthorizationRequestContext(c.Request.Context(), c.Request.URL.Query())
	if err != nil {
		h.respondAuthorizationError(c, err)
		return
	}

	user, err := h.base.readSession(c)
	if err == nil && !request.PromptLogin {
		h.authorizeForUser(c, request, user)
		return
	}
	if request.PromptNone {
		h.respondAuthorizationError(c, &oidc.AuthorizationError{
			Code:        "login_required",
			Description: "user authentication is required",
			RedirectURI: request.RedirectURI,
			State:       request.State,
		})
		return
	}

	transactionID, err := h.provider.CreateTransactionContext(c.Request.Context(), request)
	if err != nil {
		log.Printf("failed to create OIDC transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create authorization request"})
		return
	}
	h.base.setCookie(c, oidcTransactionCookieName, transactionID, 120, true)
	resumeURL := h.cfg.OIDCResumeURL(transactionID)
	loginURL := h.cfg.OIDCLoginURL(resumeURL)
	if loginURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OIDC login URL is not configured"})
		return
	}
	c.Redirect(http.StatusFound, loginURL)
}

// ログイン済みセッションで保留中のOIDC認可要求を再開する。
func (h *OIDCHandler) ResumeAuthorization(c *gin.Context) {
	user, err := h.base.readSession(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	transactionID := c.Query("transaction")
	cookieTransactionID, cookieErr := c.Cookie(h.base.cookieName(oidcTransactionCookieName))
	if cookieErr != nil || cookieTransactionID == "" || !hmac.Equal([]byte(cookieTransactionID), []byte(transactionID)) {
		h.base.clearCookie(c, oidcTransactionCookieName)
		c.JSON(http.StatusBadRequest, gin.H{"error": "authorization request is not bound to this browser"})
		return
	}
	h.base.clearCookie(c, oidcTransactionCookieName)
	request, err := h.provider.ConsumeTransactionContext(c.Request.Context(), transactionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "authorization request is expired or invalid"})
		return
	}
	h.authorizeForUser(c, request, user)
}

// ログイン済みアカウントに対する認可コードを発行する。
func (h *OIDCHandler) authorizeForUser(c *gin.Context, request oidc.AuthorizationRequest, user session.User) {
	account, err := h.accounts.Me(c.Request.Context(), user.AccountID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.respondAuthorizationError(c, &oidc.AuthorizationError{
				Code:        "access_denied",
				Description: "account is not available",
				RedirectURI: request.RedirectURI,
				State:       request.State,
			})
			return
		}
		log.Printf("failed to load OIDC account: %v", err)
		h.respondAuthorizationError(c, &oidc.AuthorizationError{
			Code:        "server_error",
			Description: "failed to load account",
			RedirectURI: request.RedirectURI,
			State:       request.State,
		})
		return
	}
	code, err := h.provider.IssueAuthorizationCodeContext(c.Request.Context(), request, account.ID)
	if err != nil {
		log.Printf("failed to issue OIDC authorization code: %v", err)
		h.respondAuthorizationError(c, &oidc.AuthorizationError{
			Code:        "server_error",
			Description: "failed to issue authorization code",
			RedirectURI: request.RedirectURI,
			State:       request.State,
		})
		return
	}

	redirectURL, err := url.Parse(request.RedirectURI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid configured redirect URI"})
		return
	}
	query := redirectURL.Query()
	query.Set("code", code)
	if request.State != "" {
		query.Set("state", request.State)
	}
	redirectURL.RawQuery = query.Encode()
	c.Redirect(http.StatusFound, redirectURL.String())
}

// 認可コードをアクセストークンとIDトークンへ交換する。
func (h *OIDCHandler) Token(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	if err := c.Request.ParseForm(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	if c.PostForm("grant_type") != "authorization_code" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
		return
	}
	clientID, clientSecret, ok := tokenClientCredentials(c)
	if !ok || !h.provider.AuthenticateClientContext(c.Request.Context(), clientID, clientSecret) {
		c.Header("WWW-Authenticate", `Basic realm="oidc"`)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}
	code := c.PostForm("code")
	redirectURI := c.PostForm("redirect_uri")
	codeVerifier := c.PostForm("code_verifier")
	data, err := h.provider.RedeemAuthorizationCodeContext(c.Request.Context(), code, clientID, redirectURI, codeVerifier)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant"})
		return
	}
	account, err := h.accounts.Me(c.Request.Context(), data.AccountID)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			log.Printf("failed to load OIDC account for token: %v", err)
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant"})
		return
	}
	tokens, err := h.provider.IssueTokenSetContext(c.Request.Context(), data, oidcUser(account))
	if err != nil {
		log.Printf("failed to issue OIDC tokens: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.JSON(http.StatusOK, tokens)
}

// BearerアクセストークンからUserInfoを返す。
func (h *OIDCHandler) UserInfo(c *gin.Context) {
	token, ok := bearerToken(c.GetHeader("Authorization"))
	if !ok {
		h.respondInvalidBearer(c)
		return
	}
	data, err := h.provider.ValidateAccessTokenContext(c.Request.Context(), token)
	if err != nil {
		h.respondInvalidBearer(c)
		return
	}
	account, err := h.accounts.Me(c.Request.Context(), data.AccountID)
	if err != nil {
		h.respondInvalidBearer(c)
		return
	}
	c.JSON(http.StatusOK, h.provider.UserInfoClaims(data, oidcUser(account)))
}

// Token endpointのBasic認証またはPOST認証情報を読み取る。
func tokenClientCredentials(c *gin.Context) (string, string, bool) {
	formClientID := c.PostForm("client_id")
	formClientSecret := c.PostForm("client_secret")
	basicClientID, basicClientSecret, basicOK := c.Request.BasicAuth()
	if basicOK {
		if formClientID != "" && formClientID != basicClientID {
			return "", "", false
		}
		return basicClientID, basicClientSecret, basicClientID != ""
	}
	return formClientID, formClientSecret, formClientID != ""
}

// AuthorizationヘッダーからBearerトークンを取り出す。
func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

// OIDCアカウント情報を発行用のユーザー型へ変換する。
func oidcUser(account domain.Account) oidc.User {
	user := oidc.User{AccountID: account.ID, Subject: account.Subject}
	if account.Identity != nil {
		user.Email = account.Identity.Email
		user.EmailVerified = account.Identity.EmailVerified
	}
	return user
}

// Authorization endpointのエラーを安全な場合だけクライアントへ返す。
func (h *OIDCHandler) respondAuthorizationError(c *gin.Context, err error) {
	var authorizationError *oidc.AuthorizationError
	if !errors.As(err, &authorizationError) || authorizationError.RedirectURI == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	redirectURL, parseErr := url.Parse(authorizationError.RedirectURI)
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	query := redirectURL.Query()
	query.Set("error", authorizationError.Code)
	if authorizationError.Description != "" {
		query.Set("error_description", authorizationError.Description)
	}
	if authorizationError.State != "" {
		query.Set("state", authorizationError.State)
	}
	redirectURL.RawQuery = query.Encode()
	c.Redirect(http.StatusFound, redirectURL.String())
}

func (h *OIDCHandler) respondInvalidBearer(c *gin.Context) {
	c.Header("WWW-Authenticate", `Bearer error="invalid_token"`)
	c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
}

package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-auth/src/infrastructure/session"
	"github.com/sandbox-auth/src/usecase"
)

type PasskeyHandler struct {
	base     *AuthHandler
	passkeys *usecase.PasskeyUsecase
}

// パスキー用の HTTP handler を作る。
func NewPasskeyHandler(base *AuthHandler, passkeys *usecase.PasskeyUsecase) *PasskeyHandler {
	return &PasskeyHandler{
		base:     base,
		passkeys: passkeys,
	}
}

// パスキー登録とログインのルートを登録する。
func (h *PasskeyHandler) RegisterRoutes(routes gin.IRoutes) {
	routes.POST("/passkeys/register/options", h.BeginRegistration)
	routes.POST("/passkeys/register/verify", h.FinishRegistration)
	routes.POST("/passkeys/login/options", h.BeginLogin)
	routes.POST("/passkeys/login/verify", h.FinishLogin)
}

// ログイン中のアカウント用にパスキー登録 options を発行する。
func (h *PasskeyHandler) BeginRegistration(c *gin.Context) {
	user, err := h.base.readSession(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// ログイン中のアカウントにパスキー登録を開始させる。
	options, err := h.passkeys.BeginRegistration(c.Request.Context(), user.AccountID)
	if errors.Is(err, usecase.ErrPasskeyLoadAccountFailed) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load account"})
		return
	}
	if errors.Is(err, usecase.ErrPasskeySessionLimitExceeded) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many passkey sessions"})
		return
	}
	if errors.Is(err, usecase.ErrPasskeySaveSessionFailed) {
		log.Printf("failed to save passkey registration session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save passkey session"})
		return
	}
	if err != nil {
		log.Printf("failed to begin passkey registration: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create passkey registration options"})
		return
	}

	h.base.setCookie(c, passkeySessionCookieName, options.SessionID, 300, true)
	c.JSON(http.StatusOK, options.Creation)
}

// 認証器から返った登録応答を検証して credential を保存する。
func (h *PasskeyHandler) FinishRegistration(c *gin.Context) {
	user, err := h.base.readSession(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	sessionID, err := c.Cookie(h.base.cookieName(passkeySessionCookieName))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passkey session is missing"})
		return
	}

	// 登録開始時のセッションを使って、認証器からの登録応答を検証する。
	err = h.passkeys.FinishRegistration(c.Request.Context(), user.AccountID, sessionID, c.Request)
	if errors.Is(err, usecase.ErrPasskeySessionAccountMismatch) {
		c.JSON(http.StatusForbidden, gin.H{"error": "passkey session does not match account"})
		return
	}
	if errors.Is(err, usecase.ErrPasskeySessionInvalid) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid passkey session"})
		return
	}
	if errors.Is(err, usecase.ErrPasskeyLoadAccountFailed) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load account"})
		return
	}
	if errors.Is(err, usecase.ErrPasskeyCredentialExists) {
		c.JSON(http.StatusConflict, gin.H{"error": "passkey credential already exists"})
		return
	}
	if errors.Is(err, usecase.ErrPasskeySaveCredentialFailed) {
		log.Printf("failed to save passkey credential: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save passkey credential"})
		return
	}
	if err != nil {
		log.Printf("failed to verify passkey registration: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to verify passkey registration"})
		return
	}

	h.base.clearCookie(c, passkeySessionCookieName)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// パスキーによる discoverable login の options を発行する。
func (h *PasskeyHandler) BeginLogin(c *gin.Context) {
	// パスキーによるログインを開始し、検証用セッションを Cookie に保持する。
	options, err := h.passkeys.BeginLogin(c.Request.Context())
	if errors.Is(err, usecase.ErrPasskeySessionLimitExceeded) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many passkey sessions"})
		return
	}
	if errors.Is(err, usecase.ErrPasskeySaveSessionFailed) {
		log.Printf("failed to save passkey login session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save passkey session"})
		return
	}
	if err != nil {
		log.Printf("failed to begin passkey login: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create passkey login options"})
		return
	}

	h.base.storeAuthRedirect(c)
	h.base.setCookie(c, passkeySessionCookieName, options.SessionID, 300, true)
	c.JSON(http.StatusOK, options.Assertion)
}

// 認証器から返ったログイン応答を検証してログインセッションを発行する。
func (h *PasskeyHandler) FinishLogin(c *gin.Context) {
	sessionID, err := c.Cookie(h.base.cookieName(passkeySessionCookieName))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passkey session is missing"})
		return
	}

	// ログイン開始時のセッションを使って、認証器からのログイン応答を検証する。
	storedAccount, err := h.passkeys.FinishLogin(c.Request.Context(), sessionID, c.Request)
	if errors.Is(err, usecase.ErrPasskeySessionInvalid) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid passkey session"})
		return
	}
	if errors.Is(err, usecase.ErrInvalidPasskeyUser) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load passkey account"})
		return
	}
	if errors.Is(err, usecase.ErrPasskeyUpdateCredentialFailed) {
		log.Printf("failed to update passkey credential: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update passkey credential"})
		return
	}
	if err != nil {
		log.Printf("failed to verify passkey login: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to verify passkey login"})
		return
	}

	sessionValue, err := h.base.sessions.SignContext(c.Request.Context(), session.FromAccount(storedAccount))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	h.base.clearCookie(c, passkeySessionCookieName)
	h.base.setCookie(c, sessionCookieName, sessionValue, 86400, true)
	h.base.respondAuthenticatedWithRedirect(c, storedAccount)
}

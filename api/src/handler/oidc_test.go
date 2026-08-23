package handler

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-nextjs/src/config"
	"github.com/sandbox-nextjs/src/domain"
	"github.com/sandbox-nextjs/src/infrastructure/oidc"
	"github.com/sandbox-nextjs/src/infrastructure/session"
	"github.com/sandbox-nextjs/src/repository"
	"github.com/sandbox-nextjs/src/usecase"
)

func TestOIDCProviderSupportsAuthorizationCodeFlow(t *testing.T) {
	handler := newOIDCTestHandler(t)
	engine := gin.New()
	handler.RegisterRoutes(engine.Group("/api"))

	verifier := "verifier"
	query := url.Values{
		"client_id":             []string{"client"},
		"redirect_uri":          []string{"https://app.example.com/callback"},
		"response_type":         []string{"code"},
		"scope":                 []string{"openid email"},
		"state":                 []string{"state-value"},
		"nonce":                 []string{"nonce-value"},
		"code_challenge":        []string{testCodeChallenge(verifier)},
		"code_challenge_method": []string{"S256"},
	}
	sessionValue, err := handler.base.sessions.Sign(session.User{AccountID: 1})
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	authorizeRequest := httptest.NewRequest(http.MethodGet, "/api/oidc/authorize?"+query.Encode(), nil)
	authorizeRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionValue})
	authorizeResponse := httptest.NewRecorder()
	engine.ServeHTTP(authorizeResponse, authorizeRequest)
	if authorizeResponse.Code != http.StatusFound {
		t.Fatalf("authorize status = %d, want %d", authorizeResponse.Code, http.StatusFound)
	}

	redirectURL, err := url.Parse(authorizeResponse.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse authorization redirect: %v", err)
	}
	if redirectURL.Query().Get("state") != "state-value" || redirectURL.Query().Get("code") == "" {
		t.Fatalf("unexpected authorization redirect: %s", redirectURL)
	}

	tokenForm := url.Values{
		"grant_type":    []string{"authorization_code"},
		"code":          []string{redirectURL.Query().Get("code")},
		"client_id":     []string{"client"},
		"client_secret": []string{"client-secret"},
		"redirect_uri":  []string{"https://app.example.com/callback"},
		"code_verifier": []string{verifier},
	}
	tokenRequest := httptest.NewRequest(http.MethodPost, "/api/oidc/token", strings.NewReader(tokenForm.Encode()))
	tokenRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenResponse := httptest.NewRecorder()
	engine.ServeHTTP(tokenResponse, tokenRequest)
	if tokenResponse.Code != http.StatusOK {
		body, _ := io.ReadAll(tokenResponse.Body)
		t.Fatalf("token status = %d, body = %s", tokenResponse.Code, body)
	}

	var tokens struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.Unmarshal(tokenResponse.Body.Bytes(), &tokens); err != nil {
		t.Fatalf("decode token response: %v", err)
	}
	if tokens.AccessToken == "" || tokens.IDToken == "" {
		t.Fatalf("unexpected token response: %+v", tokens)
	}

	userinfoRequest := httptest.NewRequest(http.MethodGet, "/api/oidc/userinfo", nil)
	userinfoRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	userinfoResponse := httptest.NewRecorder()
	engine.ServeHTTP(userinfoResponse, userinfoRequest)
	if userinfoResponse.Code != http.StatusOK {
		t.Fatalf("userinfo status = %d, want %d", userinfoResponse.Code, http.StatusOK)
	}
	var userinfo map[string]any
	if err := json.Unmarshal(userinfoResponse.Body.Bytes(), &userinfo); err != nil {
		t.Fatalf("decode userinfo response: %v", err)
	}
	if userinfo["email"] != "user@example.com" || userinfo["email_verified"] != true {
		t.Fatalf("unexpected userinfo: %#v", userinfo)
	}

	replayResponse := httptest.NewRecorder()
	engine.ServeHTTP(replayResponse, tokenRequest)
	if replayResponse.Code != http.StatusBadRequest {
		t.Fatalf("replay status = %d, want %d", replayResponse.Code, http.StatusBadRequest)
	}
}

func TestOIDCAuthorizeRedirectsUnauthenticatedUserToLogin(t *testing.T) {
	handler := newOIDCTestHandler(t)
	engine := gin.New()
	handler.RegisterRoutes(engine.Group("/api"))

	query := url.Values{
		"client_id":             []string{"client"},
		"redirect_uri":          []string{"https://app.example.com/callback"},
		"response_type":         []string{"code"},
		"scope":                 []string{"openid"},
		"nonce":                 []string{"nonce-value"},
		"code_challenge":        []string{testCodeChallenge("verifier")},
		"code_challenge_method": []string{"S256"},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/oidc/authorize?"+query.Encode(), nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusFound)
	}

	loginURL, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse login redirect: %v", err)
	}
	if loginURL.Path != "/login" || loginURL.Query().Get("redirect_to") == "" {
		t.Fatalf("unexpected login redirect: %s", loginURL)
	}
	if !handler.cfg.IsOIDCResumeURL(loginURL.Query().Get("redirect_to")) {
		t.Fatalf("redirect_to is not an OIDC resume URL: %s", loginURL.Query().Get("redirect_to"))
	}
}

func TestOIDCAuthorizeRejectsUnregisteredRedirectURI(t *testing.T) {
	handler := newOIDCTestHandler(t)
	engine := gin.New()
	handler.RegisterRoutes(engine.Group("/api"))

	query := url.Values{
		"client_id":             []string{"client"},
		"redirect_uri":          []string{"https://evil.example.com/callback"},
		"response_type":         []string{"code"},
		"scope":                 []string{"openid"},
		"nonce":                 []string{"nonce-value"},
		"code_challenge":        []string{testCodeChallenge("verifier")},
		"code_challenge_method": []string{"S256"},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/oidc/authorize?"+query.Encode(), nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if response.Header().Get("Location") != "" {
		t.Fatalf("unexpected redirect location: %s", response.Header().Get("Location"))
	}
}

func newOIDCTestHandler(t *testing.T) *OIDCHandler {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	cfg := config.Config{
		FrontendURL:       "http://localhost:3000",
		OIDCIssuerURL:     "http://localhost:8080/api",
		OIDCClients:       []config.OIDCClient{{ClientID: "client", ClientSecret: "client-secret", RedirectURIs: []string{"https://app.example.com/callback"}}},
		OIDCSigningKeyPEM: keyPEM,
	}
	repository := &oidcTestAccountRepository{account: domain.Account{
		ID:      1,
		Subject: "subject-1",
		Identity: &domain.ProviderIdentity{
			Provider:          "google",
			ProviderAccountID: "google-account",
			Email:             "user@example.com",
			EmailVerified:     true,
		},
	}}
	accounts := usecase.NewAccountUsecase(repository)
	base := NewAuthHandler(cfg, accounts)
	provider, err := oidc.NewProvider(oidc.Config{
		IssuerURL: cfg.OIDCIssuerURL,
		Clients: []oidc.Client{{
			ID:           "client",
			Secret:       "client-secret",
			RedirectURIs: []string{"https://app.example.com/callback"},
		}},
		SigningKeyPEM: keyPEM,
	})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	return NewOIDCHandler(base, accounts, provider)
}

type oidcTestAccountRepository struct {
	account domain.Account
}

func (r *oidcTestAccountRepository) UpsertProviderIdentity(_ context.Context, _ domain.ProviderIdentity) (domain.Account, error) {
	return r.account, nil
}

func (r *oidcTestAccountRepository) FindByID(_ context.Context, id int64) (domain.Account, error) {
	if id != r.account.ID {
		return domain.Account{}, repository.ErrNotFound
	}
	return r.account, nil
}

func (r *oidcTestAccountRepository) FindByWebAuthnUserHandle(_ context.Context, _ []byte) (domain.Account, error) {
	return r.account, nil
}

func testCodeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

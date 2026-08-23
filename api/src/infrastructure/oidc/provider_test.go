package oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestParseAuthorizationRequestRequiresPKCEAndNonce(t *testing.T) {
	provider := newTestProvider(t)
	challenge := codeChallenge("verifier")

	request, err := provider.ParseAuthorizationRequest(url.Values{
		"client_id":             []string{"client"},
		"redirect_uri":          []string{"https://app.example.com/callback"},
		"response_type":         []string{"code"},
		"scope":                 []string{"openid email"},
		"state":                 []string{"state-value"},
		"nonce":                 []string{"nonce-value"},
		"code_challenge":        []string{challenge},
		"code_challenge_method": []string{"S256"},
	})
	if err != nil {
		t.Fatalf("ParseAuthorizationRequest() error = %v", err)
	}
	if request.ClientID != "client" || request.State != "state-value" {
		t.Fatalf("unexpected authorization request: %+v", request)
	}

	_, err = provider.ParseAuthorizationRequest(url.Values{
		"client_id":     []string{"client"},
		"redirect_uri":  []string{"https://app.example.com/callback"},
		"response_type": []string{"code"},
		"scope":         []string{"openid"},
	})
	if err == nil {
		t.Fatal("expected missing PKCE and nonce to be rejected")
	}
}

func TestAuthorizationCodeCanOnlyBeRedeemedOnce(t *testing.T) {
	provider := newTestProvider(t)
	request, err := provider.ParseAuthorizationRequest(validAuthorizationValues())
	if err != nil {
		t.Fatalf("ParseAuthorizationRequest() error = %v", err)
	}

	code, err := provider.IssueAuthorizationCode(request, 42)
	if err != nil {
		t.Fatalf("IssueAuthorizationCode() error = %v", err)
	}

	data, err := provider.RedeemAuthorizationCode(code, "client", request.RedirectURI, "verifier")
	if err != nil {
		t.Fatalf("RedeemAuthorizationCode() error = %v", err)
	}
	if data.AccountID != 42 {
		t.Fatalf("AccountID = %d, want 42", data.AccountID)
	}

	if _, err := provider.RedeemAuthorizationCode(code, "client", request.RedirectURI, "verifier"); err == nil {
		t.Fatal("expected authorization code replay to be rejected")
	}
}

func TestIssueTokenSetIncludesOIDCClaimsAndJWKS(t *testing.T) {
	provider := newTestProvider(t)
	request, err := provider.ParseAuthorizationRequest(validAuthorizationValues())
	if err != nil {
		t.Fatalf("ParseAuthorizationRequest() error = %v", err)
	}
	code, err := provider.IssueAuthorizationCode(request, 42)
	if err != nil {
		t.Fatalf("IssueAuthorizationCode() error = %v", err)
	}
	data, err := provider.RedeemAuthorizationCode(code, "client", request.RedirectURI, "verifier")
	if err != nil {
		t.Fatalf("RedeemAuthorizationCode() error = %v", err)
	}

	tokens, err := provider.IssueTokenSet(data, User{AccountID: 42, Subject: "subject-42", Email: "user@example.com", EmailVerified: true})
	if err != nil {
		t.Fatalf("IssueTokenSet() error = %v", err)
	}
	parts := strings.Split(tokens.IDToken, ".")
	if len(parts) != 3 {
		t.Fatalf("ID token parts = %d, want 3", len(parts))
	}
	if tokens.AccessToken == "" || tokens.TokenType != "Bearer" {
		t.Fatalf("unexpected token set: %+v", tokens)
	}

	jwks := provider.JWKS()
	keys, ok := jwks["keys"].([]map[string]any)
	if !ok || len(keys) != 1 {
		t.Fatalf("unexpected JWKS: %#v", jwks)
	}
	if keys[0]["kty"] != "RSA" || keys[0]["alg"] != "RS256" {
		t.Fatalf("unexpected JWK: %#v", keys[0])
	}
}

func newTestProvider(t *testing.T) *Provider {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	provider, err := NewProvider(Config{
		IssuerURL:     "https://auth.example.com/api",
		Clients:       []Client{{ID: "client", Secret: "client-secret", RedirectURIs: []string{"https://app.example.com/callback"}}},
		SigningKeyPEM: keyPEM,
		CodeTTL:       time.Minute,
	})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	return provider
}

func validAuthorizationValues() url.Values {
	return url.Values{
		"client_id":             []string{"client"},
		"redirect_uri":          []string{"https://app.example.com/callback"},
		"response_type":         []string{"code"},
		"scope":                 []string{"openid email"},
		"state":                 []string{"state-value"},
		"nonce":                 []string{"nonce-value"},
		"code_challenge":        []string{codeChallenge("verifier")},
		"code_challenge_method": []string{"S256"},
	}
}

func codeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

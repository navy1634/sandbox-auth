package oidc

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/url"
	"strings"
	"time"
)

var (
	ErrInvalidGrant = errors.New("invalid grant")
	ErrInvalidToken = errors.New("invalid token")
)

type Client struct {
	ID           string
	Secret       string
	SecretHash   []byte
	RedirectURIs []string
	Disabled     bool
}

type Config struct {
	IssuerURL      string
	Clients        []Client
	SigningKeyPEM  []byte
	KeyID          string
	CodeTTL        time.Duration
	AccessTokenTTL time.Duration
	IDTokenTTL     time.Duration
	Store          Store
}

type User struct {
	AccountID     int64
	Subject       string
	Email         string
	EmailVerified bool
}

type AuthorizationRequest struct {
	ClientID            string
	RedirectURI         string
	Scope               []string
	State               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
	PromptNone          bool
	PromptLogin         bool
}

type AuthorizationCodeData struct {
	ClientID    string
	RedirectURI string
	AccountID   int64
	Scope       []string
	Nonce       string
}

type AccessTokenData struct {
	ClientID  string
	AccountID int64
	Scope     []string
	ExpiresAt time.Time
}

type TokenSet struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	IDToken     string `json:"id_token"`
	Scope       string `json:"scope"`
}

type AuthorizationError struct {
	Code        string
	Description string
	RedirectURI string
	State       string
}

func (e *AuthorizationError) Error() string {
	if e.Description == "" {
		return e.Code
	}
	return e.Code + ": " + e.Description
}

type Provider struct {
	issuerURL      string
	store          Store
	signer         *signer
	codeTTL        time.Duration
	accessTokenTTL time.Duration
	idTokenTTL     time.Duration
}

type signer struct {
	privateKey *rsa.PrivateKey
	keyID      string
}

// OIDC Providerを初期化する。
func NewProvider(cfg Config) (*Provider, error) {
	if cfg.IssuerURL == "" || cfg.SigningKeyPEM == nil {
		return nil, errors.New("OIDC issuer and signing key are required")
	}
	issuerURL, err := url.Parse(cfg.IssuerURL)
	if err != nil || !issuerURL.IsAbs() || issuerURL.Host == "" || issuerURL.User != nil || issuerURL.RawQuery != "" || issuerURL.Fragment != "" {
		return nil, errors.New("OIDC issuer URL is invalid")
	}
	if cfg.CodeTTL <= 0 {
		cfg.CodeTTL = 60 * time.Second
	}
	if cfg.AccessTokenTTL <= 0 {
		cfg.AccessTokenTTL = time.Hour
	}
	if cfg.IDTokenTTL <= 0 {
		cfg.IDTokenTTL = time.Hour
	}

	key, err := parsePrivateKey(cfg.SigningKeyPEM)
	if err != nil {
		return nil, err
	}
	keyID := cfg.KeyID
	if keyID == "" {
		publicKey, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(publicKey)
		keyID = base64.RawURLEncoding.EncodeToString(digest[:8])
	}

	clientIDs := make(map[string]struct{}, len(cfg.Clients))
	for _, client := range cfg.Clients {
		if client.ID == "" || len(client.RedirectURIs) == 0 {
			return nil, errors.New("OIDC client ID and redirect URIs are required")
		}
		if _, ok := clientIDs[client.ID]; ok {
			return nil, errors.New("OIDC client ID must be unique")
		}
		clientIDs[client.ID] = struct{}{}
		for _, redirectURI := range client.RedirectURIs {
			parsedRedirectURI, err := url.Parse(redirectURI)
			if err != nil || !parsedRedirectURI.IsAbs() || parsedRedirectURI.Host == "" || parsedRedirectURI.User != nil || parsedRedirectURI.Fragment != "" {
				return nil, errors.New("OIDC redirect URI is invalid")
			}
		}
	}
	store := cfg.Store
	if store == nil {
		store = NewMemoryStore()
	}
	if err := store.EnsureClients(context.Background(), cfg.Clients); err != nil {
		return nil, err
	}
	if err := store.Cleanup(context.Background(), time.Now()); err != nil {
		return nil, err
	}

	return &Provider{
		issuerURL:      strings.TrimRight(issuerURL.String(), "/"),
		signer:         &signer{privateKey: key, keyID: keyID},
		codeTTL:        cfg.CodeTTL,
		accessTokenTTL: cfg.AccessTokenTTL,
		idTokenTTL:     cfg.IDTokenTTL,
		store:          store,
	}, nil
}

// OIDC DiscoveryのURLを返す。
func (p *Provider) Discovery() map[string]any {
	return map[string]any{
		"issuer":                                p.issuerURL,
		"authorization_endpoint":                p.EndpointURL("/oidc/authorize"),
		"token_endpoint":                        p.EndpointURL("/oidc/token"),
		"userinfo_endpoint":                     p.EndpointURL("/oidc/userinfo"),
		"jwks_uri":                              p.EndpointURL("/.well-known/jwks.json"),
		"response_types_supported":              []string{"code"},
		"response_modes_supported":              []string{"query"},
		"grant_types_supported":                 []string{"authorization_code"},
		"scopes_supported":                      []string{"openid", "email", "profile"},
		"claims_supported":                      []string{"sub", "iss", "aud", "exp", "iat", "auth_time", "nonce", "email", "email_verified", "preferred_username"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		"code_challenge_methods_supported":      []string{"S256"},
	}
}

// Issuer配下のOIDCエンドポイントURLを作る。
func (p *Provider) EndpointURL(path string) string {
	issuerURL, err := url.Parse(p.issuerURL)
	if err != nil {
		return ""
	}
	issuerURL.Path = strings.TrimRight(issuerURL.Path, "/") + path
	issuerURL.RawPath = ""
	return issuerURL.String()
}

// JWKSで公開するRSA公開鍵を返す。
func (p *Provider) JWKS() map[string]any {
	publicKey := &p.signer.privateKey.PublicKey
	return map[string]any{
		"keys": []map[string]any{{
			"kty": "RSA",
			"use": "sig",
			"alg": "RS256",
			"kid": p.signer.keyID,
			"n":   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
		}},
	}
}

// OIDC認可リクエストを検証する。
func (p *Provider) ParseAuthorizationRequest(values url.Values) (AuthorizationRequest, error) {
	return p.ParseAuthorizationRequestContext(context.Background(), values)
}

func (p *Provider) ParseAuthorizationRequestContext(ctx context.Context, values url.Values) (AuthorizationRequest, error) {
	clientID := values.Get("client_id")
	state := values.Get("state")
	client, err := p.store.FindClient(ctx, clientID)
	if err != nil {
		return AuthorizationRequest{}, &AuthorizationError{Code: "invalid_request", Description: "client_id is invalid", State: state}
	}

	redirectURI := values.Get("redirect_uri")
	if !contains(client.RedirectURIs, redirectURI) {
		return AuthorizationRequest{}, &AuthorizationError{Code: "invalid_request", Description: "redirect_uri is invalid", State: state}
	}
	request := AuthorizationRequest{
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		State:               state,
		Nonce:               values.Get("nonce"),
		CodeChallenge:       values.Get("code_challenge"),
		CodeChallengeMethod: values.Get("code_challenge_method"),
	}
	invalid := func(code string, description string) error {
		return &AuthorizationError{Code: code, Description: description, RedirectURI: redirectURI, State: state}
	}
	if values.Get("response_type") != "code" {
		return AuthorizationRequest{}, invalid("unsupported_response_type", "only response_type=code is supported")
	}
	if values.Get("response_mode") != "" && values.Get("response_mode") != "query" {
		return AuthorizationRequest{}, invalid("unsupported_response_mode", "only response_mode=query is supported")
	}
	request.Scope = strings.Fields(values.Get("scope"))
	if !contains(request.Scope, "openid") {
		return AuthorizationRequest{}, invalid("invalid_scope", "openid scope is required")
	}
	for _, scope := range request.Scope {
		if scope != "openid" && scope != "email" && scope != "profile" {
			return AuthorizationRequest{}, invalid("invalid_scope", "scope is not supported")
		}
	}
	if request.Nonce == "" {
		return AuthorizationRequest{}, invalid("invalid_request", "nonce is required")
	}
	if request.CodeChallenge == "" || request.CodeChallengeMethod != "S256" {
		return AuthorizationRequest{}, invalid("invalid_request", "S256 PKCE is required")
	}
	prompts := strings.Fields(values.Get("prompt"))
	for _, prompt := range prompts {
		if prompt != "none" && prompt != "login" {
			return AuthorizationRequest{}, invalid("invalid_request", "prompt is not supported")
		}
		if prompt == "none" {
			request.PromptNone = true
		}
		if prompt == "login" {
			request.PromptLogin = true
		}
	}
	if request.PromptNone && (len(prompts) != 1 || request.PromptLogin) {
		return AuthorizationRequest{}, invalid("invalid_request", "prompt=none cannot be combined")
	}
	return request, nil
}

// ログイン後に再開する認可要求を保存する。
func (p *Provider) CreateTransaction(request AuthorizationRequest) (string, error) {
	return p.CreateTransactionContext(context.Background(), request)
}

func (p *Provider) CreateTransactionContext(ctx context.Context, request AuthorizationRequest) (string, error) {
	if err := p.store.Cleanup(ctx, time.Now()); err != nil {
		return "", err
	}
	transactionID, err := randomString(32)
	if err != nil {
		return "", err
	}
	if err := p.store.SaveTransaction(ctx, transactionID, request, time.Now().Add(p.codeTTL)); err != nil {
		return "", err
	}
	return transactionID, nil
}

// ログイン後の認可要求を一回だけ取り出す。
func (p *Provider) ConsumeTransaction(transactionID string) (AuthorizationRequest, error) {
	return p.ConsumeTransactionContext(context.Background(), transactionID)
}

func (p *Provider) ConsumeTransactionContext(ctx context.Context, transactionID string) (AuthorizationRequest, error) {
	return p.store.ConsumeTransaction(ctx, transactionID, time.Now())
}

// 認可コードを一回限りの値として発行する。
func (p *Provider) IssueAuthorizationCode(request AuthorizationRequest, accountID int64) (string, error) {
	return p.IssueAuthorizationCodeContext(context.Background(), request, accountID)
}

func (p *Provider) IssueAuthorizationCodeContext(ctx context.Context, request AuthorizationRequest, accountID int64) (string, error) {
	if err := p.store.Cleanup(ctx, time.Now()); err != nil {
		return "", err
	}
	code, err := randomString(32)
	if err != nil {
		return "", err
	}
	if err := p.store.SaveAuthorizationCode(ctx, code, AuthorizationCodeRecord{
		Data: AuthorizationCodeData{
			ClientID:    request.ClientID,
			RedirectURI: request.RedirectURI,
			AccountID:   accountID,
			Scope:       append([]string(nil), request.Scope...),
			Nonce:       request.Nonce,
		},
		CodeChallenge:       request.CodeChallenge,
		CodeChallengeMethod: request.CodeChallengeMethod,
		ExpiresAt:           time.Now().Add(p.codeTTL),
	}); err != nil {
		return "", err
	}
	return code, nil
}

// 認可コードを検証して一回だけ消費する。
func (p *Provider) RedeemAuthorizationCode(code string, clientID string, redirectURI string, codeVerifier string) (AuthorizationCodeData, error) {
	return p.RedeemAuthorizationCodeContext(context.Background(), code, clientID, redirectURI, codeVerifier)
}

func (p *Provider) RedeemAuthorizationCodeContext(ctx context.Context, code string, clientID string, redirectURI string, codeVerifier string) (AuthorizationCodeData, error) {
	authorizationCode, err := p.store.RedeemAuthorizationCode(ctx, code, clientID, redirectURI, time.Now())
	if err != nil || !verifyPKCE(authorizationCode.CodeChallenge, codeVerifier) {
		return AuthorizationCodeData{}, ErrInvalidGrant
	}
	return authorizationCode.Data, nil
}

// クライアント認証情報を検証する。
func (p *Provider) AuthenticateClient(clientID string, clientSecret string) bool {
	return p.AuthenticateClientContext(context.Background(), clientID, clientSecret)
}

func (p *Provider) AuthenticateClientContext(ctx context.Context, clientID string, clientSecret string) bool {
	client, err := p.store.FindClient(ctx, clientID)
	if err != nil || len(client.SecretHash) == 0 {
		return false
	}
	return VerifyClientSecret(client.SecretHash, clientSecret)
}

// 認可コードからアクセストークンとIDトークンを発行する。
func (p *Provider) IssueTokenSet(data AuthorizationCodeData, user User) (TokenSet, error) {
	return p.IssueTokenSetContext(context.Background(), data, user)
}

func (p *Provider) IssueTokenSetContext(ctx context.Context, data AuthorizationCodeData, user User) (TokenSet, error) {
	if user.Subject == "" {
		return TokenSet{}, errors.New("OIDC subject is required")
	}
	if err := p.store.Cleanup(ctx, time.Now()); err != nil {
		return TokenSet{}, err
	}
	accessTokenValue, err := randomString(32)
	if err != nil {
		return TokenSet{}, err
	}
	now := time.Now()
	idTokenClaims := map[string]any{
		"iss":       p.issuerURL,
		"sub":       p.userSubject(user),
		"aud":       data.ClientID,
		"exp":       now.Add(p.idTokenTTL).Unix(),
		"iat":       now.Unix(),
		"auth_time": now.Unix(),
		"nonce":     data.Nonce,
	}
	if contains(data.Scope, "email") {
		idTokenClaims["email"] = user.Email
		idTokenClaims["email_verified"] = user.EmailVerified
	}
	if contains(data.Scope, "profile") && user.Email != "" {
		idTokenClaims["preferred_username"] = user.Email
	}
	idToken, err := p.signer.sign(idTokenClaims)
	if err != nil {
		return TokenSet{}, err
	}
	expiresIn := int64(p.accessTokenTTL / time.Second)
	accessTokenData := AccessTokenData{
		ClientID:  data.ClientID,
		AccountID: user.AccountID,
		Scope:     append([]string(nil), data.Scope...),
		ExpiresAt: now.Add(p.accessTokenTTL),
	}
	if err := p.store.SaveAccessToken(ctx, accessTokenValue, accessTokenData); err != nil {
		return TokenSet{}, err
	}
	return TokenSet{
		AccessToken: accessTokenValue,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		IDToken:     idToken,
		Scope:       strings.Join(data.Scope, " "),
	}, nil
}

// Bearerアクセストークンを検証する。
func (p *Provider) ValidateAccessToken(value string) (AccessTokenData, error) {
	return p.ValidateAccessTokenContext(context.Background(), value)
}

func (p *Provider) ValidateAccessTokenContext(ctx context.Context, value string) (AccessTokenData, error) {
	return p.store.ValidateAccessToken(ctx, value, time.Now())
}

func (p *Provider) userSubject(user User) string {
	return user.Subject
}

// UserInfoへ返すクレームを作る。
func (p *Provider) UserInfoClaims(data AccessTokenData, user User) map[string]any {
	claims := map[string]any{"sub": p.userSubject(user)}
	if contains(data.Scope, "email") {
		claims["email"] = user.Email
		claims["email_verified"] = user.EmailVerified
	}
	if contains(data.Scope, "profile") && user.Email != "" {
		claims["preferred_username"] = user.Email
	}
	return claims
}

// JWTをRSA SHA-256で署名する。
func (s *signer) sign(claims map[string]any) (string, error) {
	header, err := json.Marshal(map[string]string{"alg": "RS256", "kid": s.keyID, "typ": "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	headerPart := base64.RawURLEncoding.EncodeToString(header)
	payloadPart := base64.RawURLEncoding.EncodeToString(payload)
	message := headerPart + "." + payloadPart
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return message + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func parsePrivateKey(raw []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("OIDC signing key is not PEM")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("OIDC signing key is not an RSA private key")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("OIDC signing key is not an RSA private key")
	}
	return key, nil
}

func verifyPKCE(challenge string, verifier string) bool {
	if verifier == "" {
		return false
	}
	digest := sha256.Sum256([]byte(verifier))
	return hmac.Equal([]byte(challenge), []byte(base64.RawURLEncoding.EncodeToString(digest[:])))
}

func randomString(size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

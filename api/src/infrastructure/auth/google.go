package auth

import (
	"context"
	"errors"

	"github.com/sandbox-nextjs/src/domain"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
)

type OAuthProvider interface {
	AuthCodeURL(state string) string
	ExchangeAndValidate(ctx context.Context, code string) (domain.ProviderIdentity, error)
}

type GoogleProvider struct {
	clientID string
	oauth    *oauth2.Config
}

func NewGoogleProvider(clientID string, clientSecret string, redirectURL string) *GoogleProvider {
	// Google OAuth で認証 ID とメール確認状態だけを取得する。
	return &GoogleProvider{
		clientID: clientID,
		oauth: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email"},
			Endpoint:     google.Endpoint,
		},
	}
}

func (s *GoogleProvider) AuthCodeURL(state string) string {
	return s.oauth.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

func (s *GoogleProvider) ExchangeAndValidate(ctx context.Context, code string) (domain.ProviderIdentity, error) {
	// 認可コードをトークンに交換し、ID トークンからアプリで使う ID 情報を取り出す。
	token, err := s.oauth.Exchange(ctx, code)
	if err != nil {
		return domain.ProviderIdentity{}, err
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return domain.ProviderIdentity{}, errors.New("id token is missing")
	}

	return validateGoogleIDToken(ctx, rawIDToken, s.clientID)
}

func validateGoogleIDToken(ctx context.Context, rawIDToken string, clientID string) (domain.ProviderIdentity, error) {
	// Google が発行した ID トークンか検証し、メール確認済みのユーザーだけ許可する。
	payload, err := idtoken.Validate(ctx, rawIDToken, clientID)
	if err != nil {
		return domain.ProviderIdentity{}, err
	}

	email, _ := payload.Claims["email"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)
	if email == "" || !emailVerified {
		return domain.ProviderIdentity{}, errors.New("email is not verified")
	}

	return domain.ProviderIdentity{
		Provider:          "google",
		ProviderAccountID: payload.Subject,
		Email:             email,
		EmailVerified:     emailVerified,
	}, nil
}

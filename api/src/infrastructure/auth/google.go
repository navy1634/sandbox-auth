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
	return &GoogleProvider{
		clientID: clientID,
		oauth: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		},
	}
}

func (s *GoogleProvider) AuthCodeURL(state string) string {
	return s.oauth.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

func (s *GoogleProvider) ExchangeAndValidate(ctx context.Context, code string) (domain.ProviderIdentity, error) {
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
	payload, err := idtoken.Validate(ctx, rawIDToken, clientID)
	if err != nil {
		return domain.ProviderIdentity{}, err
	}

	email, _ := payload.Claims["email"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)
	if email == "" || !emailVerified {
		return domain.ProviderIdentity{}, errors.New("email is not verified")
	}

	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)

	return domain.ProviderIdentity{
		Provider:          "google",
		ProviderAccountID: payload.Subject,
		Email:             email,
		EmailVerified:     emailVerified,
		Name:              name,
		Picture:           picture,
	}, nil
}

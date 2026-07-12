package auth

import (
	"context"
	"errors"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
	"sandbox-nextjs/api/src/domain"
)

type GoogleService struct {
	clientID string
	oauth    *oauth2.Config
}

func NewGoogleService(clientID string, clientSecret string, redirectURL string) *GoogleService {
	return &GoogleService{
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

func (s *GoogleService) AuthCodeURL(state string) string {
	return s.oauth.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

func (s *GoogleService) ExchangeAndValidate(ctx context.Context, code string) (domain.GoogleUser, error) {
	token, err := s.oauth.Exchange(ctx, code)
	if err != nil {
		return domain.GoogleUser{}, err
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return domain.GoogleUser{}, errors.New("id token is missing")
	}

	return validateGoogleIDToken(ctx, rawIDToken, s.clientID)
}

func validateGoogleIDToken(ctx context.Context, rawIDToken string, clientID string) (domain.GoogleUser, error) {
	payload, err := idtoken.Validate(ctx, rawIDToken, clientID)
	if err != nil {
		return domain.GoogleUser{}, err
	}

	email, _ := payload.Claims["email"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)
	if email == "" || !emailVerified {
		return domain.GoogleUser{}, errors.New("email is not verified")
	}

	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)

	return domain.GoogleUser{
		Provider:          "google",
		ProviderAccountID: payload.Subject,
		Email:             email,
		EmailVerified:     emailVerified,
		Name:              name,
		Picture:           picture,
	}, nil
}

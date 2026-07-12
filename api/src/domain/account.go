package domain

import (
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

type GoogleUser struct {
	Provider          string
	ProviderAccountID string
	Email             string
	EmailVerified     bool
	Name              string
	Picture           string
}

type Account struct {
	ID                 int64                 `json:"id"`
	Provider           string                `json:"provider"`
	ProviderAccountID  string                `json:"providerAccountId"`
	Email              string                `json:"email"`
	EmailVerified      bool                  `json:"emailVerified"`
	Name               string                `json:"name"`
	Picture            string                `json:"picture"`
	DisplayName        string                `json:"displayName"`
	Bio                string                `json:"bio"`
	RegisteredAt       *time.Time            `json:"registeredAt"`
	CreatedAt          time.Time             `json:"createdAt"`
	UpdatedAt          time.Time             `json:"updatedAt"`
	WebAuthnUserHandle []byte                `json:"-"`
	Credentials        []webauthn.Credential `json:"-"`
}

type ProfileInput struct {
	DisplayName string
	Bio         string
}

type PasskeySession struct {
	AccountID *int64
	Ceremony  string
	Session   webauthn.SessionData
}

func (account Account) WebAuthnID() []byte {
	return account.WebAuthnUserHandle
}

func (account Account) WebAuthnName() string {
	return account.Email
}

func (account Account) WebAuthnDisplayName() string {
	if account.DisplayName != "" {
		return account.DisplayName
	}
	if account.Name != "" {
		return account.Name
	}
	return account.Email
}

func (account Account) WebAuthnCredentials() []webauthn.Credential {
	return account.Credentials
}

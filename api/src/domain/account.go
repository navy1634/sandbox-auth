package domain

import (
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

type ProviderIdentity struct {
	Provider          string
	ProviderAccountID string
	Email             string
	EmailVerified     bool
	Name              string
	Picture           string
}

type Account struct {
	ID                 int64                 `json:"id"`
	Identity           *ProviderIdentity     `json:"identity,omitempty"`
	DisplayName        string                `json:"displayName"`
	Bio                string                `json:"bio"`
	Credentials        []webauthn.Credential `json:"-"`
	WebAuthnUserHandle []byte                `json:"-"`
	CreatedAt          time.Time             `json:"createdAt"`
	UpdatedAt          time.Time             `json:"updatedAt"`
	RegisteredAt       *time.Time            `json:"registeredAt"`
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
	if account.Identity != nil {
		return account.Identity.Email
	}
	return ""
}

func (account Account) WebAuthnDisplayName() string {
	if account.DisplayName != "" {
		return account.DisplayName
	}
	if account.Identity != nil && account.Identity.Name != "" {
		return account.Identity.Name
	}
	return account.WebAuthnName()
}

func (account Account) WebAuthnCredentials() []webauthn.Credential {
	return account.Credentials
}

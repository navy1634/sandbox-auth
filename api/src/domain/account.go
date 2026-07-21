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
}

type Account struct {
	ID                 int64                 `json:"id"`
	Identity           *ProviderIdentity     `json:"identity,omitempty"`
	Credentials        []webauthn.Credential `json:"-"`
	WebAuthnUserHandle []byte                `json:"-"`
	CreatedAt          time.Time             `json:"createdAt"`
	UpdatedAt          time.Time             `json:"updatedAt"`
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
	// WebAuthn に表示する名前は、共通アカウントに紐づく認証 ID のメールを使う。
	return account.WebAuthnName()
}

func (account Account) WebAuthnCredentials() []webauthn.Credential {
	return account.Credentials
}

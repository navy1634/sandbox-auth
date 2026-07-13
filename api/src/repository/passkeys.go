package repository

import (
	"context"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/sandbox-nextjs/src/domain"
)

type PasskeyRepository interface {
	FindWebAuthnUserByID(ctx context.Context, id int64) (domain.Account, error)
	FindWebAuthnUserByHandle(ctx context.Context, handle []byte) (domain.Account, error)
	ListCredentials(ctx context.Context, accountID int64) ([]webauthn.Credential, error)
	SaveCredential(ctx context.Context, accountID int64, credential *webauthn.Credential) error
	UpdateCredential(ctx context.Context, accountID int64, credential *webauthn.Credential) error
	SaveSession(ctx context.Context, id string, accountID *int64, ceremony string, session *webauthn.SessionData, ttl time.Duration) error
	ConsumeSession(ctx context.Context, id string, ceremony string) (domain.PasskeySession, error)
}

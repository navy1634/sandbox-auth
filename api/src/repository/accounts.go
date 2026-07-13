package repository

import (
	"context"
	"errors"

	"github.com/sandbox-nextjs/src/domain"
)

var ErrNotFound = errors.New("not found")

type AccountRepository interface {
	UpsertProviderIdentity(ctx context.Context, identity domain.ProviderIdentity) (domain.Account, error)
	FindByID(ctx context.Context, id int64) (domain.Account, error)
	UpdateProfile(ctx context.Context, id int64, input domain.ProfileInput) (domain.Account, error)
	FindByWebAuthnUserHandle(ctx context.Context, handle []byte) (domain.Account, error)
}

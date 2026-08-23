package repository

import (
	"context"
	"errors"

	"github.com/sandbox-auth/src/domain"
)

var ErrNotFound = errors.New("not found")

type AccountRepository interface {
	// 外部認証 ID をアカウントへ紐づけ、未登録ならアカウントも作成する。
	UpsertProviderIdentity(ctx context.Context, identity domain.ProviderIdentity) (domain.Account, error)
	FindByID(ctx context.Context, id int64) (domain.Account, error)
	FindByWebAuthnUserHandle(ctx context.Context, handle []byte) (domain.Account, error)
}

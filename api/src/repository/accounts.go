package repository

import (
	"context"
	"errors"

	"github.com/sandbox-nextjs/src/domain"
)

var ErrNotFound = errors.New("not found")

type AccountRepository interface {
	// 外部認証 ID をアカウントへ紐づけ、未登録ならアカウントも作成する。
	UpsertProviderIdentity(ctx context.Context, identity domain.ProviderIdentity) (domain.Account, error)
	FindByID(ctx context.Context, id int64) (domain.Account, error)
	// プロフィール更新時に、初回登録完了の時刻も必要に応じて保存する。
	UpdateProfile(ctx context.Context, id int64, input domain.ProfileInput) (domain.Account, error)
	FindByWebAuthnUserHandle(ctx context.Context, handle []byte) (domain.Account, error)
}

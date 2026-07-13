package usecase

import (
	"context"

	"github.com/sandbox-nextjs/src/domain"
	"github.com/sandbox-nextjs/src/repository"
)

type AccountUsecase struct {
	accounts repository.AccountRepository
}

func NewAccountUsecase(accounts repository.AccountRepository) *AccountUsecase {
	return &AccountUsecase{accounts: accounts}
}

func (u *AccountUsecase) Me(ctx context.Context, accountID int64) (domain.Account, error) {
	return u.accounts.FindByID(ctx, accountID)
}

func (u *AccountUsecase) UpdateProfile(ctx context.Context, accountID int64, input domain.ProfileInput) (domain.Account, error) {
	// 保存前にプロフィール入力を正規化し、業務ルールを検証する。
	input = input.Normalize()
	if err := input.Validate(); err != nil {
		return domain.Account{}, err
	}

	return u.accounts.UpdateProfile(ctx, accountID, input)
}

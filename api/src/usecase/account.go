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

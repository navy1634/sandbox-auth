package usecase

import (
	"context"
	"errors"

	"github.com/sandbox-nextjs/src/domain"
	"github.com/sandbox-nextjs/src/infrastructure/auth"
	"github.com/sandbox-nextjs/src/infrastructure/session"
	"github.com/sandbox-nextjs/src/repository"
)

var (
	ErrAuthProviderNotFound = errors.New("auth provider not found")
	ErrOAuthExchangeFailed  = errors.New("oauth exchange failed")
	ErrOAuthSaveFailed      = errors.New("oauth save failed")
)

type OAuthUsecase struct {
	accounts  repository.AccountRepository
	providers map[string]auth.OAuthProvider
}

func NewOAuthUsecase(accounts repository.AccountRepository, providers map[string]auth.OAuthProvider) *OAuthUsecase {
	return &OAuthUsecase{
		accounts:  accounts,
		providers: providers,
	}
}

func (u *OAuthUsecase) BeginLogin(providerName string) (string, string, error) {
	provider, ok := u.providers[providerName]
	if !ok {
		return "", "", ErrAuthProviderNotFound
	}

	// OAuth の状態値を作り、認証プロバイダーへのリダイレクト先を返す。
	state, err := session.RandomString(32)
	if err != nil {
		return "", "", err
	}

	return state, provider.AuthCodeURL(state), nil
}

func (u *OAuthUsecase) Callback(ctx context.Context, providerName string, code string) (domain.Account, error) {
	provider, ok := u.providers[providerName]
	if !ok {
		return domain.Account{}, ErrAuthProviderNotFound
	}

	// OAuth の認可コードを検証済みの ID 情報に交換し、アカウントへ反映する。
	identity, err := provider.ExchangeAndValidate(ctx, code)
	if err != nil {
		return domain.Account{}, errors.Join(ErrOAuthExchangeFailed, err)
	}

	storedAccount, err := u.accounts.UpsertProviderIdentity(ctx, identity)
	if err != nil {
		return domain.Account{}, errors.Join(ErrOAuthSaveFailed, err)
	}

	return storedAccount, nil
}

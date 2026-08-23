package persistence

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/sandbox-auth/src/domain"
	"github.com/sandbox-auth/src/ent"
	entaccount "github.com/sandbox-auth/src/ent/account"
	entidentity "github.com/sandbox-auth/src/ent/authidentity"
	"github.com/sandbox-auth/src/infrastructure/database"
	"github.com/sandbox-auth/src/repository"
)

type EntAccountRepository struct {
	client *ent.Client
}

func NewEntAccountRepository(client *ent.Client) *EntAccountRepository {
	return &EntAccountRepository{client: client}
}

func (r *EntAccountRepository) db(ctx context.Context) (*ent.Client, error) {
	return database.ClientFromContext(ctx, r.client)
}

func (r *EntAccountRepository) UpsertProviderIdentity(ctx context.Context, identity domain.ProviderIdentity) (domain.Account, error) {
	client, err := r.db(ctx)
	if err != nil {
		return domain.Account{}, err
	}

	// 既存の外部認証 ID があれば、メールと確認状態を更新する
	storedIdentity, err := client.AuthIdentity.Query().
		Where(
			entidentity.Provider(identity.Provider),
			entidentity.ProviderAccountID(identity.ProviderAccountID),
		).
		Only(ctx)
	if err == nil {
		updatedIdentity, err := client.AuthIdentity.UpdateOneID(storedIdentity.ID).
			SetEmail(identity.Email).
			SetEmailVerified(identity.EmailVerified).
			Save(ctx)
		if err != nil {
			return domain.Account{}, err
		}
		return r.findByIDWithIdentity(ctx, updatedIdentity.AccountID, updatedIdentity)
	}
	if !ent.IsNotFound(err) {
		return domain.Account{}, err
	}

	// 初回ログイン時は、WebAuthn のユーザーハンドルを持つアカウントも作成する
	userHandle, err := randomBytes(32)
	if err != nil {
		return domain.Account{}, err
	}
	subject, err := randomSubject()
	if err != nil {
		return domain.Account{}, err
	}

	createdAccount, err := client.Account.Create().
		SetOidcSubject(subject).
		SetWebauthnUserHandle(userHandle).
		Save(ctx)
	if err != nil {
		return domain.Account{}, err
	}

	createdIdentity, err := client.AuthIdentity.Create().
		SetAccountID(int64(createdAccount.ID)).
		SetProvider(identity.Provider).
		SetProviderAccountID(identity.ProviderAccountID).
		SetEmail(identity.Email).
		SetEmailVerified(identity.EmailVerified).
		Save(ctx)
	if err != nil {
		return domain.Account{}, err
	}

	return toAccount(createdAccount, createdIdentity), nil
}

func (r *EntAccountRepository) FindByID(ctx context.Context, id int64) (domain.Account, error) {
	client, err := r.db(ctx)
	if err != nil {
		return domain.Account{}, err
	}

	storedAccount, err := client.Account.Get(ctx, int(id))
	if err != nil {
		return domain.Account{}, mapError(err)
	}

	storedIdentity, err := r.primaryIdentity(ctx, id)
	if err != nil {
		return domain.Account{}, err
	}

	return toAccount(storedAccount, storedIdentity), nil
}

func (r *EntAccountRepository) FindByWebAuthnUserHandle(ctx context.Context, handle []byte) (domain.Account, error) {
	client, err := r.db(ctx)
	if err != nil {
		return domain.Account{}, err
	}

	storedAccount, err := client.Account.Query().
		Where(entaccount.WebauthnUserHandle(handle)).
		Only(ctx)
	if err != nil {
		return domain.Account{}, mapError(err)
	}

	storedIdentity, err := r.primaryIdentity(ctx, int64(storedAccount.ID))
	if err != nil {
		return domain.Account{}, err
	}

	return toAccount(storedAccount, storedIdentity), nil
}

func (r *EntAccountRepository) findByIDWithIdentity(ctx context.Context, id int64, identity *ent.AuthIdentity) (domain.Account, error) {
	client, err := r.db(ctx)
	if err != nil {
		return domain.Account{}, err
	}

	storedAccount, err := client.Account.Get(ctx, int(id))
	if err != nil {
		return domain.Account{}, mapError(err)
	}

	return toAccount(storedAccount, identity), nil
}

func (r *EntAccountRepository) primaryIdentity(ctx context.Context, accountID int64) (*ent.AuthIdentity, error) {
	client, err := r.db(ctx)
	if err != nil {
		return nil, err
	}

	// 表示用の代表 ID として、最初に紐づいた外部認証 ID を使う
	storedIdentity, err := client.AuthIdentity.Query().
		Where(entidentity.AccountID(accountID)).
		Order(ent.Asc(entidentity.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return storedIdentity, nil
}

func toAccount(storedAccount *ent.Account, identity *ent.AuthIdentity) domain.Account {
	// Ent の保存形式から、handler と usecase が扱う domain.Account へ詰め替える
	account := domain.Account{
		ID:                 int64(storedAccount.ID),
		Subject:            storedAccount.OidcSubject,
		CreatedAt:          storedAccount.CreatedAt,
		UpdatedAt:          storedAccount.UpdatedAt,
		WebAuthnUserHandle: storedAccount.WebauthnUserHandle,
	}
	if identity != nil {
		account.Identity = &domain.ProviderIdentity{
			Provider:          identity.Provider,
			ProviderAccountID: identity.ProviderAccountID,
			Email:             identity.Email,
			EmailVerified:     identity.EmailVerified,
		}
	}
	return account
}

func randomBytes(size int) ([]byte, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

func randomSubject() (string, error) {
	bytes, err := randomBytes(32)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func mapError(err error) error {
	if ent.IsNotFound(err) {
		return repository.ErrNotFound
	}
	return err
}

var _ repository.AccountRepository = (*EntAccountRepository)(nil)

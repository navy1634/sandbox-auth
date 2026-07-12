package repository

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"time"

	"github.com/sandbox-nextjs/src/domain"
	"github.com/sandbox-nextjs/src/ent"
	entaccount "github.com/sandbox-nextjs/src/ent/account"
	entidentity "github.com/sandbox-nextjs/src/ent/authidentity"
	"github.com/sandbox-nextjs/src/infrastructure/database"
)

type AccountRepository struct {
	client *ent.Client
}

func NewAccountRepository(client *ent.Client) *AccountRepository {
	return &AccountRepository{client: client}
}

func (r *AccountRepository) db(ctx context.Context) (*ent.Client, error) {
	return database.ClientFromContext(ctx, r.client)
}

func (r *AccountRepository) UpsertProviderIdentity(ctx context.Context, identity domain.ProviderIdentity) (domain.Account, error) {
	client, err := r.db(ctx)
	if err != nil {
		return domain.Account{}, err
	}

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
			SetName(identity.Name).
			SetPicture(identity.Picture).
			Save(ctx)
		if err != nil {
			return domain.Account{}, err
		}
		return r.findByIDWithIdentity(ctx, updatedIdentity.AccountID, updatedIdentity)
	}
	if !ent.IsNotFound(err) {
		return domain.Account{}, err
	}

	userHandle, err := randomBytes(32)
	if err != nil {
		return domain.Account{}, err
	}

	createdAccount, err := client.Account.Create().
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
		SetName(identity.Name).
		SetPicture(identity.Picture).
		Save(ctx)
	if err != nil {
		return domain.Account{}, err
	}

	return toAccount(createdAccount, createdIdentity), nil
}

func (r *AccountRepository) FindByID(ctx context.Context, id int64) (domain.Account, error) {
	client, err := r.db(ctx)
	if err != nil {
		return domain.Account{}, err
	}

	storedAccount, err := client.Account.Get(ctx, int(id))
	if err != nil {
		return domain.Account{}, err
	}

	storedIdentity, err := r.primaryIdentity(ctx, id)
	if err != nil {
		return domain.Account{}, err
	}

	return toAccount(storedAccount, storedIdentity), nil
}

func (r *AccountRepository) UpdateProfile(ctx context.Context, id int64, input domain.ProfileInput) (domain.Account, error) {
	client, err := r.db(ctx)
	if err != nil {
		return domain.Account{}, err
	}

	displayName := strings.TrimSpace(input.DisplayName)
	bio := strings.TrimSpace(input.Bio)

	if displayName == "" {
		return domain.Account{}, errors.New("display name is required")
	}
	if len([]rune(displayName)) > 100 {
		return domain.Account{}, errors.New("display name is too long")
	}
	if len([]rune(bio)) > 500 {
		return domain.Account{}, errors.New("bio is too long")
	}

	builder := client.Account.UpdateOneID(int(id)).
		SetDisplayName(displayName).
		SetBio(bio)

	storedAccount, err := client.Account.Get(ctx, int(id))
	if err != nil {
		return domain.Account{}, err
	}
	if storedAccount.RegisteredAt == nil {
		builder.SetRegisteredAt(time.Now())
	}

	updatedAccount, err := builder.Save(ctx)
	if err != nil {
		return domain.Account{}, err
	}

	storedIdentity, err := r.primaryIdentity(ctx, id)
	if err != nil {
		return domain.Account{}, err
	}

	return toAccount(updatedAccount, storedIdentity), nil
}

func (r *AccountRepository) FindByWebAuthnUserHandle(ctx context.Context, handle []byte) (domain.Account, error) {
	client, err := r.db(ctx)
	if err != nil {
		return domain.Account{}, err
	}

	storedAccount, err := client.Account.Query().
		Where(entaccount.WebauthnUserHandle(handle)).
		Only(ctx)
	if err != nil {
		return domain.Account{}, err
	}

	storedIdentity, err := r.primaryIdentity(ctx, int64(storedAccount.ID))
	if err != nil {
		return domain.Account{}, err
	}

	return toAccount(storedAccount, storedIdentity), nil
}

func (r *AccountRepository) findByIDWithIdentity(ctx context.Context, id int64, identity *ent.AuthIdentity) (domain.Account, error) {
	client, err := r.db(ctx)
	if err != nil {
		return domain.Account{}, err
	}

	storedAccount, err := client.Account.Get(ctx, int(id))
	if err != nil {
		return domain.Account{}, err
	}

	return toAccount(storedAccount, identity), nil
}

func (r *AccountRepository) primaryIdentity(ctx context.Context, accountID int64) (*ent.AuthIdentity, error) {
	client, err := r.db(ctx)
	if err != nil {
		return nil, err
	}

	return client.AuthIdentity.Query().
		Where(entidentity.AccountID(accountID)).
		Order(ent.Asc(entidentity.FieldCreatedAt)).
		First(ctx)
}

func toAccount(storedAccount *ent.Account, identity *ent.AuthIdentity) domain.Account {
	account := domain.Account{
		ID:                 int64(storedAccount.ID),
		DisplayName:        storedAccount.DisplayName,
		Bio:                storedAccount.Bio,
		RegisteredAt:       storedAccount.RegisteredAt,
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
			Name:              identity.Name,
			Picture:           identity.Picture,
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

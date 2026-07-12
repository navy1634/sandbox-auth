package repository

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"sandbox-nextjs/api/src/domain"
	"sandbox-nextjs/api/src/ent"
	entaccount "sandbox-nextjs/api/src/ent/account"
	entcredential "sandbox-nextjs/api/src/ent/webauthncredential"
	entsession "sandbox-nextjs/api/src/ent/webauthnsession"
)

type AccountRepository struct {
	client *ent.Client
}

func NewAccountRepository(client *ent.Client) *AccountRepository {
	return &AccountRepository{client: client}
}

func (r *AccountRepository) UpsertGoogleAccount(ctx context.Context, user domain.GoogleUser) (domain.Account, error) {
	storedAccount, err := r.client.Account.Query().
		Where(
			entaccount.Provider(user.Provider),
			entaccount.ProviderAccountID(user.ProviderAccountID),
		).
		Only(ctx)
	if err == nil {
		updatedAccount, err := r.client.Account.UpdateOneID(storedAccount.ID).
			SetEmail(user.Email).
			SetEmailVerified(user.EmailVerified).
			SetName(user.Name).
			SetPicture(user.Picture).
			Save(ctx)
		if err != nil {
			return domain.Account{}, err
		}
		return toAccount(updatedAccount), nil
	}
	if !ent.IsNotFound(err) {
		return domain.Account{}, err
	}

	userHandle, err := randomBytes(32)
	if err != nil {
		return domain.Account{}, err
	}

	createdAccount, err := r.client.Account.Create().
		SetProvider(user.Provider).
		SetProviderAccountID(user.ProviderAccountID).
		SetEmail(user.Email).
		SetEmailVerified(user.EmailVerified).
		SetName(user.Name).
		SetPicture(user.Picture).
		SetWebauthnUserHandle(userHandle).
		Save(ctx)
	if err != nil {
		return domain.Account{}, err
	}

	return toAccount(createdAccount), nil
}

func (r *AccountRepository) FindByID(ctx context.Context, id int64) (domain.Account, error) {
	storedAccount, err := r.client.Account.Get(ctx, int(id))
	if err != nil {
		return domain.Account{}, err
	}

	return toAccount(storedAccount), nil
}

func (r *AccountRepository) UpdateProfile(ctx context.Context, id int64, input domain.ProfileInput) (domain.Account, error) {
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

	builder := r.client.Account.UpdateOneID(int(id)).
		SetDisplayName(displayName).
		SetBio(bio)

	storedAccount, err := r.client.Account.Get(ctx, int(id))
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

	return toAccount(updatedAccount), nil
}

func (r *AccountRepository) FindWebAuthnUserByID(ctx context.Context, id int64) (domain.Account, error) {
	account, err := r.FindByID(ctx, id)
	if err != nil {
		return domain.Account{}, err
	}

	credentials, err := r.ListCredentials(ctx, id)
	if err != nil {
		return domain.Account{}, err
	}

	account.Credentials = credentials
	return account, nil
}

func (r *AccountRepository) FindWebAuthnUserByHandle(ctx context.Context, handle []byte) (domain.Account, error) {
	storedAccount, err := r.client.Account.Query().
		Where(entaccount.WebauthnUserHandle(handle)).
		Only(ctx)
	if err != nil {
		return domain.Account{}, err
	}

	account := toAccount(storedAccount)
	credentials, err := r.ListCredentials(ctx, account.ID)
	if err != nil {
		return domain.Account{}, err
	}

	account.Credentials = credentials
	return account, nil
}

func (r *AccountRepository) ListCredentials(ctx context.Context, accountID int64) ([]webauthn.Credential, error) {
	storedCredentials, err := r.client.WebauthnCredential.Query().
		Where(entcredential.AccountID(accountID)).
		Order(ent.Asc(entcredential.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	credentials := make([]webauthn.Credential, 0, len(storedCredentials))
	for _, storedCredential := range storedCredentials {
		var credential webauthn.Credential
		if err := json.Unmarshal(storedCredential.CredentialJSON, &credential); err != nil {
			return nil, err
		}
		credentials = append(credentials, credential)
	}

	return credentials, nil
}

func (r *AccountRepository) SaveCredential(ctx context.Context, accountID int64, credential *webauthn.Credential) error {
	raw, err := json.Marshal(credential)
	if err != nil {
		return err
	}

	storedCredential, err := r.client.WebauthnCredential.Query().
		Where(entcredential.CredentialID(credential.ID)).
		Only(ctx)
	if err == nil {
		return r.client.WebauthnCredential.UpdateOneID(storedCredential.ID).
			SetAccountID(accountID).
			SetCredentialJSON(raw).
			Exec(ctx)
	}
	if !ent.IsNotFound(err) {
		return err
	}

	return r.client.WebauthnCredential.Create().
		SetAccountID(accountID).
		SetCredentialID(credential.ID).
		SetCredentialJSON(raw).
		Exec(ctx)
}

func (r *AccountRepository) UpdateCredential(ctx context.Context, accountID int64, credential *webauthn.Credential) error {
	raw, err := json.Marshal(credential)
	if err != nil {
		return err
	}

	storedCredential, err := r.client.WebauthnCredential.Query().
		Where(
			entcredential.AccountID(accountID),
			entcredential.CredentialID(credential.ID),
		).
		Only(ctx)
	if err != nil {
		return err
	}

	return r.client.WebauthnCredential.UpdateOneID(storedCredential.ID).
		SetCredentialJSON(raw).
		SetLastUsedAt(time.Now()).
		Exec(ctx)
}

func (r *AccountRepository) SavePasskeySession(ctx context.Context, id string, accountID *int64, ceremony string, session *webauthn.SessionData, ttl time.Duration) error {
	raw, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return r.client.WebauthnSession.Create().
		SetID(id).
		SetNillableAccountID(accountID).
		SetCeremony(ceremony).
		SetSessionJSON(raw).
		SetExpiresAt(time.Now().Add(ttl)).
		Exec(ctx)
}

func (r *AccountRepository) ConsumePasskeySession(ctx context.Context, id string, ceremony string) (domain.PasskeySession, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.PasskeySession{}, err
	}
	defer tx.Rollback()

	storedSession, err := tx.WebauthnSession.Query().
		Where(
			entsession.ID(id),
			entsession.Ceremony(ceremony),
			entsession.ExpiresAtGT(time.Now()),
		).
		Only(ctx)
	if err != nil {
		return domain.PasskeySession{}, err
	}

	if err := tx.WebauthnSession.DeleteOneID(id).Exec(ctx); err != nil {
		return domain.PasskeySession{}, err
	}

	var session webauthn.SessionData
	if err := json.Unmarshal(storedSession.SessionJSON, &session); err != nil {
		return domain.PasskeySession{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.PasskeySession{}, err
	}

	return domain.PasskeySession{
		AccountID: storedSession.AccountID,
		Ceremony:  ceremony,
		Session:   session,
	}, nil
}

func toAccount(storedAccount *ent.Account) domain.Account {
	return domain.Account{
		ID:                 int64(storedAccount.ID),
		Provider:           storedAccount.Provider,
		ProviderAccountID:  storedAccount.ProviderAccountID,
		Email:              storedAccount.Email,
		EmailVerified:      storedAccount.EmailVerified,
		Name:               storedAccount.Name,
		Picture:            storedAccount.Picture,
		DisplayName:        storedAccount.DisplayName,
		Bio:                storedAccount.Bio,
		RegisteredAt:       storedAccount.RegisteredAt,
		CreatedAt:          storedAccount.CreatedAt,
		UpdatedAt:          storedAccount.UpdatedAt,
		WebAuthnUserHandle: storedAccount.WebauthnUserHandle,
	}
}

func randomBytes(size int) ([]byte, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

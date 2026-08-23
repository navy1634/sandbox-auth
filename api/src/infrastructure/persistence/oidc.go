package persistence

import (
	"context"
	"time"

	"github.com/sandbox-nextjs/src/ent"
	"github.com/sandbox-nextjs/src/ent/oidcaccesstoken"
	"github.com/sandbox-nextjs/src/ent/oidcauthorizationcode"
	"github.com/sandbox-nextjs/src/ent/oidcauthorizationtransaction"
	"github.com/sandbox-nextjs/src/ent/oidcclient"
	"github.com/sandbox-nextjs/src/ent/oidcclientredirecturi"
	"github.com/sandbox-nextjs/src/infrastructure/database"
	"github.com/sandbox-nextjs/src/infrastructure/oidc"
)

type EntOIDCStore struct {
	client *ent.Client
}

func NewEntOIDCStore(client *ent.Client) *EntOIDCStore {
	return &EntOIDCStore{client: client}
}

func (s *EntOIDCStore) db(ctx context.Context) (*ent.Client, error) {
	return database.ClientFromContext(ctx, s.client)
}

func (s *EntOIDCStore) Cleanup(ctx context.Context, now time.Time) error {
	db, err := s.db(ctx)
	if err != nil {
		return err
	}
	if _, err := db.OIDCAccessToken.Delete().
		Where(oidcaccesstoken.ExpiresAtLTE(now)).
		Exec(ctx); err != nil {
		return err
	}
	if _, err := db.OIDCAuthorizationCode.Delete().
		Where(oidcauthorizationcode.ExpiresAtLTE(now)).
		Exec(ctx); err != nil {
		return err
	}
	_, err = db.OIDCAuthorizationTransaction.Delete().
		Where(oidcauthorizationtransaction.ExpiresAtLTE(now)).
		Exec(ctx)
	return err
}

func (s *EntOIDCStore) EnsureClients(ctx context.Context, clients []oidc.Client) error {
	db, err := s.db(ctx)
	if err != nil {
		return err
	}
	tx, err := db.Tx(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	db = tx.Client()
	configured := make(map[string]struct{}, len(clients))

	for _, client := range clients {
		configured[client.ID] = struct{}{}
		secretHash := client.SecretHash
		if len(secretHash) == 0 {
			secretHash, err = oidc.HashClientSecret(client.Secret)
			if err != nil {
				return err
			}
		}
		storedClient, err := db.OIDCClient.Query().
			Where(oidcclient.ClientID(client.ID)).
			Only(ctx)
		if ent.IsNotFound(err) {
			storedClient, err = db.OIDCClient.Create().
				SetClientID(client.ID).
				SetClientSecretHash(secretHash).
				SetDisabled(client.Disabled).
				Save(ctx)
		} else if err == nil {
			storedClient, err = db.OIDCClient.UpdateOneID(storedClient.ID).
				SetClientSecretHash(secretHash).
				SetDisabled(client.Disabled).
				Save(ctx)
		}
		if err != nil {
			return err
		}

		if _, err := db.OIDCClientRedirectURI.Delete().
			Where(oidcclientredirecturi.ClientID(storedClient.ClientID)).
			Exec(ctx); err != nil {
			return err
		}
		for _, redirectURI := range client.RedirectURIs {
			if err := db.OIDCClientRedirectURI.Create().
				SetClientID(storedClient.ClientID).
				SetRedirectURI(redirectURI).
				Exec(ctx); err != nil {
				return err
			}
		}
	}
	storedClients, err := db.OIDCClient.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, storedClient := range storedClients {
		if _, ok := configured[storedClient.ClientID]; ok {
			continue
		}
		if _, err := db.OIDCClient.UpdateOneID(storedClient.ID).
			SetDisabled(true).
			Save(ctx); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (s *EntOIDCStore) FindClient(ctx context.Context, clientID string) (oidc.Client, error) {
	db, err := s.db(ctx)
	if err != nil {
		return oidc.Client{}, err
	}
	storedClient, err := db.OIDCClient.Query().
		Where(
			oidcclient.ClientID(clientID),
			oidcclient.Disabled(false),
		).
		Only(ctx)
	if ent.IsNotFound(err) {
		return oidc.Client{}, oidc.ErrInvalidClient
	}
	if err != nil {
		return oidc.Client{}, err
	}
	redirects, err := db.OIDCClientRedirectURI.Query().
		Where(oidcclientredirecturi.ClientID(clientID)).
		Order(ent.Asc(oidcclientredirecturi.FieldRedirectURI)).
		All(ctx)
	if err != nil {
		return oidc.Client{}, err
	}
	redirectURIs := make([]string, 0, len(redirects))
	for _, redirect := range redirects {
		redirectURIs = append(redirectURIs, redirect.RedirectURI)
	}
	return oidc.Client{
		ID:           storedClient.ClientID,
		SecretHash:   append([]byte(nil), storedClient.ClientSecretHash...),
		RedirectURIs: redirectURIs,
		Disabled:     storedClient.Disabled,
	}, nil
}

func (s *EntOIDCStore) SaveTransaction(ctx context.Context, transactionID string, request oidc.AuthorizationRequest, expiresAt time.Time) error {
	db, err := s.db(ctx)
	if err != nil {
		return err
	}
	return db.OIDCAuthorizationTransaction.Create().
		SetTransactionHash(oidc.HashSecret(transactionID)).
		SetClientID(request.ClientID).
		SetRedirectURI(request.RedirectURI).
		SetScope(request.Scope).
		SetState(request.State).
		SetNonce(request.Nonce).
		SetCodeChallenge(request.CodeChallenge).
		SetCodeChallengeMethod(request.CodeChallengeMethod).
		SetExpiresAt(expiresAt).
		Exec(ctx)
}

func (s *EntOIDCStore) ConsumeTransaction(ctx context.Context, transactionID string, now time.Time) (oidc.AuthorizationRequest, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return oidc.AuthorizationRequest{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	db := tx.Client()
	stored, err := db.OIDCAuthorizationTransaction.Query().
		Where(
			oidcauthorizationtransaction.TransactionHash(oidc.HashSecret(transactionID)),
			oidcauthorizationtransaction.ExpiresAtGT(now),
			oidcauthorizationtransaction.ConsumedAtIsNil(),
		).
		Only(ctx)
	if ent.IsNotFound(err) {
		return oidc.AuthorizationRequest{}, oidc.ErrInvalidGrant
	}
	if err != nil {
		return oidc.AuthorizationRequest{}, err
	}
	updated, err := db.OIDCAuthorizationTransaction.Update().
		Where(
			oidcauthorizationtransaction.ID(stored.ID),
			oidcauthorizationtransaction.ExpiresAtGT(now),
			oidcauthorizationtransaction.ConsumedAtIsNil(),
		).
		SetConsumedAt(now).
		Save(ctx)
	if err != nil {
		return oidc.AuthorizationRequest{}, err
	}
	if updated != 1 {
		return oidc.AuthorizationRequest{}, oidc.ErrInvalidGrant
	}
	if err := tx.Commit(); err != nil {
		return oidc.AuthorizationRequest{}, err
	}
	committed = true
	return oidc.AuthorizationRequest{
		ClientID:            stored.ClientID,
		RedirectURI:         stored.RedirectURI,
		Scope:               append([]string(nil), stored.Scope...),
		State:               stored.State,
		Nonce:               stored.Nonce,
		CodeChallenge:       stored.CodeChallenge,
		CodeChallengeMethod: stored.CodeChallengeMethod,
	}, nil
}

func (s *EntOIDCStore) SaveAuthorizationCode(ctx context.Context, code string, record oidc.AuthorizationCodeRecord) error {
	db, err := s.db(ctx)
	if err != nil {
		return err
	}
	return db.OIDCAuthorizationCode.Create().
		SetCodeHash(oidc.HashSecret(code)).
		SetClientID(record.Data.ClientID).
		SetRedirectURI(record.Data.RedirectURI).
		SetAccountID(record.Data.AccountID).
		SetScope(record.Data.Scope).
		SetNonce(record.Data.Nonce).
		SetCodeChallenge(record.CodeChallenge).
		SetCodeChallengeMethod(record.CodeChallengeMethod).
		SetExpiresAt(record.ExpiresAt).
		Exec(ctx)
}

func (s *EntOIDCStore) RedeemAuthorizationCode(ctx context.Context, code string, clientID string, redirectURI string, now time.Time) (oidc.AuthorizationCodeRecord, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return oidc.AuthorizationCodeRecord{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	db := tx.Client()
	stored, err := db.OIDCAuthorizationCode.Query().
		Where(
			oidcauthorizationcode.CodeHash(oidc.HashSecret(code)),
			oidcauthorizationcode.ClientID(clientID),
			oidcauthorizationcode.RedirectURI(redirectURI),
			oidcauthorizationcode.ExpiresAtGT(now),
			oidcauthorizationcode.ConsumedAtIsNil(),
		).
		Only(ctx)
	if ent.IsNotFound(err) {
		return oidc.AuthorizationCodeRecord{}, oidc.ErrInvalidGrant
	}
	if err != nil {
		return oidc.AuthorizationCodeRecord{}, err
	}
	updated, err := db.OIDCAuthorizationCode.Update().
		Where(
			oidcauthorizationcode.ID(stored.ID),
			oidcauthorizationcode.ExpiresAtGT(now),
			oidcauthorizationcode.ConsumedAtIsNil(),
		).
		SetConsumedAt(now).
		Save(ctx)
	if err != nil {
		return oidc.AuthorizationCodeRecord{}, err
	}
	if updated != 1 {
		return oidc.AuthorizationCodeRecord{}, oidc.ErrInvalidGrant
	}
	if err := tx.Commit(); err != nil {
		return oidc.AuthorizationCodeRecord{}, err
	}
	committed = true
	return oidc.AuthorizationCodeRecord{
		Data: oidc.AuthorizationCodeData{
			ClientID:    stored.ClientID,
			RedirectURI: stored.RedirectURI,
			AccountID:   stored.AccountID,
			Scope:       append([]string(nil), stored.Scope...),
			Nonce:       stored.Nonce,
		},
		CodeChallenge:       stored.CodeChallenge,
		CodeChallengeMethod: stored.CodeChallengeMethod,
		ExpiresAt:           stored.ExpiresAt,
	}, nil
}

func (s *EntOIDCStore) SaveAccessToken(ctx context.Context, token string, data oidc.AccessTokenData) error {
	db, err := s.db(ctx)
	if err != nil {
		return err
	}
	return db.OIDCAccessToken.Create().
		SetTokenHash(oidc.HashSecret(token)).
		SetClientID(data.ClientID).
		SetAccountID(data.AccountID).
		SetScope(data.Scope).
		SetExpiresAt(data.ExpiresAt).
		Exec(ctx)
}

func (s *EntOIDCStore) ValidateAccessToken(ctx context.Context, token string, now time.Time) (oidc.AccessTokenData, error) {
	db, err := s.db(ctx)
	if err != nil {
		return oidc.AccessTokenData{}, err
	}
	stored, err := db.OIDCAccessToken.Query().
		Where(
			oidcaccesstoken.TokenHash(oidc.HashSecret(token)),
			oidcaccesstoken.ExpiresAtGT(now),
			oidcaccesstoken.RevokedAtIsNil(),
		).
		Only(ctx)
	if ent.IsNotFound(err) {
		return oidc.AccessTokenData{}, oidc.ErrInvalidToken
	}
	if err != nil {
		return oidc.AccessTokenData{}, err
	}
	if _, err := db.OIDCClient.Query().
		Where(
			oidcclient.ClientID(stored.ClientID),
			oidcclient.Disabled(false),
		).
		Only(ctx); ent.IsNotFound(err) {
		return oidc.AccessTokenData{}, oidc.ErrInvalidToken
	} else if err != nil {
		return oidc.AccessTokenData{}, err
	}
	return oidc.AccessTokenData{
		ClientID:  stored.ClientID,
		AccountID: stored.AccountID,
		Scope:     append([]string(nil), stored.Scope...),
		ExpiresAt: stored.ExpiresAt,
	}, nil
}

var _ oidc.Store = (*EntOIDCStore)(nil)

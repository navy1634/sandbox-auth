package persistence

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/sandbox-auth/src/domain"
	"github.com/sandbox-auth/src/ent"
	entcredential "github.com/sandbox-auth/src/ent/webauthncredential"
	entsession "github.com/sandbox-auth/src/ent/webauthnsession"
	"github.com/sandbox-auth/src/infrastructure/database"
	"github.com/sandbox-auth/src/repository"
)

const (
	maxActiveWebAuthnRegistrationSessions = 1000
	maxActiveWebAuthnLoginSessions        = 10000
)

type EntPasskeyRepository struct {
	client   *ent.Client
	accounts *EntAccountRepository
	mu       sync.Mutex
}

// ent を使う PasskeyRepository を作る
func NewEntPasskeyRepository(client *ent.Client, accounts *EntAccountRepository) *EntPasskeyRepository {
	return &EntPasskeyRepository{client: client, accounts: accounts}
}

// リクエスト context にあるトランザクション用 client を優先して返す
func (r *EntPasskeyRepository) db(ctx context.Context) (*ent.Client, error) {
	return database.ClientFromContext(ctx, r.client)
}

// WebAuthn ユーザー ID から認証情報付きのアカウントを返す
func (r *EntPasskeyRepository) FindWebAuthnUserByID(ctx context.Context, id int64) (domain.Account, error) {
	// WebAuthn 検証に必要なアカウント情報と認証情報をまとめて返す
	account, err := r.accounts.FindByID(ctx, id)
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

// WebAuthn user handle から認証情報付きのアカウントを返す
func (r *EntPasskeyRepository) FindWebAuthnUserByHandle(ctx context.Context, handle []byte) (domain.Account, error) {
	account, err := r.accounts.FindByWebAuthnUserHandle(ctx, handle)
	if err != nil {
		return domain.Account{}, err
	}

	credentials, err := r.ListCredentials(ctx, account.ID)
	if err != nil {
		return domain.Account{}, err
	}

	account.Credentials = credentials
	return account, nil
}

// 指定アカウントに登録された WebAuthn credential を返す
func (r *EntPasskeyRepository) ListCredentials(ctx context.Context, accountID int64) ([]webauthn.Credential, error) {
	client, err := r.db(ctx)
	if err != nil {
		return nil, err
	}

	// DB には JSON で保存している認証情報を、WebAuthn ライブラリの型へ戻す
	storedCredentials, err := client.WebauthnCredential.Query().
		Where(entcredential.AccountID(accountID)).
		Order(ent.Asc(entcredential.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, mapError(err)
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

// 指定アカウントの WebAuthn credential を新規保存する
func (r *EntPasskeyRepository) SaveCredential(ctx context.Context, accountID int64, credential *webauthn.Credential) error {
	client, err := r.db(ctx)
	if err != nil {
		return err
	}

	// WebAuthn の認証情報は構造が大きいため、そのまま JSON として保存する
	raw, err := json.Marshal(credential)
	if err != nil {
		return err
	}

	_, err = client.WebauthnCredential.Query().
		Where(entcredential.CredentialID(credential.ID)).
		Only(ctx)
	if err == nil {
		return repository.ErrPasskeyCredentialExists
	}
	if !ent.IsNotFound(err) {
		return err
	}

	return client.WebauthnCredential.Create().
		SetAccountID(accountID).
		SetCredentialID(credential.ID).
		SetCredentialJSON(raw).
		Exec(ctx)
}

// 指定アカウントの既存 WebAuthn credential を更新する
func (r *EntPasskeyRepository) UpdateCredential(ctx context.Context, accountID int64, credential *webauthn.Credential) error {
	client, err := r.db(ctx)
	if err != nil {
		return err
	}

	raw, err := json.Marshal(credential)
	if err != nil {
		return err
	}

	storedCredential, err := client.WebauthnCredential.Query().
		Where(
			entcredential.AccountID(accountID),
			entcredential.CredentialID(credential.ID),
		).
		Only(ctx)
	if err != nil {
		return mapError(err)
	}

	return client.WebauthnCredential.UpdateOneID(storedCredential.ID).
		SetCredentialJSON(raw).
		SetLastUsedAt(time.Now()).
		Exec(ctx)
}

// WebAuthn ceremony のセッションを期限付きで保存する
func (r *EntPasskeyRepository) SaveSession(ctx context.Context, id string, accountID *int64, ceremony string, session *webauthn.SessionData, ttl time.Duration) error {
	// この処理では、件数確認から INSERT までを同一プロセス内で直列化する
	r.mu.Lock()
	defer r.mu.Unlock()

	client, err := r.db(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	// この処理では、保存前に期限切れ WebAuthn セッションを削除する
	if _, err := client.WebauthnSession.Delete().
		Where(entsession.ExpiresAtLTE(now)).
		Exec(ctx); err != nil {
		return err
	}
	if err := r.ensureSessionLimit(ctx, client, accountID, ceremony, now); err != nil {
		return err
	}

	// チャレンジの検証に必要なセッションデータを、有効期限付きで保存する
	raw, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return client.WebauthnSession.Create().
		SetID(id).
		SetNillableAccountID(accountID).
		SetCeremony(ceremony).
		SetSessionJSON(raw).
		SetExpiresAt(now.Add(ttl)).
		Exec(ctx)
}

// 有効な WebAuthn セッション数が保存上限を超えていないか確認する
func (r *EntPasskeyRepository) ensureSessionLimit(ctx context.Context, client *ent.Client, accountID *int64, ceremony string, now time.Time) error {
	query := client.WebauthnSession.Query().
		Where(
			entsession.Ceremony(ceremony),
			entsession.ExpiresAtGT(now),
		)
	limit := maxActiveWebAuthnLoginSessions
	if accountID != nil {
		query = query.Where(entsession.AccountID(*accountID))
		limit = maxActiveWebAuthnRegistrationSessions
	} else {
		query = query.Where(entsession.AccountIDIsNil())
	}

	count, err := query.Count(ctx)
	if err != nil {
		return err
	}
	if count >= limit {
		return repository.ErrTooManyPasskeySessions
	}
	return nil
}

// WebAuthn ceremony セッションを取得して削除する
func (r *EntPasskeyRepository) ConsumeSession(ctx context.Context, id string, ceremony string) (domain.PasskeySession, error) {
	client, err := r.db(ctx)
	if err != nil {
		return domain.PasskeySession{}, err
	}

	// 処理種別と有効期限が一致するセッションだけを使い、再利用を防ぐため削除する
	storedSession, err := client.WebauthnSession.Query().
		Where(
			entsession.ID(id),
			entsession.Ceremony(ceremony),
			entsession.ExpiresAtGT(time.Now()),
		).
		Only(ctx)
	if err != nil {
		return domain.PasskeySession{}, mapError(err)
	}

	if err := client.WebauthnSession.DeleteOneID(id).Exec(ctx); err != nil {
		return domain.PasskeySession{}, err
	}

	var session webauthn.SessionData
	if err := json.Unmarshal(storedSession.SessionJSON, &session); err != nil {
		return domain.PasskeySession{}, err
	}

	return domain.PasskeySession{
		AccountID: storedSession.AccountID,
		Ceremony:  ceremony,
		Session:   session,
	}, nil
}

var _ repository.PasskeyRepository = (*EntPasskeyRepository)(nil)

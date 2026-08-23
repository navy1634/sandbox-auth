package repository

import (
	"context"
	"errors"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/sandbox-auth/src/domain"
)

// 保存可能な WebAuthn セッション数を超えたときのエラー。
var ErrTooManyPasskeySessions = errors.New("too many passkey sessions")

// 同じ credential ID が既に保存されているときのエラー。
var ErrPasskeyCredentialExists = errors.New("passkey credential already exists")

// パスキーの credential と WebAuthn セッションを永続化する。
type PasskeyRepository interface {
	// WebAuthn ライブラリへ渡すため、アカウントと登録済み認証情報をまとめて取得する。
	FindWebAuthnUserByID(ctx context.Context, id int64) (domain.Account, error)
	// WebAuthn user handle からアカウントと登録済み認証情報を取得する。
	FindWebAuthnUserByHandle(ctx context.Context, handle []byte) (domain.Account, error)
	// 指定アカウントの WebAuthn credential を一覧する。
	ListCredentials(ctx context.Context, accountID int64) ([]webauthn.Credential, error)
	// 指定アカウントへ WebAuthn credential を保存する。
	SaveCredential(ctx context.Context, accountID int64, credential *webauthn.Credential) error
	// 指定アカウントの WebAuthn credential を更新する。
	UpdateCredential(ctx context.Context, accountID int64, credential *webauthn.Credential) error
	// WebAuthn のチャレンジを検証リクエストまで一時保存する。
	SaveSession(ctx context.Context, id string, accountID *int64, ceremony string, session *webauthn.SessionData, ttl time.Duration) error
	// WebAuthn セッションを取得して削除する。
	ConsumeSession(ctx context.Context, id string, ceremony string) (domain.PasskeySession, error)
}

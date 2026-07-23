package usecase

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/sandbox-nextjs/src/domain"
	"github.com/sandbox-nextjs/src/infrastructure/auth"
	"github.com/sandbox-nextjs/src/infrastructure/session"
	"github.com/sandbox-nextjs/src/repository"
)

const (
	PasskeyRegisterCeremony = "passkey_register"
	PasskeyLoginCeremony    = "passkey_login"
)

var (
	ErrPasskeySessionAccountMismatch = errors.New("passkey session does not match account")
	ErrInvalidPasskeyUser            = errors.New("invalid passkey user")
	ErrPasskeyLoadAccountFailed      = errors.New("failed to load passkey account")
	ErrPasskeySaveSessionFailed      = errors.New("failed to save passkey session")
	ErrPasskeySessionInvalid         = errors.New("invalid passkey session")
	ErrPasskeySaveCredentialFailed   = errors.New("failed to save passkey credential")
	ErrPasskeyUpdateCredentialFailed = errors.New("failed to update passkey credential")
	ErrPasskeyVerifyFailed           = errors.New("failed to verify passkey")
	ErrPasskeySessionLimitExceeded   = errors.New("too many passkey sessions")
	ErrPasskeyCredentialExists       = errors.New("passkey credential already exists")
)

type PasskeyUsecase struct {
	passkeys repository.PasskeyRepository
	passkey  *auth.PasskeyService
}

type PasskeyRegistrationOptions struct {
	Creation  *protocol.CredentialCreation
	SessionID string
}

type PasskeyLoginOptions struct {
	Assertion *protocol.CredentialAssertion
	SessionID string
}

// パスキー認証のユースケースを作る。
func NewPasskeyUsecase(passkeys repository.PasskeyRepository, passkey *auth.PasskeyService) *PasskeyUsecase {
	return &PasskeyUsecase{
		passkeys: passkeys,
		passkey:  passkey,
	}
}

// 指定アカウントのパスキー登録 ceremony を開始する。
func (u *PasskeyUsecase) BeginRegistration(ctx context.Context, accountID int64) (PasskeyRegistrationOptions, error) {
	// WebAuthn ユーザーを読み込み、登録用の選択肢を発行する。
	storedAccount, err := u.passkeys.FindWebAuthnUserByID(ctx, accountID)
	if err != nil {
		return PasskeyRegistrationOptions{}, errors.Join(ErrPasskeyLoadAccountFailed, err)
	}

	creation, passkeySession, err := u.passkey.BeginRegistration(storedAccount)
	if err != nil {
		return PasskeyRegistrationOptions{}, err
	}

	// 検証リクエストが来るまでチャレンジ用セッションを保存する。
	sessionID, err := session.RandomString(32)
	if err != nil {
		return PasskeyRegistrationOptions{}, err
	}

	if err := u.passkeys.SaveSession(ctx, sessionID, &accountID, PasskeyRegisterCeremony, passkeySession, 5*time.Minute); errors.Is(err, repository.ErrTooManyPasskeySessions) {
		return PasskeyRegistrationOptions{}, errors.Join(ErrPasskeySessionLimitExceeded, err)
	} else if err != nil {
		return PasskeyRegistrationOptions{}, errors.Join(ErrPasskeySaveSessionFailed, err)
	}

	return PasskeyRegistrationOptions{
		Creation:  creation,
		SessionID: sessionID,
	}, nil
}

// 登録 ceremony のセッションと認証器応答を検証して credential を保存する。
func (u *PasskeyUsecase) FinishRegistration(ctx context.Context, accountID int64, sessionID string, request *http.Request) error {
	// チャレンジを消費し、ログイン中のアカウントのものか確認する。
	passkeySession, err := u.passkeys.ConsumeSession(ctx, sessionID, PasskeyRegisterCeremony)
	if err != nil {
		return errors.Join(ErrPasskeySessionInvalid, err)
	}
	if passkeySession.AccountID == nil || *passkeySession.AccountID != accountID {
		return ErrPasskeySessionAccountMismatch
	}

	storedAccount, err := u.passkeys.FindWebAuthnUserByID(ctx, accountID)
	if err != nil {
		return errors.Join(ErrPasskeyLoadAccountFailed, err)
	}

	// 認証器の応答を検証し、新しい認証情報を保存する。
	credential, err := u.passkey.FinishRegistration(storedAccount, passkeySession.Session, request)
	if err != nil {
		return errors.Join(ErrPasskeyVerifyFailed, err)
	}

	if err := u.passkeys.SaveCredential(ctx, storedAccount.ID, credential); errors.Is(err, repository.ErrPasskeyCredentialExists) {
		return errors.Join(ErrPasskeyCredentialExists, err)
	} else if err != nil {
		return errors.Join(ErrPasskeySaveCredentialFailed, err)
	}

	return nil
}

// ユーザー名なしのパスキーログイン ceremony を開始する。
func (u *PasskeyUsecase) BeginLogin(ctx context.Context) (PasskeyLoginOptions, error) {
	// ユーザー名なしログイン用のチャレンジを発行し、検証用に保存する。
	assertion, passkeySession, err := u.passkey.BeginLogin()
	if err != nil {
		return PasskeyLoginOptions{}, err
	}

	sessionID, err := session.RandomString(32)
	if err != nil {
		return PasskeyLoginOptions{}, err
	}

	if err := u.passkeys.SaveSession(ctx, sessionID, nil, PasskeyLoginCeremony, passkeySession, 5*time.Minute); errors.Is(err, repository.ErrTooManyPasskeySessions) {
		return PasskeyLoginOptions{}, errors.Join(ErrPasskeySessionLimitExceeded, err)
	} else if err != nil {
		return PasskeyLoginOptions{}, errors.Join(ErrPasskeySaveSessionFailed, err)
	}

	return PasskeyLoginOptions{
		Assertion: assertion,
		SessionID: sessionID,
	}, nil
}

// ログイン ceremony のセッションと認証器応答を検証してアカウントを返す。
func (u *PasskeyUsecase) FinishLogin(ctx context.Context, sessionID string, request *http.Request) (domain.Account, error) {
	// 認証器の応答を検証する前に、ログイン用チャレンジを消費する。
	passkeySession, err := u.passkeys.ConsumeSession(ctx, sessionID, PasskeyLoginCeremony)
	if err != nil {
		return domain.Account{}, errors.Join(ErrPasskeySessionInvalid, err)
	}

	// 検証中にパスキーのユーザーハンドルからアカウントを解決する。
	validatedUser, validatedCredential, err := u.passkey.FinishLogin(func(rawID []byte, userHandle []byte) (webauthn.User, error) {
		return u.passkeys.FindWebAuthnUserByHandle(ctx, userHandle)
	}, passkeySession.Session, request)
	if err != nil {
		return domain.Account{}, err
	}

	storedAccount, ok := validatedUser.(domain.Account)
	if !ok {
		return domain.Account{}, ErrInvalidPasskeyUser
	}

	// ログイン成功後に認証情報の利用情報を保存する。
	if err := u.passkeys.UpdateCredential(ctx, storedAccount.ID, validatedCredential); err != nil {
		return domain.Account{}, errors.Join(ErrPasskeyUpdateCredentialFailed, err)
	}

	return storedAccount, nil
}

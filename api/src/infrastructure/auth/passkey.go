package auth

import (
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/sandbox-auth/src/domain"
)

type PasskeyService struct {
	webauthn *webauthn.WebAuthn
}

// RP 設定を使って WebAuthn service を作る。
func NewPasskeyService(rpID string, rpOrigin string) (*PasskeyService, error) {
	// このアプリの RP 情報と認証器要件を WebAuthn ライブラリへ渡す。
	passkey, err := webauthn.New(&webauthn.Config{
		RPID:          rpID,
		RPDisplayName: "sandbox_auth",
		RPOrigins:     []string{rpOrigin},
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementRequired,
			UserVerification: protocol.VerificationRequired,
		},
	})
	if err != nil {
		return nil, err
	}

	return &PasskeyService{webauthn: passkey}, nil
}

// 指定ユーザーの WebAuthn 登録 ceremony を開始する。
func (s *PasskeyService) BeginRegistration(user domain.Account) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
	// 登録済みの認証情報を除外し、同じ認証器の重複登録を避ける。
	return s.webauthn.BeginRegistration(
		user,
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		webauthn.WithExclusions(webauthn.Credentials(user.WebAuthnCredentials()).CredentialDescriptors()),
		webauthn.WithExtensions(map[string]any{"credProps": true}),
	)
}

// WebAuthn 登録 ceremony の応答を検証して credential を返す。
func (s *PasskeyService) FinishRegistration(user domain.Account, session webauthn.SessionData, request *http.Request) (*webauthn.Credential, error) {
	return s.webauthn.FinishRegistration(user, session, request)
}

// discoverable credential を使う WebAuthn ログイン ceremony を開始する。
func (s *PasskeyService) BeginLogin() (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	// ユーザー名入力なしで、認証器からユーザーを解決するログインを開始する。
	return s.webauthn.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationRequired))
}

// WebAuthn ログイン ceremony の応答を検証してユーザーと credential を返す。
func (s *PasskeyService) FinishLogin(handler webauthn.DiscoverableUserHandler, session webauthn.SessionData, request *http.Request) (webauthn.User, *webauthn.Credential, error) {
	return s.webauthn.FinishPasskeyLogin(handler, session, request)
}

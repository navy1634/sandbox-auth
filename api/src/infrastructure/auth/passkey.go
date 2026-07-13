package auth

import (
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/sandbox-nextjs/src/domain"
)

type PasskeyService struct {
	webauthn *webauthn.WebAuthn
}

func NewPasskeyService(rpID string, rpOrigin string) (*PasskeyService, error) {
	// このアプリの RP 情報と認証器要件を WebAuthn ライブラリへ渡す。
	passkey, err := webauthn.New(&webauthn.Config{
		RPID:          rpID,
		RPDisplayName: "sandbox_nextjs",
		RPOrigins:     []string{rpOrigin},
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementRequired,
			UserVerification: protocol.VerificationPreferred,
		},
	})
	if err != nil {
		return nil, err
	}

	return &PasskeyService{webauthn: passkey}, nil
}

func (s *PasskeyService) BeginRegistration(user domain.Account) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
	// 登録済みの認証情報を除外し、同じ認証器の重複登録を避ける。
	return s.webauthn.BeginRegistration(
		user,
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		webauthn.WithExclusions(webauthn.Credentials(user.WebAuthnCredentials()).CredentialDescriptors()),
		webauthn.WithExtensions(map[string]any{"credProps": true}),
	)
}

func (s *PasskeyService) FinishRegistration(user domain.Account, session webauthn.SessionData, request *http.Request) (*webauthn.Credential, error) {
	return s.webauthn.FinishRegistration(user, session, request)
}

func (s *PasskeyService) BeginLogin() (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	// ユーザー名入力なしで、認証器からユーザーを解決するログインを開始する。
	return s.webauthn.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationPreferred))
}

func (s *PasskeyService) FinishLogin(handler webauthn.DiscoverableUserHandler, session webauthn.SessionData, request *http.Request) (webauthn.User, *webauthn.Credential, error) {
	return s.webauthn.FinishPasskeyLogin(handler, session, request)
}

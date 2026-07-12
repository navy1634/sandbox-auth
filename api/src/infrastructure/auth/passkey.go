package auth

import (
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"sandbox-nextjs/api/src/domain"
)

type PasskeyService struct {
	webauthn *webauthn.WebAuthn
}

func NewPasskeyService(rpID string, rpOrigin string) (*PasskeyService, error) {
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
	return s.webauthn.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationPreferred))
}

func (s *PasskeyService) FinishLogin(handler webauthn.DiscoverableUserHandler, session webauthn.SessionData, request *http.Request) (webauthn.User, *webauthn.Credential, error) {
	return s.webauthn.FinishPasskeyLogin(handler, session, request)
}

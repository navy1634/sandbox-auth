package router

import (
	"github.com/sandbox-nextjs/src/config"
	"github.com/sandbox-nextjs/src/ent"
	"github.com/sandbox-nextjs/src/handler"
	"github.com/sandbox-nextjs/src/infrastructure/auth"
	"github.com/sandbox-nextjs/src/infrastructure/oidc"
	"github.com/sandbox-nextjs/src/infrastructure/persistence"
	"github.com/sandbox-nextjs/src/usecase"
)

// 認証関連の永続化、認証、ユースケース、HTTP ハンドラを組み立てる。
func newAuthHandlers(cfg config.Config, db *ent.Client) (*handler.AuthHandler, []handler.AuthMethodHandler, *handler.OIDCHandler, error) {
	// リポジトリ
	accountRepository := persistence.NewEntAccountRepository(db)
	passkeyRepository := persistence.NewEntPasskeyRepository(db, accountRepository)
	// ユースケース
	accountUsecase := usecase.NewAccountUsecase(accountRepository)
	oauthUsecase := usecase.NewOAuthUsecase(accountRepository, map[string]auth.OAuthProvider{
		"google": auth.NewGoogleProvider(cfg.GoogleClientID, cfg.GoogleSecret, cfg.GoogleRedirectURL),
	})
	passkeyService, err := auth.NewPasskeyService(cfg.PasskeyRPID, cfg.PasskeyRPOrigin)
	if err != nil {
		return nil, nil, nil, err
	}
	passkeyUsecase := usecase.NewPasskeyUsecase(passkeyRepository, passkeyService)

	// ハンドラ
	authHandler := handler.NewAuthHandler(cfg, accountUsecase, persistence.NewEntSessionStore(db))
	oauthHandler := handler.NewOAuthHandler(authHandler, oauthUsecase)
	passkeyHandler := handler.NewPasskeyHandler(authHandler, passkeyUsecase)
	var oidcHandler *handler.OIDCHandler
	if cfg.OIDCEnabled() {
		clients := make([]oidc.Client, 0, len(cfg.OIDCClients))
		for _, client := range cfg.OIDCClients {
			clients = append(clients, oidc.Client{
				ID:           client.ClientID,
				Secret:       client.ClientSecret,
				RedirectURIs: client.RedirectURIs,
			})
		}
		provider, err := oidc.NewProvider(oidc.Config{
			IssuerURL:     cfg.OIDCIssuerURL,
			Clients:       clients,
			Store:         persistence.NewEntOIDCStore(db),
			SigningKeyPEM: cfg.OIDCSigningKeyPEM,
			KeyID:         cfg.OIDCKeyID,
		})
		if err != nil {
			return nil, nil, nil, err
		}
		oidcHandler = handler.NewOIDCHandler(authHandler, accountUsecase, provider)
	}

	return authHandler, []handler.AuthMethodHandler{oauthHandler, passkeyHandler}, oidcHandler, nil
}

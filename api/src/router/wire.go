package router

import (
	"github.com/sandbox-nextjs/src/config"
	"github.com/sandbox-nextjs/src/ent"
	"github.com/sandbox-nextjs/src/handler"
	"github.com/sandbox-nextjs/src/infrastructure/auth"
	"github.com/sandbox-nextjs/src/infrastructure/persistence"
	"github.com/sandbox-nextjs/src/usecase"
)

// 認証関連の永続化、認証、ユースケース、HTTP ハンドラを組み立てる。
func newAuthHandlers(cfg config.Config, db *ent.Client) (*handler.AuthHandler, []handler.AuthMethodHandler, error) {
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
		return nil, nil, err
	}
	passkeyUsecase := usecase.NewPasskeyUsecase(passkeyRepository, passkeyService)

	// ハンドラ
	authHandler := handler.NewAuthHandler(cfg, accountUsecase)
	oauthHandler := handler.NewOAuthHandler(authHandler, oauthUsecase)
	passkeyHandler := handler.NewPasskeyHandler(authHandler, passkeyUsecase)

	return authHandler, []handler.AuthMethodHandler{oauthHandler, passkeyHandler}, nil
}

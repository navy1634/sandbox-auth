package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-nextjs/src/config"
	"github.com/sandbox-nextjs/src/ent"
	"github.com/sandbox-nextjs/src/handler"
	"github.com/sandbox-nextjs/src/infrastructure/auth"
	"github.com/sandbox-nextjs/src/infrastructure/database"
	"github.com/sandbox-nextjs/src/infrastructure/persistence"
	"github.com/sandbox-nextjs/src/usecase"
)

func RegisterRoutes(engine *gin.Engine, cfg config.Config, db *ent.Client) error {
	// ルート登録前に、永続化、認証、ユースケース、HTTP ハンドラを組み立てる。
	accountRepository := persistence.NewEntAccountRepository(db)
	passkeyRepository := persistence.NewEntPasskeyRepository(db, accountRepository)
	accountUsecase := usecase.NewAccountUsecase(accountRepository)
	oauthUsecase := usecase.NewOAuthUsecase(accountRepository, map[string]auth.OAuthProvider{
		"google": auth.NewGoogleProvider(cfg.GoogleClientID, cfg.GoogleSecret, cfg.GoogleRedirectURL),
	})
	passkeyService, err := auth.NewPasskeyService(cfg.PasskeyRPID, cfg.PasskeyRPOrigin)
	if err != nil {
		return err
	}
	passkeyUsecase := usecase.NewPasskeyUsecase(passkeyRepository, passkeyService)
	authHandler := handler.NewAuthHandler(cfg, accountUsecase)
	oauthHandler := handler.NewOAuthHandler(authHandler, oauthUsecase)
	passkeyHandler := handler.NewPasskeyHandler(authHandler, passkeyUsecase)

	engine.Use(corsMiddleware(cfg.CORSOrigins()...))
	engine.Use(database.TransactionMiddleware(db))

	engine.GET("/health", authHandler.Health)
	engine.GET("/me", authHandler.Me)
	engine.POST("/account/profile", authHandler.UpdateProfile)
	engine.POST("/auth/logout", authHandler.Logout)

	for _, authMethod := range []handler.AuthMethodHandler{oauthHandler, passkeyHandler} {
		authMethod.RegisterRoutes(engine)
	}

	return nil
}

func corsMiddleware(allowedOrigins ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = allowedOrigins[0]
		}
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				c.Header("Access-Control-Allow-Origin", origin)
				break
			}
		}
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

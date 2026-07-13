package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-nextjs/src/config"
	"github.com/sandbox-nextjs/src/ent"
	"github.com/sandbox-nextjs/src/handler"
	"github.com/sandbox-nextjs/src/infrastructure/database"
	"github.com/sandbox-nextjs/src/infrastructure/persistence"
)

func RegisterRoutes(engine *gin.Engine, cfg config.Config, db *ent.Client) error {
	accountRepository := persistence.NewEntAccountRepository(db)
	passkeyRepository := persistence.NewEntPasskeyRepository(db, accountRepository)
	authHandler := handler.NewAuthHandler(cfg, accountRepository)
	oauthHandler := handler.NewOAuthHandler(cfg, authHandler)
	passkeyHandler, err := handler.NewPasskeyHandler(cfg, authHandler, passkeyRepository)
	if err != nil {
		return err
	}

	engine.Use(corsMiddleware(cfg.FrontendURL))
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

func corsMiddleware(frontendURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", frontendURL)
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

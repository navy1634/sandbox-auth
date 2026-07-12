package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sandbox-nextjs/api/src/config"
	"sandbox-nextjs/api/src/ent"
	"sandbox-nextjs/api/src/handler"
	"sandbox-nextjs/api/src/repository"
)

func RegisterRoutes(engine *gin.Engine, cfg config.Config, db *ent.Client) error {
	authHandler, err := handler.NewAuthHandler(cfg, repository.NewAccountRepository(db))
	if err != nil {
		return err
	}

	engine.Use(corsMiddleware(cfg.FrontendURL))

	engine.GET("/health", authHandler.Health)
	engine.GET("/auth/google/login", authHandler.GoogleLogin)
	engine.GET("/auth/google/callback", authHandler.GoogleCallback)
	engine.GET("/me", authHandler.Me)
	engine.POST("/account/profile", authHandler.UpdateProfile)
	engine.POST("/passkeys/register/options", authHandler.BeginPasskeyRegistration)
	engine.POST("/passkeys/register/verify", authHandler.FinishPasskeyRegistration)
	engine.POST("/passkeys/login/options", authHandler.BeginPasskeyLogin)
	engine.POST("/passkeys/login/verify", authHandler.FinishPasskeyLogin)
	engine.POST("/auth/logout", authHandler.Logout)

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

package router

import (
	"github.com/gin-gonic/gin"
	"github.com/sandbox-nextjs/src/config"
	"github.com/sandbox-nextjs/src/ent"
	"github.com/sandbox-nextjs/src/handler"
)

// 認証関連の HTTP ルートと middleware を Gin engine に登録する。
func RegisterRoutes(engine *gin.Engine, cfg config.Config, db *ent.Client) error {
	authHandler, authMethods, err := newAuthHandlers(cfg, db)
	if err != nil {
		return err
	}

	registerMiddleware(engine, cfg, db)
	registerAuthRoutes(engine, authHandler, authMethods)

	return nil
}

// 認証関連のルートを登録する。
func registerAuthRoutes(engine *gin.Engine, authHandler *handler.AuthHandler, authMethods []handler.AuthMethodHandler) {
	engine.GET("/health", authHandler.Health)
	engine.GET("/me", authHandler.Me)
	engine.POST("/auth/logout", authHandler.Logout)

	for _, authMethod := range authMethods {
		authMethod.RegisterRoutes(engine)
	}
}

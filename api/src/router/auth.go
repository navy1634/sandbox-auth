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
	registerAuthRoutesOn(engine, authHandler, authMethods)
	registerAuthRoutesOn(engine.Group("/api"), authHandler, authMethods)
}

func registerAuthRoutesOn(routes gin.IRoutes, authHandler *handler.AuthHandler, authMethods []handler.AuthMethodHandler) {
	routes.GET("/health", authHandler.Health)
	routes.GET("/me", authHandler.Me)
	routes.POST("/auth/logout", authHandler.Logout)

	for _, authMethod := range authMethods {
		authMethod.RegisterRoutes(routes)
	}
}

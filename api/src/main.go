package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-auth/src/config"
	"github.com/sandbox-auth/src/infrastructure/database"
	"github.com/sandbox-auth/src/router"
)

func main() {
	// 設定、DB、ルーティングを組み立てて API サーバーを起動する。
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	}()

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	// 信頼するプロキシ CIDR を設定し、空なら forwarded header を使わない。
	if err := engine.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Fatal(err)
	}
	// この logger では、アクセスログにクエリ文字列を出さない。
	engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{SkipQueryString: true}))
	engine.Use(gin.CustomRecoveryWithWriter(gin.DefaultErrorWriter, func(c *gin.Context, err any) {
		log.Printf("panic recovered: %v", err)
		c.AbortWithStatus(http.StatusInternalServerError)
	}))
	if err := router.RegisterRoutes(engine, cfg, db); err != nil {
		log.Fatal(err)
	}

	// 読み書きの timeout を明示する。
	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

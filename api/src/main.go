package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-nextjs/src/config"
	"github.com/sandbox-nextjs/src/infrastructure/database"
	"github.com/sandbox-nextjs/src/router"
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

	gin.SetMode(config.GetEnv("GIN_MODE", gin.DebugMode))
	engine := gin.Default()
	if err := router.RegisterRoutes(engine, cfg, db); err != nil {
		log.Fatal(err)
	}

	if err := engine.Run(cfg.Addr); err != nil {
		log.Fatal(err)
	}
}

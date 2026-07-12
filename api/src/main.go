package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"sandbox-nextjs/api/src/config"
	"sandbox-nextjs/api/src/infrastructure/database"
	"sandbox-nextjs/api/src/router"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	gin.SetMode(config.GetEnv("GIN_MODE", gin.DebugMode))
	engine := gin.Default()
	if err := router.RegisterRoutes(engine, cfg, db); err != nil {
		log.Fatal(err)
	}

	if err := engine.Run(cfg.Addr); err != nil {
		log.Fatal(err)
	}
}

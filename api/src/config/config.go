package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	Addr              string
	FrontendURL       string
	GoogleClientID    string
	GoogleSecret      string
	GoogleRedirectURL string
	DatabaseURL       string
	PasskeyRPID       string
	PasskeyRPOrigin   string
	SessionSecret     []byte
}

func Load() (Config, error) {
	// 環境変数から API の起動設定と外部サービス設定を読み込む。
	cfg := Config{
		Addr:              GetEnv("ADDR", ":8080"),
		FrontendURL:       strings.TrimRight(GetEnv("FRONTEND_URL", "http://localhost:3000"), "/"),
		GoogleClientID:    os.Getenv("GOOGLE_ID"),
		GoogleSecret:      os.Getenv("GOOGLE_SECRET"),
		GoogleRedirectURL: GetEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		PasskeyRPID:       GetEnv("PASSKEY_RP_ID", "localhost"),
		PasskeyRPOrigin:   GetEnv("PASSKEY_RP_ORIGIN", "http://localhost:3000"),
		SessionSecret:     []byte(os.Getenv("AUTH_SECRET")),
	}

	if cfg.GoogleClientID == "" || cfg.GoogleSecret == "" {
		return cfg, errors.New("GOOGLE_ID and GOOGLE_SECRET are required")
	}
	// Cookie セッションの署名に使うため、短すぎるシークレットは拒否する。
	if len(cfg.SessionSecret) < 32 {
		return cfg, errors.New("AUTH_SECRET must be at least 32 bytes")
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL is required")
	}

	return cfg, nil
}

func GetEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

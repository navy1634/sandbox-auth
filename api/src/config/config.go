package config

import (
	"errors"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Addr                string
	FrontendURL         string
	DefaultRedirectURL  string
	AllowedRedirectURLs []string
	GoogleClientID      string
	GoogleSecret        string
	GoogleRedirectURL   string
	DatabaseURL         string
	PasskeyRPID         string
	PasskeyRPOrigin     string
	SessionSecret       []byte
}

func Load() (Config, error) {
	frontendURL := strings.TrimRight(GetEnv("FRONTEND_URL", "http://localhost:3000"), "/")
	defaultRedirectURL := strings.TrimRight(GetEnv("DEFAULT_REDIRECT_URL", frontendURL+"/mypage"), "/")

	// 環境変数から API の起動設定と外部サービス設定を読み込む。
	cfg := Config{
		Addr:                GetEnv("ADDR", ":8080"),
		FrontendURL:         frontendURL,
		DefaultRedirectURL:  defaultRedirectURL,
		AllowedRedirectURLs: splitURLs(GetEnv("ALLOWED_REDIRECT_URLS", defaultRedirectURL)),
		GoogleClientID:      os.Getenv("GOOGLE_ID"),
		GoogleSecret:        os.Getenv("GOOGLE_SECRET"),
		GoogleRedirectURL:   GetEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		PasskeyRPID:         GetEnv("PASSKEY_RP_ID", "localhost"),
		PasskeyRPOrigin:     GetEnv("PASSKEY_RP_ORIGIN", "http://localhost:3000"),
		SessionSecret:       []byte(os.Getenv("AUTH_SECRET")),
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
	if cfg.DefaultRedirectURL == "" {
		return cfg, errors.New("DEFAULT_REDIRECT_URL is required")
	}
	if len(cfg.AllowedRedirectURLs) == 0 {
		return cfg, errors.New("ALLOWED_REDIRECT_URLS must include at least one URL")
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

func splitURLs(raw string) []string {
	values := strings.Split(raw, ",")
	urls := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimRight(strings.TrimSpace(value), "/")
		if trimmed != "" {
			urls = append(urls, trimmed)
		}
	}
	return urls
}

func (cfg Config) AuthRedirectURL(raw string) string {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return cfg.DefaultRedirectURL
	}
	redirectURL, ok := normalizeRedirectURL(candidate, cfg.DefaultRedirectURL)
	if !ok {
		return cfg.DefaultRedirectURL
	}
	for _, allowedURL := range cfg.AllowedRedirectURLs {
		if redirectAllowed(redirectURL, allowedURL) {
			return redirectURL
		}
	}
	return cfg.DefaultRedirectURL
}

func (cfg Config) CORSOrigins() []string {
	origins := []string{originOf(cfg.FrontendURL)}
	for _, redirectURL := range cfg.AllowedRedirectURLs {
		origins = append(origins, originOf(redirectURL))
	}
	return compactStrings(origins)
}

func normalizeRedirectURL(raw string, applicationURL string) (string, bool) {
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		baseURL, err := url.Parse(applicationURL)
		if err != nil {
			return "", false
		}
		relativeURL, err := url.Parse(raw)
		if err != nil {
			return "", false
		}
		return baseURL.ResolveReference(relativeURL).String(), true
	}

	parsedURL, err := url.Parse(raw)
	if err != nil || !parsedURL.IsAbs() || parsedURL.Host == "" || parsedURL.User != nil {
		return "", false
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", false
	}
	return parsedURL.String(), true
}

func redirectAllowed(rawTargetURL string, rawAllowedURL string) bool {
	targetURL, targetErr := url.Parse(rawTargetURL)
	allowedURL, allowedErr := url.Parse(rawAllowedURL)
	if targetErr != nil || allowedErr != nil {
		return false
	}
	if !strings.EqualFold(targetURL.Scheme, allowedURL.Scheme) || !strings.EqualFold(targetURL.Host, allowedURL.Host) {
		return false
	}

	allowedPath := strings.TrimRight(allowedURL.EscapedPath(), "/")
	targetPath := strings.TrimRight(targetURL.EscapedPath(), "/")
	if allowedPath == "" {
		return true
	}
	return targetPath == allowedPath || strings.HasPrefix(targetPath, allowedPath+"/")
}

func originOf(raw string) string {
	parsedURL, err := url.Parse(raw)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return ""
	}
	return parsedURL.Scheme + "://" + parsedURL.Host
}

func compactStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

package config

import (
	"errors"
	"net"
	"net/url"
	"os"
	"strings"

	"golang.org/x/net/publicsuffix"
)

type Config struct {
	Addr                string
	FrontendURL         string
	DefaultRedirectURL  string
	AllowedRedirectURLs []string
	CORSAllowedOrigins  []string
	CookieDomain        string
	GoogleClientID      string
	GoogleSecret        string
	GoogleRedirectURL   string
	DatabaseURL         string
	PasskeyRPID         string
	PasskeyRPOrigin     string
	SessionSecret       []byte
	TrustedProxies      []string
}

// 環境変数から API 設定を読み込み、必須値を検証する。
func Load() (Config, error) {
	frontendURL := strings.TrimRight(GetEnv("FRONTEND_URL", "http://localhost:3000"), "/")
	defaultRedirectURL := strings.TrimRight(GetEnv("DEFAULT_REDIRECT_URL", frontendURL+"/mypage"), "/")
	cookieDomain, err := normalizeCookieDomain(frontendURL, os.Getenv("COOKIE_DOMAIN"))
	if err != nil {
		return Config{}, err
	}
	databaseURL, err := databaseURLFromEnv()
	if err != nil {
		return Config{}, err
	}

	// 環境変数から API の起動設定と外部サービス設定を読み込む。
	cfg := Config{
		Addr:                GetEnv("ADDR", ":8080"),
		FrontendURL:         frontendURL,
		DefaultRedirectURL:  defaultRedirectURL,
		AllowedRedirectURLs: splitURLs(GetEnv("ALLOWED_REDIRECT_URLS", defaultRedirectURL)),
		CORSAllowedOrigins:  splitOrigins(GetEnv("CORS_ALLOWED_ORIGINS", originOf(frontendURL))),
		CookieDomain:        cookieDomain,
		GoogleClientID:      os.Getenv("GOOGLE_ID"),
		GoogleSecret:        os.Getenv("GOOGLE_SECRET"),
		GoogleRedirectURL:   GetEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback"),
		DatabaseURL:         databaseURL,
		PasskeyRPID:         GetEnv("PASSKEY_RP_ID", "localhost"),
		PasskeyRPOrigin:     GetEnv("PASSKEY_RP_ORIGIN", "http://localhost:3000"),
		SessionSecret:       []byte(os.Getenv("AUTH_SECRET")),
		TrustedProxies:      splitStrings(GetEnv("TRUSTED_PROXIES", "")),
	}

	if cfg.GoogleClientID == "" || cfg.GoogleSecret == "" {
		return cfg, errors.New("GOOGLE_ID and GOOGLE_SECRET are required")
	}
	// Cookie セッションの署名に使うため、短すぎるシークレットは拒否する。
	if len(cfg.SessionSecret) < 32 {
		return cfg, errors.New("AUTH_SECRET must be at least 32 bytes")
	}
	if cfg.DefaultRedirectURL == "" {
		return cfg, errors.New("DEFAULT_REDIRECT_URL is required")
	}
	if len(cfg.AllowedRedirectURLs) == 0 {
		return cfg, errors.New("ALLOWED_REDIRECT_URLS must include at least one URL")
	}
	if len(cfg.CORSAllowedOrigins) == 0 {
		return cfg, errors.New("CORS_ALLOWED_ORIGINS must include at least one origin")
	}

	return cfg, nil
}

// Cookie の共有先を frontend host 配下の public suffix ではないドメインに制限する。
func normalizeCookieDomain(frontendURL string, raw string) (string, error) {
	domain := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(raw)), ".")
	if domain == "" {
		return "", nil
	}
	parsedFrontendURL, err := url.Parse(frontendURL)
	if err != nil || parsedFrontendURL.Hostname() == "" {
		return "", errors.New("FRONTEND_URL must include a valid host")
	}
	host := strings.ToLower(parsedFrontendURL.Hostname())
	if host != domain && !strings.HasSuffix(host, "."+domain) {
		return "", errors.New("COOKIE_DOMAIN must be a parent of FRONTEND_URL host")
	}
	registrableDomain, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil || registrableDomain == "" {
		return "", errors.New("COOKIE_DOMAIN must not be a public suffix")
	}
	return domain, nil
}

// DB 接続用の環境変数から PostgreSQL 接続 URL を組み立てる。
func databaseURLFromEnv() (string, error) {
	host := GetEnv("DB_HOST", "localhost")
	port := GetEnv("DB_PORT", "5432")
	name := os.Getenv("DB_NAME")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	sslmode := GetEnv("DB_SSLMODE", "disable")

	if name == "" {
		return "", errors.New("DB_NAME is required")
	}
	if user == "" {
		return "", errors.New("DB_USER is required")
	}
	if password == "" {
		return "", errors.New("DB_PASSWORD is required")
	}

	query := url.Values{}
	query.Set("sslmode", sslmode)
	databaseURL := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, password),
		Host:     net.JoinHostPort(host, port),
		Path:     "/" + name,
		RawQuery: query.Encode(),
	}
	return databaseURL.String(), nil
}

// 環境変数が空なら fallback を返す。
func GetEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// カンマ区切りの URL 文字列を空要素なしのリストへ変換する。
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

// カンマ区切りの文字列を空要素なしのリストへ変換する。
func splitStrings(raw string) []string {
	values := strings.Split(raw, ",")
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// 指定された戻り先 URL が許可済みならその URL を返す。
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

// credentialed CORS で許可する Origin の重複を除いて返す。
func (cfg Config) CORSOrigins() []string {
	return compactStrings(cfg.CORSAllowedOrigins)
}

// カンマ区切りの URL 文字列から Origin だけを取り出す。
func splitOrigins(raw string) []string {
	values := strings.Split(raw, ",")
	origins := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimRight(strings.TrimSpace(value), "/")
		if trimmed == "" {
			continue
		}
		origin := originOf(trimmed)
		if origin == trimmed {
			origins = append(origins, origin)
		}
	}
	return origins
}

// 相対 URL と絶対 URL を検証済みの絶対 URL へ正規化する。
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

// 戻り先 URL が許可 URL の scheme、host、path 条件を満たすかを返す。
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

// URL 文字列から scheme と host だけを取り出す。
func originOf(raw string) string {
	parsedURL, err := url.Parse(raw)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return ""
	}
	return parsedURL.Scheme + "://" + parsedURL.Host
}

// 空文字と重複を取り除いた文字列リストを返す。
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

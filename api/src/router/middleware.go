package router

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-nextjs/src/config"
	"github.com/sandbox-nextjs/src/ent"
	"github.com/sandbox-nextjs/src/infrastructure/database"
)

// API 全体へ適用する middleware を登録する。
func registerMiddleware(engine *gin.Engine, cfg config.Config, db *ent.Client) {
	engine.Use(corsMiddleware(cfg.CORSOrigins()...))
	engine.Use(originMiddleware(cfg.CORSOrigins()...))
	engine.Use(rateLimitMiddleware(map[string]rateLimitRule{
		"/passkeys/login/options": {
			Limit:  20,
			Window: time.Minute,
		},
	}))
	engine.Use(maxBodyBytesMiddleware(1 << 20))
	engine.Use(database.TransactionMiddleware(db))
}

type rateLimitRule struct {
	Limit  int
	Window time.Duration
}

type rateLimitStore struct {
	mu       sync.Mutex
	requests map[string][]time.Time
}

// 指定したルートごとのレート制限を適用する。
func rateLimitMiddleware(rules map[string]rateLimitRule) gin.HandlerFunc {
	store := &rateLimitStore{requests: map[string][]time.Time{}}
	return func(c *gin.Context) {
		rule, ok := rules[c.FullPath()]
		if !ok {
			c.Next()
			return
		}

		// この処理では、ClientIP とルートパスを rate limit のキーにする。
		if !store.allow(c.ClientIP()+" "+c.FullPath(), rule, time.Now()) {
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		c.Next()
	}
}

// 指定キーのリクエスト時刻を記録し、上限内なら true を返す。
func (s *rateLimitStore) allow(key string, rule rateLimitRule, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := now.Add(-rule.Window)
	// この処理では、window より古いキーを map から消す。
	for storedKey, timestamps := range s.requests {
		if len(timestamps) == 0 || timestamps[len(timestamps)-1].Before(cutoff) {
			delete(s.requests, storedKey)
		}
	}

	timestamps := s.requests[key]
	kept := timestamps[:0]
	for _, timestamp := range timestamps {
		if timestamp.After(cutoff) {
			kept = append(kept, timestamp)
		}
	}
	if len(kept) >= rule.Limit {
		s.requests[key] = kept
		return false
	}
	s.requests[key] = append(kept, now)
	return true
}

// 状態変更リクエストの Origin が許可リストにあるかを検証する。
func originMiddleware(allowedOrigins ...string) gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		if _, ok := allowed[c.GetHeader("Origin")]; !ok {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

// リクエスト本文の読み取りサイズを指定バイト数に制限する。
func maxBodyBytesMiddleware(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}

// 許可済み Origin だけを credentialed CORS の応答ヘッダーへ反映する。
func corsMiddleware(allowedOrigins ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = allowedOrigins[0]
		}
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				c.Header("Access-Control-Allow-Origin", origin)
				break
			}
		}
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

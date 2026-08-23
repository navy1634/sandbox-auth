package router

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// 本文サイズ上限を超えたリクエストが 413 になることを確認する。
func TestMaxBodyBytesMiddlewareLimitsRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(maxBodyBytesMiddleware(4))
	engine.POST("/body", func(c *gin.Context) {
		_, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/body", strings.NewReader("12345"))
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

// 許可リスト外の Origin を持つ POST が 403 になることを確認する。
func TestOriginMiddlewareRejectsUntrustedPostOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(originMiddleware("https://app.example.com"))
	engine.POST("/logout", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

// 許可リスト内の Origin を持つ POST が通ることを確認する。
func TestOriginMiddlewareAllowsTrustedPostOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(originMiddleware("https://app.example.com"))
	engine.POST("/logout", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// OIDC token endpointはサーバー間POSTのためOriginなしでも通ることを確認する。
func TestOriginMiddlewareAllowsOIDCTokenWithoutOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(originMiddleware("https://app.example.com"))
	engine.POST("/oidc/token", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/oidc/token", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// 同じ送信元から上限を超えて送ると 429 になることを確認する。
func TestRateLimitMiddlewareRejectsRequestsOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(rateLimitMiddleware(map[string]rateLimitRule{
		"/passkeys/login/options": {
			Limit:  1,
			Window: time.Minute,
		},
	}))
	engine.POST("/passkeys/login/options", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// ここでは同じ RemoteAddr から 2 回連続でリクエストを送る。
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/passkeys/login/options", nil)
		req.RemoteAddr = "192.0.2.10:12345"
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if i == 0 && rec.Code != http.StatusOK {
			t.Fatalf("first status = %d, want %d", rec.Code, http.StatusOK)
		}
		if i == 1 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("second status = %d, want %d", rec.Code, http.StatusTooManyRequests)
		}
	}
}

// 信頼プロキシなしでは X-Forwarded-For ではなく RemoteAddr で制限されることを確認する。
func TestRateLimitMiddlewareIgnoresSpoofedForwardedForWithoutTrustedProxies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	if err := engine.SetTrustedProxies(nil); err != nil {
		t.Fatalf("set trusted proxies: %v", err)
	}
	engine.Use(rateLimitMiddleware(map[string]rateLimitRule{
		"/passkeys/login/options": {
			Limit:  1,
			Window: time.Minute,
		},
	}))
	engine.POST("/passkeys/login/options", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// ここでは RemoteAddr は同じまま、X-Forwarded-For だけを変えて 2 回送る。
	for i, forwardedFor := range []string{"198.51.100.1", "198.51.100.2"} {
		req := httptest.NewRequest(http.MethodPost, "/passkeys/login/options", nil)
		req.RemoteAddr = "192.0.2.10:12345"
		req.Header.Set("X-Forwarded-For", forwardedFor)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if i == 0 && rec.Code != http.StatusOK {
			t.Fatalf("first status = %d, want %d", rec.Code, http.StatusOK)
		}
		if i == 1 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("second status = %d, want %d", rec.Code, http.StatusTooManyRequests)
		}
	}
}

// 信頼プロキシ経由では X-Forwarded-For の利用者 IP ごとに制限されることを確認する。
func TestRateLimitMiddlewareUsesForwardedForFromTrustedProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	if err := engine.SetTrustedProxies([]string{"192.0.2.10"}); err != nil {
		t.Fatalf("set trusted proxies: %v", err)
	}
	engine.Use(rateLimitMiddleware(map[string]rateLimitRule{
		"/passkeys/login/options": {
			Limit:  1,
			Window: time.Minute,
		},
	}))
	engine.POST("/passkeys/login/options", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// ここでは同じプロキシから、別の利用者 IP として 2 回送る。
	for _, forwardedFor := range []string{"198.51.100.1", "198.51.100.2"} {
		req := httptest.NewRequest(http.MethodPost, "/passkeys/login/options", nil)
		req.RemoteAddr = "192.0.2.10:12345"
		req.Header.Set("X-Forwarded-For", forwardedFor)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	}
}

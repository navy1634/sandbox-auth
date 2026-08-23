package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-auth/src/config"
)

func TestCookieNameUsesHostPrefixForSecureCookies(t *testing.T) {
	handler := NewAuthHandler(config.Config{FrontendURL: "https://app.example.com"}, nil)

	got := handler.cookieName(stateCookieName)
	if got != hostStateCookieName {
		t.Fatalf("cookieName() = %q, want %q", got, hostStateCookieName)
	}
}

func TestCookieNameKeepsLocalNameForInsecureCookies(t *testing.T) {
	handler := NewAuthHandler(config.Config{FrontendURL: "http://localhost:3000"}, nil)

	got := handler.cookieName(stateCookieName)
	if got != stateCookieName {
		t.Fatalf("cookieName() = %q, want %q", got, stateCookieName)
	}
}

func TestSessionCookieUsesSharedDomainWithoutHostPrefix(t *testing.T) {
	handler := NewAuthHandler(config.Config{
		FrontendURL:  "https://auth.sandbox.example.com",
		CookieDomain: ".sandbox.example.com",
	}, nil)

	if got := handler.cookieName(sessionCookieName); got != sessionCookieName {
		t.Fatalf("cookieName() = %q, want %q", got, sessionCookieName)
	}
	if got := handler.cookieName(stateCookieName); got != hostStateCookieName {
		t.Fatalf("cookieName() = %q, want %q", got, hostStateCookieName)
	}
}

func TestSessionCookieIsSharedAcrossConfiguredSubdomains(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAuthHandler(config.Config{
		FrontendURL:  "https://auth.sandbox.example.com",
		CookieDomain: ".sandbox.example.com",
	}, nil)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	handler.setCookie(context, sessionCookieName, "signed-session", 3600, true)

	response := recorder.Result()
	cookies := response.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("len(cookies) = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != sessionCookieName {
		t.Fatalf("cookie.Name = %q, want %q", cookie.Name, sessionCookieName)
	}
	if cookie.Domain != "sandbox.example.com" {
		t.Fatalf("cookie.Domain = %q, want %q", cookie.Domain, "sandbox.example.com")
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie security attributes = HttpOnly:%v Secure:%v SameSite:%v", cookie.HttpOnly, cookie.Secure, cookie.SameSite)
	}
}

func TestOAuthStateCookieRemainsHostOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAuthHandler(config.Config{
		FrontendURL:  "https://auth.sandbox.example.com",
		CookieDomain: ".sandbox.example.com",
	}, nil)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	handler.setCookie(context, stateCookieName, "state", 300, true)

	cookie := recorder.Result().Cookies()[0]
	if cookie.Name != hostStateCookieName {
		t.Fatalf("cookie.Name = %q, want %q", cookie.Name, hostStateCookieName)
	}
	if cookie.Domain != "" {
		t.Fatalf("cookie.Domain = %q, want empty", cookie.Domain)
	}
}

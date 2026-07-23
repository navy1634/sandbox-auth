package handler

import (
	"testing"

	"github.com/sandbox-nextjs/src/config"
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

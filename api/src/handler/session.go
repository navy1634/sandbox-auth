package handler

import (
	"crypto/hmac"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"sandbox-nextjs/api/src/infrastructure/session"
)

func (h *AuthHandler) validateState(c *gin.Context) error {
	expected, err := c.Cookie(stateCookieName)
	if err != nil {
		return err
	}
	if expected == "" || !hmac.Equal([]byte(expected), []byte(c.Query("state"))) {
		return errors.New("state mismatch")
	}
	return nil
}

func (h *AuthHandler) readSession(c *gin.Context) (session.User, error) {
	value, err := c.Cookie(sessionCookieName)
	if err != nil {
		return session.User{}, err
	}
	return h.sessions.Verify(value)
}

func (h *AuthHandler) setCookie(c *gin.Context, name string, value string, maxAge int, httpOnly bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, value, maxAge, "/", "", isHTTPS(h.cfg.FrontendURL), httpOnly)
}

func (h *AuthHandler) clearCookie(c *gin.Context, name string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, "", -1, "/", "", isHTTPS(h.cfg.FrontendURL), true)
}

func isHTTPS(rawURL string) bool {
	return strings.HasPrefix(rawURL, "https://")
}

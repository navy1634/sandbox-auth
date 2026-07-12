package session

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"sandbox-nextjs/api/src/domain"
)

type User struct {
	AccountID         int64  `json:"accountId"`
	Provider          string `json:"provider"`
	ProviderAccountID string `json:"providerAccountId"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	Picture           string `json:"picture"`
}

type Manager struct {
	secret []byte
}

func NewManager(secret []byte) *Manager {
	return &Manager{secret: secret}
}

func (m *Manager) Sign(user User) (string, error) {
	payload, err := json.Marshal(user)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(encodedPayload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return encodedPayload + "." + signature, nil
}

func (m *Manager) Verify(value string) (User, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return User{}, errors.New("invalid session format")
	}

	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)

	actual, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return User{}, err
	}
	if !hmac.Equal(expected, actual) {
		return User{}, errors.New("invalid session signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return User{}, err
	}

	var user User
	if err := json.Unmarshal(payload, &user); err != nil {
		return User{}, err
	}

	return user, nil
}

func FromAccount(account domain.Account) User {
	return User{
		AccountID:         account.ID,
		Provider:          account.Provider,
		ProviderAccountID: account.ProviderAccountID,
		Email:             account.Email,
		Name:              account.Name,
		Picture:           account.Picture,
	}
}

func RandomString(size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

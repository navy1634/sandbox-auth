package session

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/sandbox-nextjs/src/domain"
)

type User struct {
	AccountID         int64  `json:"accountId"`
	Provider          string `json:"provider,omitempty"`
	ProviderAccountID string `json:"providerAccountId,omitempty"`
	Email             string `json:"email,omitempty"`
	Name              string `json:"name,omitempty"`
	Picture           string `json:"picture,omitempty"`
}

type Manager struct {
	secret []byte
}

func NewManager(secret []byte) *Manager {
	return &Manager{secret: secret}
}

func (m *Manager) Sign(user User) (string, error) {
	// セッション本文を URL 安全な文字列にし、HMAC 署名を付けて改ざんを検出できる形にする。
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
	// Cookie の署名を検証してから、セッション本文を復元する。
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
	// Cookie に入れるセッション情報だけをアカウントから取り出す。
	user := User{AccountID: account.ID}
	if account.Identity != nil {
		user.Provider = account.Identity.Provider
		user.ProviderAccountID = account.Identity.ProviderAccountID
		user.Email = account.Identity.Email
		user.Name = account.Identity.Name
		user.Picture = account.Identity.Picture
	}
	return user
}

func RandomString(size int) (string, error) {
	// 状態値やセッション ID に使うランダムな URL 安全文字列を作る。
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

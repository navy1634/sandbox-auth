package session

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/sandbox-nextjs/src/domain"
)

const DefaultTTL = 24 * time.Hour

type User struct {
	AccountID         int64  `json:"accountId"`
	Provider          string `json:"provider,omitempty"`
	ProviderAccountID string `json:"providerAccountId,omitempty"`
	Email             string `json:"email,omitempty"`
}

type signedUser struct {
	User
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	JTI       string `json:"jti"`
}

type Manager struct {
	secret  []byte
	revoked map[string]int64
	mu      sync.Mutex
}

// 署名と検証に使う session Manager を作る。
func NewManager(secret []byte) *Manager {
	return &Manager{
		secret:  secret,
		revoked: map[string]int64{},
	}
}

// ユーザー情報を期限付きの署名済み Cookie 値へ変換する。
func (m *Manager) Sign(user User) (string, error) {
	return m.signAt(user, time.Now())
}

// 指定時刻を発行時刻として署名済み Cookie 値を作る。
func (m *Manager) signAt(user User, now time.Time) (string, error) {
	// セッション本文を URL 安全な文字列にし、HMAC 署名を付けて改ざんを検出できる形にする。
	jti, err := RandomString(32)
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(signedUser{
		User:      user,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(DefaultTTL).Unix(),
		JTI:       jti,
	})
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(encodedPayload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return encodedPayload + "." + signature, nil
}

// 署名済み Cookie 値を検証してユーザー情報を返す。
func (m *Manager) Verify(value string) (User, error) {
	// Cookie の署名を検証してから、セッション本文を復元する。
	signed, err := m.verifySigned(value)
	if err != nil {
		return User{}, err
	}
	return signed.User, nil
}

// 署名済みセッション Cookie の JTI を失効済みとして記録する。
func (m *Manager) Revoke(value string) error {
	signed, err := m.verifySigned(value)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	// この処理では、期限切れの失効情報を消してから JTI を revoke 済みにする。
	m.pruneRevoked(time.Now().Unix())
	m.revoked[signed.JTI] = signed.ExpiresAt
	return nil
}

// セッション Cookie の署名、有効期限、失効状態を検証する。
func (m *Manager) verifySigned(value string) (signedUser, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return signedUser{}, errors.New("invalid session format")
	}

	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)

	actual, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return signedUser{}, err
	}
	if !hmac.Equal(expected, actual) {
		return signedUser{}, errors.New("invalid session signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return signedUser{}, err
	}

	var signed signedUser
	if err := json.Unmarshal(payload, &signed); err != nil {
		return signedUser{}, err
	}
	if signed.ExpiresAt <= time.Now().Unix() {
		return signedUser{}, errors.New("session expired")
	}
	if signed.JTI == "" {
		return signedUser{}, errors.New("session id is missing")
	}

	// この処理では、denylist にある JTI を revoked として扱う。
	if m.isRevoked(signed.JTI, time.Now().Unix()) {
		return signedUser{}, errors.New("session revoked")
	}

	return signed, nil
}

// 指定した JTI が失効済みとして記録されているかを返す。
func (m *Manager) isRevoked(jti string, now int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	// この処理では、検証時にも期限切れの失効情報を消す。
	m.pruneRevoked(now)

	_, ok := m.revoked[jti]
	return ok
}

// 期限切れの失効情報を denylist から削除する。
func (m *Manager) pruneRevoked(now int64) {
	for jti, expiresAt := range m.revoked {
		if expiresAt <= now {
			delete(m.revoked, jti)
		}
	}
}

// Cookie に保存するユーザー情報をアカウントから作る。
func FromAccount(account domain.Account) User {
	// Cookie に入れるセッション情報だけをアカウントから取り出す。
	user := User{AccountID: account.ID}
	if account.Identity != nil {
		user.Provider = account.Identity.Provider
		user.ProviderAccountID = account.Identity.ProviderAccountID
		user.Email = account.Identity.Email
	}
	return user
}

// 指定バイト数の乱数から URL safe な文字列を作る。
func RandomString(size int) (string, error) {
	// 状態値やセッション ID に使うランダムな URL 安全文字列を作る。
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

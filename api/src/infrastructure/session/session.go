package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"github.com/sandbox-auth/src/domain"
)

const DefaultTTL = 24 * time.Hour

type User struct {
	AccountID         int64  `json:"accountId"`
	Provider          string `json:"provider,omitempty"`
	ProviderAccountID string `json:"providerAccountId,omitempty"`
	Email             string `json:"email,omitempty"`
}

var ErrInvalidSession = errors.New("invalid session")

type Store interface {
	Save(ctx context.Context, token string, user User, expiresAt time.Time) error
	Find(ctx context.Context, token string, now time.Time) (User, error)
	Revoke(ctx context.Context, token string, now time.Time) error
}

type Manager struct {
	store Store
}

// Storeを使うsession Managerを作る。
func NewManager(stores ...Store) *Manager {
	store := Store(NewMemoryStore())
	if len(stores) > 0 && stores[0] != nil {
		store = stores[0]
	}
	return &Manager{store: store}
}

// ユーザー情報を期限付きのランダムな Cookie 値へ変換する。
func (m *Manager) Sign(user User) (string, error) {
	return m.SignContext(context.Background(), user)
}

// 指定時刻を発行時刻としてランダムな Cookie 値を作る。
func (m *Manager) signAt(user User, now time.Time) (string, error) {
	return m.signAtContext(context.Background(), user, now)
}

func (m *Manager) SignContext(ctx context.Context, user User) (string, error) {
	return m.signAtContext(ctx, user, time.Now())
}

func (m *Manager) signAtContext(ctx context.Context, user User, now time.Time) (string, error) {
	// Cookie には高エントロピーのトークンだけを置き、セッション本体は Store に保存する。
	token, err := RandomString(32)
	if err != nil {
		return "", err
	}
	if err := m.store.Save(ctx, token, user, now.Add(DefaultTTL)); err != nil {
		return "", err
	}
	return token, nil
}

// ランダムな Cookie 値を検証してユーザー情報を返す。
func (m *Manager) Verify(value string) (User, error) {
	return m.VerifyContext(context.Background(), value)
}

func (m *Manager) VerifyContext(ctx context.Context, value string) (User, error) {
	if value == "" {
		return User{}, ErrInvalidSession
	}
	return m.store.Find(ctx, value, time.Now())
}

// セッション Cookie を失効済みとして記録する。
func (m *Manager) Revoke(value string) error {
	return m.RevokeContext(context.Background(), value)
}

func (m *Manager) RevokeContext(ctx context.Context, value string) error {
	if value == "" {
		return ErrInvalidSession
	}
	return m.store.Revoke(ctx, value, time.Now())
}

type MemoryStore struct {
	mu       sync.Mutex
	sessions map[string]memorySession
}

type memorySession struct {
	user      User
	expiresAt time.Time
	revokedAt *time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{sessions: map[string]memorySession{}}
}

func (s *MemoryStore) Save(_ context.Context, token string, user User, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune(time.Now())
	s.sessions[string(HashToken(token))] = memorySession{user: user, expiresAt: expiresAt}
	return nil
}

func (s *MemoryStore) Find(_ context.Context, token string, now time.Time) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune(now)
	stored, ok := s.sessions[string(HashToken(token))]
	if !ok || stored.revokedAt != nil || !stored.expiresAt.After(now) {
		return User{}, ErrInvalidSession
	}
	return stored.user, nil
}

func (s *MemoryStore) Revoke(_ context.Context, token string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune(now)
	key := string(HashToken(token))
	stored, ok := s.sessions[key]
	if !ok || !stored.expiresAt.After(now) {
		return ErrInvalidSession
	}
	stored.revokedAt = &now
	s.sessions[key] = stored
	return nil
}

func (s *MemoryStore) prune(now time.Time) {
	for key, stored := range s.sessions {
		if !stored.expiresAt.After(now) {
			delete(s.sessions, key)
		}
	}
}

func HashToken(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
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

package oidc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
)

var ErrInvalidClient = errors.New("invalid client")

type AuthorizationCodeRecord struct {
	Data                AuthorizationCodeData
	CodeChallenge       string
	CodeChallengeMethod string
	ExpiresAt           time.Time
}

type Store interface {
	EnsureClients(ctx context.Context, clients []Client) error
	Cleanup(ctx context.Context, now time.Time) error
	FindClient(ctx context.Context, clientID string) (Client, error)
	SaveTransaction(ctx context.Context, transactionID string, request AuthorizationRequest, expiresAt time.Time) error
	ConsumeTransaction(ctx context.Context, transactionID string, now time.Time) (AuthorizationRequest, error)
	SaveAuthorizationCode(ctx context.Context, code string, record AuthorizationCodeRecord) error
	RedeemAuthorizationCode(ctx context.Context, code string, clientID string, redirectURI string, now time.Time) (AuthorizationCodeRecord, error)
	SaveAccessToken(ctx context.Context, token string, data AccessTokenData) error
	ValidateAccessToken(ctx context.Context, token string, now time.Time) (AccessTokenData, error)
}

type MemoryStore struct {
	mu           sync.Mutex
	clients      map[string]Client
	transactions map[string]memoryTransaction
	codes        map[string]AuthorizationCodeRecord
	accessTokens map[string]AccessTokenData
}

type memoryTransaction struct {
	request   AuthorizationRequest
	expiresAt time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		clients:      map[string]Client{},
		transactions: map[string]memoryTransaction{},
		codes:        map[string]AuthorizationCodeRecord{},
		accessTokens: map[string]AccessTokenData{},
	}
}

func (s *MemoryStore) EnsureClients(_ context.Context, clients []Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, client := range clients {
		secretHash := client.SecretHash
		if len(secretHash) == 0 {
			var err error
			secretHash, err = HashClientSecret(client.Secret)
			if err != nil {
				return err
			}
		}
		s.clients[client.ID] = Client{
			ID:           client.ID,
			SecretHash:   append([]byte(nil), secretHash...),
			RedirectURIs: append([]string(nil), client.RedirectURIs...),
		}
	}
	return nil
}

func (s *MemoryStore) FindClient(_ context.Context, clientID string) (Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, ok := s.clients[clientID]
	if !ok || client.Disabled {
		return Client{}, ErrInvalidClient
	}
	client.RedirectURIs = append([]string(nil), client.RedirectURIs...)
	client.SecretHash = append([]byte(nil), client.SecretHash...)
	return client, nil
}

func (s *MemoryStore) Cleanup(_ context.Context, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune(now)
	return nil
}

func (s *MemoryStore) SaveTransaction(_ context.Context, transactionID string, request AuthorizationRequest, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.prune(time.Now())
	if len(s.transactions) >= 10000 {
		return errors.New("too many OIDC transactions")
	}
	s.transactions[string(stringHash(transactionID))] = memoryTransaction{
		request:   copyAuthorizationRequest(request),
		expiresAt: expiresAt,
	}
	return nil
}

func (s *MemoryStore) ConsumeTransaction(_ context.Context, transactionID string, now time.Time) (AuthorizationRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.prune(now)
	key := string(stringHash(transactionID))
	transaction, ok := s.transactions[key]
	if !ok {
		return AuthorizationRequest{}, ErrInvalidGrant
	}
	delete(s.transactions, key)
	return copyAuthorizationRequest(transaction.request), nil
}

func (s *MemoryStore) SaveAuthorizationCode(_ context.Context, code string, record AuthorizationCodeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.prune(time.Now())
	if len(s.codes) >= 10000 {
		return errors.New("too many OIDC authorization codes")
	}
	s.codes[string(stringHash(code))] = copyAuthorizationCodeRecord(record)
	return nil
}

func (s *MemoryStore) RedeemAuthorizationCode(_ context.Context, code string, clientID string, redirectURI string, now time.Time) (AuthorizationCodeRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.prune(now)
	key := string(stringHash(code))
	record, ok := s.codes[key]
	if !ok || record.Data.ClientID != clientID || record.Data.RedirectURI != redirectURI {
		return AuthorizationCodeRecord{}, ErrInvalidGrant
	}
	delete(s.codes, key)
	return copyAuthorizationCodeRecord(record), nil
}

func (s *MemoryStore) SaveAccessToken(_ context.Context, token string, data AccessTokenData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.prune(time.Now())
	if len(s.accessTokens) >= 10000 {
		return errors.New("too many OIDC access tokens")
	}
	s.accessTokens[string(stringHash(token))] = copyAccessTokenData(data)
	return nil
}

func (s *MemoryStore) ValidateAccessToken(_ context.Context, token string, now time.Time) (AccessTokenData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.prune(now)
	data, ok := s.accessTokens[string(stringHash(token))]
	client, clientOK := s.clients[data.ClientID]
	if !ok || !clientOK || client.Disabled {
		return AccessTokenData{}, ErrInvalidToken
	}
	return copyAccessTokenData(data), nil
}

func (s *MemoryStore) prune(now time.Time) {
	for key, transaction := range s.transactions {
		if !transaction.expiresAt.After(now) {
			delete(s.transactions, key)
		}
	}
	for key, code := range s.codes {
		if !code.ExpiresAt.After(now) {
			delete(s.codes, key)
		}
	}
	for key, token := range s.accessTokens {
		if !token.ExpiresAt.After(now) {
			delete(s.accessTokens, key)
		}
	}
}

func HashSecret(value string) []byte {
	return stringHash(value)
}

func HashClientSecret(value string) ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	key := argon2.IDKey([]byte(value), salt, 1, 64*1024, 2, 32)
	encoded := strings.Join([]string{
		"argon2id",
		"v=19",
		"m=65536,t=1,p=2",
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	}, "$")
	return []byte(encoded), nil
}

func VerifyClientSecret(hash []byte, value string) bool {
	parts := strings.Split(string(hash), "$")
	if len(parts) != 5 || parts[0] != "argon2id" || parts[1] != "v=19" || parts[2] != "m=65536,t=1,p=2" {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(value), salt, 1, 64*1024, 2, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func stringHash(value string) []byte {
	digest := sha256.Sum256([]byte(value))
	return digest[:]
}

func copyAuthorizationRequest(request AuthorizationRequest) AuthorizationRequest {
	request.Scope = append([]string(nil), request.Scope...)
	return request
}

func copyAuthorizationCodeRecord(record AuthorizationCodeRecord) AuthorizationCodeRecord {
	record.Data.Scope = append([]string(nil), record.Data.Scope...)
	return record
}

func copyAccessTokenData(data AccessTokenData) AccessTokenData {
	data.Scope = append([]string(nil), data.Scope...)
	return data
}

package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/sandbox-auth/src/infrastructure/session"
)

func TestEntSessionStorePersistsHashedAndRevocableSession(t *testing.T) {
	ctx := context.Background()
	client := newOIDCTestClient(t)
	account, err := client.Account.Create().
		SetOidcSubject("subject-1").
		SetWebauthnUserHandle([]byte("handle-1")).
		Save(ctx)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	store := NewEntSessionStore(client)
	expiresAt := time.Now().Add(time.Minute)
	if err := store.Save(ctx, "session-token", session.User{AccountID: int64(account.ID)}, expiresAt); err != nil {
		t.Fatalf("save session: %v", err)
	}

	stored, err := client.AuthSession.Query().Only(ctx)
	if err != nil {
		t.Fatalf("query session: %v", err)
	}
	if string(stored.TokenHash) == "session-token" || string(stored.TokenHash) == "" {
		t.Fatal("session token must be stored as a hash")
	}

	user, err := store.Find(ctx, "session-token", time.Now())
	if err != nil {
		t.Fatalf("find session: %v", err)
	}
	if user.AccountID != int64(account.ID) {
		t.Fatalf("AccountID = %d, want %d", user.AccountID, account.ID)
	}

	if err := store.Revoke(ctx, "session-token", time.Now()); err != nil {
		t.Fatalf("revoke session: %v", err)
	}
	if _, err := store.Find(ctx, "session-token", time.Now()); err == nil {
		t.Fatal("expected revoked session to be rejected")
	}
}

func TestEntSessionStoreRejectsExpiredSession(t *testing.T) {
	ctx := context.Background()
	client := newOIDCTestClient(t)
	account, err := client.Account.Create().
		SetOidcSubject("subject-2").
		SetWebauthnUserHandle([]byte("handle-2")).
		Save(ctx)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	store := NewEntSessionStore(client)
	if err := store.Save(ctx, "expired-token", session.User{AccountID: int64(account.ID)}, time.Now().Add(-time.Second)); err != nil {
		t.Fatalf("save session: %v", err)
	}
	if _, err := store.Find(ctx, "expired-token", time.Now()); err == nil {
		t.Fatal("expected expired session to be rejected")
	}
}

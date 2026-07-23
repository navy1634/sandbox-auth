package session

import (
	"testing"
	"time"
)

// TTL を過ぎた Cookie が拒否されることを確認する。
func TestVerifyRejectsExpiredSession(t *testing.T) {
	manager := NewManager([]byte("01234567890123456789012345678901"))
	value, err := manager.signAt(User{AccountID: 1, Email: "user@example.com"}, time.Now().Add(-DefaultTTL-time.Second))
	if err != nil {
		t.Fatalf("sign session: %v", err)
	}

	if _, err := manager.Verify(value); err == nil {
		t.Fatal("expected expired session to be rejected")
	}
}

// 期限内の Cookie からユーザー情報を復元できることを確認する。
func TestVerifyAcceptsSignedSessionBeforeExpiry(t *testing.T) {
	manager := NewManager([]byte("01234567890123456789012345678901"))
	value, err := manager.Sign(User{AccountID: 1, Email: "user@example.com"})
	if err != nil {
		t.Fatalf("sign session: %v", err)
	}

	user, err := manager.Verify(value)
	if err != nil {
		t.Fatalf("verify session: %v", err)
	}
	if user.AccountID != 1 || user.Email != "user@example.com" {
		t.Fatalf("unexpected user: %+v", user)
	}
}

// revoke 済み Cookie が拒否されることを確認する。
func TestVerifyRejectsRevokedSession(t *testing.T) {
	manager := NewManager([]byte("01234567890123456789012345678901"))
	value, err := manager.Sign(User{AccountID: 1, Email: "user@example.com"})
	if err != nil {
		t.Fatalf("sign session: %v", err)
	}

	if err := manager.Revoke(value); err != nil {
		t.Fatalf("revoke session: %v", err)
	}

	if _, err := manager.Verify(value); err == nil {
		t.Fatal("expected revoked session to be rejected")
	}
}

// 期限切れの revoke エントリが検証時に削除されることを確認する。
func TestVerifyPrunesExpiredRevokedSessions(t *testing.T) {
	manager := NewManager([]byte("01234567890123456789012345678901"))
	manager.revoked["expired"] = time.Now().Add(-time.Second).Unix()
	value, err := manager.Sign(User{AccountID: 1, Email: "user@example.com"})
	if err != nil {
		t.Fatalf("sign session: %v", err)
	}

	if _, err := manager.Verify(value); err != nil {
		t.Fatalf("verify session: %v", err)
	}
	if _, ok := manager.revoked["expired"]; ok {
		t.Fatal("expected expired revoked session to be pruned")
	}
}

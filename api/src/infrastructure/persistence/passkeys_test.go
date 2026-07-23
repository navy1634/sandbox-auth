package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/sandbox-nextjs/src/ent"
	"github.com/sandbox-nextjs/src/ent/enttest"
	entsession "github.com/sandbox-nextjs/src/ent/webauthnsession"
	"github.com/sandbox-nextjs/src/infrastructure/database"
	"github.com/sandbox-nextjs/src/repository"
	_ "modernc.org/sqlite"
)

// 新規保存時に期限切れ WebAuthn セッションが削除されることを確認する。
func TestSaveSessionDeletesExpiredSessionsBeforeCreate(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	repo := NewEntPasskeyRepository(client, nil)
	expiredAt := time.Now().Add(-time.Minute)

	err := client.WebauthnSession.Create().
		SetID("expired-session").
		SetCeremony("passkey_login").
		SetSessionJSON(json.RawMessage(`{}`)).
		SetExpiresAt(expiredAt).
		Exec(ctx)
	if err != nil {
		t.Fatalf("create expired session: %v", err)
	}

	// ここでは新しいログインセッションを保存し、保存前 cleanup を動かす。
	err = repo.SaveSession(ctx, "new-session", nil, "passkey_login", &webauthn.SessionData{}, time.Minute)
	if err != nil {
		t.Fatalf("save session: %v", err)
	}

	exists, err := client.WebauthnSession.Query().
		Where(entsession.ID("expired-session")).
		Exist(ctx)
	if err != nil {
		t.Fatalf("query expired session: %v", err)
	}
	if exists {
		t.Fatal("expected expired session to be deleted")
	}
}

// 同一アカウントの登録セッションが上限に達したら拒否されることを確認する。
func TestSaveSessionRejectsWhenActiveRegistrationSessionsReachLimit(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	repo := NewEntPasskeyRepository(client, nil)
	activeUntil := time.Now().Add(time.Minute)
	accountID := int64(1)

	// ここでは期限内の登録セッションを上限数まで作る。
	for i := range maxActiveWebAuthnRegistrationSessions {
		err := client.WebauthnSession.Create().
			SetID(fmt.Sprintf("active-session-%d", i)).
			SetAccountID(accountID).
			SetCeremony("passkey_register").
			SetSessionJSON(json.RawMessage(`{}`)).
			SetExpiresAt(activeUntil).
			Exec(ctx)
		if err != nil {
			t.Fatalf("create active session %d: %v", i, err)
		}
	}

	err := repo.SaveSession(ctx, "overflow-session", &accountID, "passkey_register", &webauthn.SessionData{}, time.Minute)
	if !errors.Is(err, repository.ErrTooManyPasskeySessions) {
		t.Fatalf("expected session limit error, got %v", err)
	}
}

// 登録用上限数の匿名ログインセッションでは拒否されないことを確認する。
func TestSaveSessionDoesNotApplySharedLimitToLoginSessions(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	repo := NewEntPasskeyRepository(client, nil)
	activeUntil := time.Now().Add(time.Minute)

	for i := range maxActiveWebAuthnRegistrationSessions {
		err := client.WebauthnSession.Create().
			SetID(fmt.Sprintf("active-login-session-%d", i)).
			SetCeremony("passkey_login").
			SetSessionJSON(json.RawMessage(`{}`)).
			SetExpiresAt(activeUntil).
			Exec(ctx)
		if err != nil {
			t.Fatalf("create active login session %d: %v", i, err)
		}
	}

	err := repo.SaveSession(ctx, "new-login-session", nil, "passkey_login", &webauthn.SessionData{}, time.Minute)
	if err != nil {
		t.Fatalf("save login session: %v", err)
	}
}

// 匿名ログインセッションが上限に達したら拒否されることを確認する。
func TestSaveSessionRejectsWhenActiveLoginSessionsReachLimit(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	repo := NewEntPasskeyRepository(client, nil)
	activeUntil := time.Now().Add(time.Minute)

	// ここでは期限内の匿名ログインセッションを上限数まで作る。
	for i := range maxActiveWebAuthnLoginSessions {
		err := client.WebauthnSession.Create().
			SetID(fmt.Sprintf("active-login-overflow-session-%d", i)).
			SetCeremony("passkey_login").
			SetSessionJSON(json.RawMessage(`{}`)).
			SetExpiresAt(activeUntil).
			Exec(ctx)
		if err != nil {
			t.Fatalf("create active login session %d: %v", i, err)
		}
	}

	err := repo.SaveSession(ctx, "overflow-login-session", nil, "passkey_login", &webauthn.SessionData{}, time.Minute)
	if !errors.Is(err, repository.ErrTooManyPasskeySessions) {
		t.Fatalf("expected session limit error, got %v", err)
	}
}

// 同じ credential ID を別アカウントへ保存できないことを確認する。
func TestSaveCredentialRejectsExistingCredentialID(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	repo := NewEntPasskeyRepository(client, nil)
	credential := &webauthn.Credential{ID: []byte("credential-id")}

	// ここでは先に account 1 へ credential を保存する。
	err := repo.SaveCredential(ctx, 1, credential)
	if err != nil {
		t.Fatalf("save credential: %v", err)
	}

	err = repo.SaveCredential(ctx, 2, credential)
	if !errors.Is(err, repository.ErrPasskeyCredentialExists) {
		t.Fatalf("expected credential exists error, got %v", err)
	}
}

// 失敗リクエスト内で消費した WebAuthn セッションがロールバックされることを確認する。
func TestConsumeSessionUsesRequestTransaction(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	repo := NewEntPasskeyRepository(client, nil)

	err := client.WebauthnSession.Create().
		SetID("rollback-session").
		SetCeremony("passkey_login").
		SetSessionJSON(json.RawMessage(`{}`)).
		SetExpiresAt(time.Now().Add(time.Minute)).
		Exec(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(database.TransactionMiddleware(client))
	engine.POST("/consume", func(c *gin.Context) {
		if _, err := repo.ConsumeSession(c.Request.Context(), "rollback-session", "passkey_login"); err != nil {
			t.Fatalf("consume session: %v", err)
		}
		c.Status(http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodPost, "/consume", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	exists, err := client.WebauthnSession.Query().
		Where(entsession.ID("rollback-session")).
		Exist(ctx)
	if err != nil {
		t.Fatalf("query session: %v", err)
	}
	if !exists {
		t.Fatal("expected consumed session to be rolled back")
	}
}

// テスト名ごとに別のインメモリ SQLite クライアントを作る。
func newTestClient(t *testing.T) *ent.Client {
	t.Helper()

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(1)", url.QueryEscape(t.Name())))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close sqlite: %v", err)
		}
	})

	driver := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(driver)))
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close ent client: %v", err)
		}
	})
	return client
}

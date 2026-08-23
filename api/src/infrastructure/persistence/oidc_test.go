package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/sandbox-nextjs/src/ent"
	"github.com/sandbox-nextjs/src/ent/enttest"
	"github.com/sandbox-nextjs/src/infrastructure/oidc"
	_ "modernc.org/sqlite"
)

// OIDC の一時データが再起動後も使えるストアとして保存されることを確認する。
func TestEntOIDCStorePersistsAndConsumesProtocolData(t *testing.T) {
	ctx := context.Background()
	client := newOIDCTestClient(t)
	store := NewEntOIDCStore(client)

	clientConfig := oidc.Client{
		ID:           "client",
		Secret:       "client-secret",
		RedirectURIs: []string{"https://app.example.com/callback"},
	}
	if err := store.EnsureClients(ctx, []oidc.Client{clientConfig}); err != nil {
		t.Fatalf("ensure clients: %v", err)
	}
	storedClient, err := store.FindClient(ctx, clientConfig.ID)
	if err != nil {
		t.Fatalf("find client: %v", err)
	}
	if string(storedClient.SecretHash) == clientConfig.Secret || !oidc.VerifyClientSecret(storedClient.SecretHash, clientConfig.Secret) {
		t.Fatal("expected client secret to be stored as a verifiable hash")
	}

	request := oidc.AuthorizationRequest{
		ClientID:            clientConfig.ID,
		RedirectURI:         clientConfig.RedirectURIs[0],
		Scope:               []string{"openid", "email"},
		State:               "state",
		Nonce:               "nonce",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
	}
	if err := store.SaveTransaction(ctx, "transaction", request, time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("save transaction: %v", err)
	}
	consumedRequest, err := store.ConsumeTransaction(ctx, "transaction", time.Now())
	if err != nil {
		t.Fatalf("consume transaction: %v", err)
	}
	if consumedRequest.ClientID != request.ClientID || consumedRequest.Nonce != request.Nonce {
		t.Fatalf("unexpected transaction: %+v", consumedRequest)
	}
	if _, err := store.ConsumeTransaction(ctx, "transaction", time.Now()); err == nil {
		t.Fatal("expected transaction replay to fail")
	}

	codeData := oidc.AuthorizationCodeData{
		ClientID:    clientConfig.ID,
		RedirectURI: clientConfig.RedirectURIs[0],
		AccountID:   42,
		Scope:       request.Scope,
		Nonce:       request.Nonce,
	}
	if err := store.SaveAuthorizationCode(ctx, "authorization-code", oidc.AuthorizationCodeRecord{
		Data:                codeData,
		CodeChallenge:       request.CodeChallenge,
		CodeChallengeMethod: request.CodeChallengeMethod,
		ExpiresAt:           time.Now().Add(time.Minute),
	}); err != nil {
		t.Fatalf("save authorization code: %v", err)
	}
	redeemedCode, err := store.RedeemAuthorizationCode(ctx, "authorization-code", clientConfig.ID, request.RedirectURI, time.Now())
	if err != nil {
		t.Fatalf("redeem authorization code: %v", err)
	}
	if redeemedCode.Data.AccountID != codeData.AccountID || redeemedCode.CodeChallenge != request.CodeChallenge {
		t.Fatalf("unexpected authorization code: %+v", redeemedCode)
	}
	if _, err := store.RedeemAuthorizationCode(ctx, "authorization-code", clientConfig.ID, request.RedirectURI, time.Now()); err == nil {
		t.Fatal("expected authorization code replay to fail")
	}

	accessTokenData := oidc.AccessTokenData{
		ClientID:  clientConfig.ID,
		AccountID: codeData.AccountID,
		Scope:     codeData.Scope,
		ExpiresAt: time.Now().Add(time.Minute),
	}
	if err := store.SaveAccessToken(ctx, "access-token", accessTokenData); err != nil {
		t.Fatalf("save access token: %v", err)
	}
	validatedToken, err := store.ValidateAccessToken(ctx, "access-token", time.Now())
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}
	if validatedToken.AccountID != accessTokenData.AccountID {
		t.Fatalf("AccountID = %d, want %d", validatedToken.AccountID, accessTokenData.AccountID)
	}

	storedCode, err := client.OIDCAuthorizationCode.Query().Only(ctx)
	if err != nil {
		t.Fatalf("query authorization code: %v", err)
	}
	if string(storedCode.CodeHash) == "authorization-code" {
		t.Fatal("authorization code must not be stored in plaintext")
	}
}

// テスト名ごとに別のインメモリ SQLite クライアントを作る。
func newOIDCTestClient(t *testing.T) *ent.Client {
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

package persistence

import (
	"context"
	"time"

	"github.com/sandbox-nextjs/src/ent"
	"github.com/sandbox-nextjs/src/ent/authsession"
	"github.com/sandbox-nextjs/src/infrastructure/database"
	"github.com/sandbox-nextjs/src/infrastructure/session"
)

type EntSessionStore struct {
	client *ent.Client
}

func NewEntSessionStore(client *ent.Client) *EntSessionStore {
	return &EntSessionStore{client: client}
}

func (s *EntSessionStore) db(ctx context.Context) (*ent.Client, error) {
	return database.ClientFromContext(ctx, s.client)
}

func (s *EntSessionStore) Save(ctx context.Context, token string, user session.User, expiresAt time.Time) error {
	db, err := s.db(ctx)
	if err != nil {
		return err
	}
	return db.AuthSession.Create().
		SetTokenHash(session.HashToken(token)).
		SetAccountID(user.AccountID).
		SetExpiresAt(expiresAt).
		Exec(ctx)
}

func (s *EntSessionStore) Find(ctx context.Context, token string, now time.Time) (session.User, error) {
	db, err := s.db(ctx)
	if err != nil {
		return session.User{}, err
	}
	stored, err := db.AuthSession.Query().
		Where(
			authsession.TokenHash(session.HashToken(token)),
			authsession.ExpiresAtGT(now),
			authsession.RevokedAtIsNil(),
		).
		Only(ctx)
	if ent.IsNotFound(err) {
		return session.User{}, session.ErrInvalidSession
	}
	if err != nil {
		return session.User{}, err
	}
	return session.User{AccountID: stored.AccountID}, nil
}

func (s *EntSessionStore) Revoke(ctx context.Context, token string, now time.Time) error {
	db, err := s.db(ctx)
	if err != nil {
		return err
	}
	_, err = db.AuthSession.Update().
		Where(
			authsession.TokenHash(session.HashToken(token)),
			authsession.ExpiresAtGT(now),
			authsession.RevokedAtIsNil(),
		).
		SetRevokedAt(now).
		Save(ctx)
	return err
}

var _ session.Store = (*EntSessionStore)(nil)

-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE accounts
  ADD COLUMN webauthn_user_handle BYTEA;

UPDATE accounts
SET webauthn_user_handle = gen_random_bytes(32)
WHERE webauthn_user_handle IS NULL;

ALTER TABLE accounts
  ALTER COLUMN webauthn_user_handle SET NOT NULL;

CREATE UNIQUE INDEX accounts_webauthn_user_handle_key ON accounts (webauthn_user_handle);

CREATE TABLE webauthn_credentials (
  id BIGSERIAL PRIMARY KEY,
  account_id BIGINT NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
  credential_id BYTEA NOT NULL,
  credential_json JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_used_at TIMESTAMPTZ,
  CONSTRAINT webauthn_credentials_credential_id_unique UNIQUE (credential_id)
);

CREATE INDEX webauthn_credentials_account_id_idx ON webauthn_credentials (account_id);

CREATE TABLE webauthn_sessions (
  id TEXT PRIMARY KEY,
  account_id BIGINT REFERENCES accounts (id) ON DELETE CASCADE,
  ceremony TEXT NOT NULL,
  session_json JSONB NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX webauthn_sessions_expires_at_idx ON webauthn_sessions (expires_at);

-- +goose Down
DROP TABLE webauthn_sessions;
DROP TABLE webauthn_credentials;
DROP INDEX accounts_webauthn_user_handle_key;
ALTER TABLE accounts DROP COLUMN webauthn_user_handle;

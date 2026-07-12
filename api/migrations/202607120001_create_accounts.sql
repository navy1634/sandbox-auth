-- +goose Up
CREATE TABLE accounts (
  id BIGSERIAL PRIMARY KEY,
  display_name TEXT NOT NULL DEFAULT '',
  bio TEXT NOT NULL DEFAULT '',
  registered_at TIMESTAMPTZ,
  webauthn_user_handle BYTEA NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT accounts_webauthn_user_handle_unique UNIQUE (webauthn_user_handle)
);

CREATE TABLE auth_identities (
  id BIGSERIAL PRIMARY KEY,
  account_id BIGINT NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
  provider TEXT NOT NULL,
  provider_account_id TEXT NOT NULL,
  email TEXT NOT NULL,
  email_verified BOOLEAN NOT NULL DEFAULT false,
  name TEXT NOT NULL DEFAULT '',
  picture TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT auth_identities_account_provider_unique UNIQUE (account_id, provider),
  CONSTRAINT auth_identities_provider_account_id_unique UNIQUE (provider, provider_account_id)
);

CREATE INDEX auth_identities_account_id_idx ON auth_identities (account_id);
CREATE INDEX auth_identities_email_idx ON auth_identities (email);

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
DROP TABLE auth_identities;
DROP TABLE accounts;

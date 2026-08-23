-- +goose Up
CREATE TABLE accounts (
  id BIGSERIAL PRIMARY KEY,
  oidc_subject TEXT NOT NULL UNIQUE,
  webauthn_user_handle BYTEA NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE FUNCTION prevent_account_oidc_subject_update() RETURNS trigger AS $$
BEGIN
  IF NEW.oidc_subject <> OLD.oidc_subject THEN
    RAISE EXCEPTION 'accounts.oidc_subject is immutable';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER accounts_oidc_subject_immutable
BEFORE UPDATE OF oidc_subject ON accounts
FOR EACH ROW EXECUTE FUNCTION prevent_account_oidc_subject_update();

CREATE TABLE auth_sessions (
  id BIGSERIAL PRIMARY KEY,
  token_hash BYTEA NOT NULL UNIQUE,
  account_id BIGINT NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX auth_sessions_account_id_expires_at_idx ON auth_sessions (account_id, expires_at);
CREATE INDEX auth_sessions_expires_at_idx ON auth_sessions (expires_at);
CREATE INDEX auth_sessions_active_token_idx ON auth_sessions (token_hash) WHERE revoked_at IS NULL;

CREATE TABLE auth_identities (
  id BIGSERIAL PRIMARY KEY,
  account_id BIGINT NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
  provider TEXT NOT NULL,
  provider_account_id TEXT NOT NULL,
  email TEXT NOT NULL,
  email_verified BOOLEAN NOT NULL DEFAULT false,
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

CREATE TABLE oidc_clients (
  id BIGSERIAL PRIMARY KEY,
  client_id TEXT NOT NULL UNIQUE,
  client_secret_hash BYTEA NOT NULL,
  disabled BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE oidc_client_redirect_uris (
  id BIGSERIAL PRIMARY KEY,
  client_id TEXT NOT NULL REFERENCES oidc_clients (client_id) ON DELETE CASCADE,
  redirect_uri TEXT NOT NULL,
  CONSTRAINT oidc_client_redirect_uris_unique UNIQUE (client_id, redirect_uri)
);

CREATE INDEX oidc_client_redirect_uris_client_id_idx ON oidc_client_redirect_uris (client_id);

CREATE TABLE oidc_authorization_transactions (
  id BIGSERIAL PRIMARY KEY,
  transaction_hash BYTEA NOT NULL UNIQUE,
  client_id TEXT NOT NULL REFERENCES oidc_clients (client_id) ON DELETE CASCADE,
  redirect_uri TEXT NOT NULL,
  scope JSONB NOT NULL,
  state TEXT NOT NULL DEFAULT '',
  nonce TEXT NOT NULL,
  code_challenge TEXT NOT NULL,
  code_challenge_method TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX oidc_authorization_transactions_expires_at_idx ON oidc_authorization_transactions (expires_at);
CREATE INDEX oidc_authorization_transactions_client_id_expires_at_idx ON oidc_authorization_transactions (client_id, expires_at);

CREATE TABLE oidc_authorization_codes (
  id BIGSERIAL PRIMARY KEY,
  code_hash BYTEA NOT NULL UNIQUE,
  client_id TEXT NOT NULL REFERENCES oidc_clients (client_id) ON DELETE CASCADE,
  redirect_uri TEXT NOT NULL,
  account_id BIGINT NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
  scope JSONB NOT NULL,
  nonce TEXT NOT NULL,
  code_challenge TEXT NOT NULL,
  code_challenge_method TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX oidc_authorization_codes_expires_at_idx ON oidc_authorization_codes (expires_at);
CREATE INDEX oidc_authorization_codes_client_id_expires_at_idx ON oidc_authorization_codes (client_id, expires_at);
CREATE INDEX oidc_authorization_codes_account_id_expires_at_idx ON oidc_authorization_codes (account_id, expires_at);

CREATE TABLE oidc_access_tokens (
  id BIGSERIAL PRIMARY KEY,
  token_hash BYTEA NOT NULL UNIQUE,
  client_id TEXT NOT NULL REFERENCES oidc_clients (client_id) ON DELETE CASCADE,
  account_id BIGINT NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
  scope JSONB NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX oidc_access_tokens_expires_at_idx ON oidc_access_tokens (expires_at);
CREATE INDEX oidc_access_tokens_account_id_expires_at_idx ON oidc_access_tokens (account_id, expires_at);

-- +goose Down
DROP TABLE oidc_access_tokens;
DROP TABLE oidc_authorization_codes;
DROP TABLE oidc_authorization_transactions;
DROP TABLE oidc_client_redirect_uris;
DROP TABLE oidc_clients;
DROP TABLE auth_sessions;
DROP TABLE webauthn_sessions;
DROP TABLE webauthn_credentials;
DROP TABLE auth_identities;
DROP TRIGGER accounts_oidc_subject_immutable ON accounts;
DROP FUNCTION prevent_account_oidc_subject_update();
DROP TABLE accounts;

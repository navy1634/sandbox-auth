# api

Gin で動く API サーバーです。Google OAuth の callback とアプリ用セッション Cookie の発行を担当します。

## 環境変数

```txt
GOOGLE_ID=
GOOGLE_SECRET=
AUTH_SECRET=
DATABASE_URL=postgres://postgres:password@localhost:5432/test?sslmode=disable
FRONTEND_URL=http://localhost:3000
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback
PASSKEY_RP_ID=localhost
PASSKEY_RP_ORIGIN=http://localhost:3000
```

Docker Compose でリモート端末から使う場合は、ルートディレクトリで公開 URL を指定します。

```txt
SANDBOX_FRONTEND_URL=http://192.0.2.10:3000 \
SANDBOX_API_URL=http://192.0.2.10:8080 \
SANDBOX_RP_ID=192.0.2.10 \
docker compose up --build
```

Google Cloud Console の OAuth callback URL には、次を登録してください。

```txt
http://localhost:8080/auth/google/callback
```

## migration

```txt
mise run migrate:up
mise run migrate:down
```

`accounts` に Google アカウントとプロフィールを保存し、`webauthn_credentials` と `webauthn_sessions` にパスキーの credential と ceremony session を保存します。

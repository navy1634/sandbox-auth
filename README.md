# sandbox_nextjs

Next.js のフロントエンドと、Go Gin のバックエンドを分けた構成です。

## ディレクトリ

```txt
web  Next.js
api  Gin API
```

## ローカル起動

```txt
docker compose up --build
```

フロントエンドは `http://localhost:3000`、バックエンドは `http://localhost:8080` で起動します。

## Google OAuth

Google Cloud Console の OAuth callback URL には、次を登録してください。

```txt
http://localhost:8080/auth/google/callback
```

`api/.env.local` に次の環境変数を設定してください。

```txt
GOOGLE_ID=
GOOGLE_SECRET=
AUTH_SECRET=
DATABASE_URL=postgres://postgres:password@localhost:5432/test?sslmode=disable
PASSKEY_RP_ID=localhost
PASSKEY_RP_ORIGIN=http://localhost:3000
```

ログイン確認ページは `http://localhost:3000/login` です。初回登録は `http://localhost:3000/register`、マイページは `http://localhost:3000/mypage` で確認できます。

DB は Docker Compose の `db` サービスで起動します。migration は `api` ディレクトリで `mise run migrate:up` を実行します。

## Passkey

パスキーは Google ログイン後に初回登録画面かマイページから登録します。登録後はログイン画面の「パスキーでログイン」からセッションを作成できます。

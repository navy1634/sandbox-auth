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

## API なしで WireMock を使う

API サーバーを用意せず、フロントエンドの接続先を WireMock に向ける場合は次のように起動します。

```txt
SANDBOX_API_URL=http://localhost:8081 docker compose --profile mock up --build web wiremock
```

WireMock は `http://localhost:8081` で起動します。スタブは `wiremock/mappings` に置いています。

現在のスタブは、フロントエンドが使う次のエンドポイントを返します。

- `GET /me` は、`app_session` Cookie があればログイン済み、なければ未ログインとして返します。
- `GET /auth/google/login` は、モック用の `app_session` Cookie を発行してマイページへリダイレクトします。
- `POST /auth/logout` は、`app_session` Cookie を削除します。
- `POST /account/profile` は、ログイン済みならプロフィール更新済みのレスポンスを返します。
- `POST /passkeys/login/options` と `POST /passkeys/register/options` は、ブラウザ API に渡す公開鍵オプションを返します。
- `POST /passkeys/login/verify` と `POST /passkeys/register/verify` は、検証成功のレスポンスを返します。

## リモートから使う

リモート端末のブラウザから使う場合は、sandbox を動かすマシンの IP アドレスまたは DNS 名を指定して起動します。

```txt
SANDBOX_FRONTEND_URL=http://192.0.2.10:3000 \
SANDBOX_API_URL=http://192.0.2.10:8080 \
SANDBOX_RP_ID=192.0.2.10 \
docker compose up --build
```

Google Cloud Console の OAuth callback URL には、次も登録してください。

```txt
http://192.0.2.10:8080/auth/google/callback
```

パスキーは HTTPS か `localhost` で使うのが前提です。IP アドレスの HTTP で動かす場合、ブラウザによってはパスキー登録やログインが失敗します。その場合は HTTPS のドメインを用意し、`SANDBOX_FRONTEND_URL`、`SANDBOX_API_URL`、`SANDBOX_RP_ID` をそのドメインに合わせてください。

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

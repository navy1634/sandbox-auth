# sandbox_auth

Next.js と Go Gin で作った SSO 認証サービスです。Google OAuth とパスキーでログインし、認証 web 自身には Cookie セッションを、外部アプリには OIDC の ID トークンを発行します。

## 構成

```txt
.
├── api        Go Gin API
├── web        Next.js アプリ
├── wiremock   フロントエンド確認用の WireMock スタブ
└── compose.yml
```

## 使用している主な技術

| 種別 | 技術 |
| ---- | ---- |
| フロントエンド | Next.js、React、TypeScript |
| バックエンド | Go、Gin |
| データベース | PostgreSQL |
| 認証 | Google OAuth、WebAuthn、Cookie セッション、OIDC Provider |
| モック API | WireMock |

詳しいバージョンは `web/package.json`、`api/go.mod`、`compose.yml` を参照してください。

## 起動

API は `api/.env.local` を読み込みます。`api/.env.template` を `api/.env.local` としてコピーし、必要な値を設定してください。

```txt
docker compose up --build
```

起動後は次の URL にアクセスできます。

| 用途 | URL |
| ---- | --- |
| フロントエンド | <http://localhost:3000> |
| API | <http://localhost:8080> |
| 単独確認用の戻り先 | <http://localhost:3000/mypage> |

認証 web は 3000 番ポート固定で扱います。3000 番が別プロセスで使われている場合は、Next.js を別ポートへ逃がさず、先にそのプロセスを止めてください。

## 認証 web の戻り先

認証 web 自身の画面から許可済みの戻り先へ移動する場合だけ、`/login` に `redirect_to` を付けます。この方法は外部アプリとの標準的な接続方法ではありません。

```txt
http://localhost:3000/login?redirect_to=http%3A%2F%2Flocalhost%3A3100%2Fdashboard
```

API 側の `ALLOWED_REDIRECT_URLS` に戻り先の origin を含めてください。`mise run dev` で API を起動する場合は `api/.env.local`、Docker Compose で起動する場合は `SANDBOX_ALLOWED_REDIRECT_URLS` または `compose.yml` のデフォルト値が使われます。設定を変えた後は API の再起動が必要です。

## 外部アプリからOIDCで接続する

`api/.env.local` の `OIDC_ISSUER_URL`、`OIDC_CLIENTS`、RSA署名鍵を設定すると、外部アプリは OIDC Discovery から接続できます。クライアントの `redirect_uris` には外部アプリのバックエンド callback URL を完全一致で登録し、外部アプリ側では Authorization Code Flow と PKCE（S256）を使ってください。外部アプリから認証 web の `/login?redirect_to` を呼ぶ必要はありません。設定例とエンドポイントは `api/README.md` を参照してください。

## 詳細

API の環境変数、migration、エンドポイント、開発コマンドは `api/README.md` を参照してください。

フロントエンドの画面、API 接続、開発コマンドは `web/README.md` を参照してください。

WireMock の役割、起動方法、スタブの内容は `wiremock/README.md` を参照してください。

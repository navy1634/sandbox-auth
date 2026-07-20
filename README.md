# sandbox_auth

Next.js と Go Gin で作った認証サンプルです。Google OAuth でログインし、アプリ用の Cookie セッションを発行します。ログイン後はプロフィール登録、パスキー登録、パスキーによるログインを確認できます。

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
| 認証 | Google OAuth、WebAuthn、Cookie セッション |
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
| フロントエンド | http://localhost:3000 |
| API | http://localhost:8080 |

## API なしで確認する

API サーバーを用意せずにフロントエンドを確認する場合は、WireMock を使います。起動方法とスタブの内容は `wiremock/README.md` を参照してください。

## 詳細

API の環境変数、migration、エンドポイント、開発コマンドは `api/README.md` を参照してください。

フロントエンドの画面、API 接続、開発コマンドは `web/README.md` を参照してください。

WireMock の役割、起動方法、スタブの内容は `wiremock/README.md` を参照してください。

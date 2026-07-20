# api

Gin で動く認証 API サーバーです。Google OAuth の callback、アプリ用 Cookie セッションの発行、プロフィール更新、パスキー登録とパスキーログインを担当します。

## 環境変数

`api/.env.template` を `api/.env.local` としてコピーし、必要な値を設定してください。

`AUTH_SECRET` は Cookie セッションの署名に使うため、32 バイト以上の推測されにくい文字列にしてください。

Google OAuth を使う場合は、Google Cloud Console の OAuth callback URL に次の URL を登録してください。

```txt
http://localhost:8080/auth/google/callback
```

## ローカル起動

```txt
mise run dev
```

通常の `go run` で起動する場合は、次のコマンドを使います。

```txt
mise run run
```

## migration

```txt
mise run migrate:up
mise run migrate:down
```

`accounts` にアカウントとプロフィール、`auth_identities` に OAuth の認証情報、`webauthn_credentials` と `webauthn_sessions` にパスキーの credential と ceremony session を保存します。

## エンドポイント

| メソッド | パス | 内容 |
| -------- | ---- | ---- |
| `GET` | `/health` | ヘルスチェックを返します。 |
| `GET` | `/me` | ログイン中のユーザー情報を返します。 |
| `POST` | `/account/profile` | プロフィールを更新します。 |
| `POST` | `/auth/logout` | セッション Cookie を削除します。 |
| `GET` | `/auth/:provider/login` | OAuth ログインを開始します。 |
| `GET` | `/auth/:provider/callback` | OAuth callback を処理します。 |
| `POST` | `/passkeys/register/options` | パスキー登録用の公開鍵オプションを返します。 |
| `POST` | `/passkeys/register/verify` | パスキー登録結果を検証します。 |
| `POST` | `/passkeys/login/options` | パスキーログイン用の公開鍵オプションを返します。 |
| `POST` | `/passkeys/login/verify` | パスキーログイン結果を検証します。 |

## コマンド

| コマンド | 内容 |
| -------- | ---- |
| `mise run dev` | Air で API を開発起動します。 |
| `mise run build` | API のバイナリをビルドします。 |
| `mise run run` | API を `go run` で起動します。 |
| `mise run install` | Go module を整理します。 |
| `mise run migrate:up` | DB マイグレーションを適用します。 |
| `mise run migrate:down` | DB マイグレーションを戻します。 |
| `mise run format` | Go のコードを整形します。 |
| `mise run lint` | golangci-lint を実行します。 |
| `mise run lint:config-chk` | golangci-lint の設定を検証します。 |
| `mise run test` | Go のテストを実行します。 |
| `mise run ci` | build、format、lint、test を実行します。 |

## トラブルシューティング

### `GOOGLE_ID and GOOGLE_SECRET are required` で終了する

`api/.env.local` に Google OAuth の値を設定してください。Google OAuth を使わずに画面だけ確認する場合は、`wiremock/README.md` を参照してください。

### `AUTH_SECRET must be at least 32 bytes` で終了する

`AUTH_SECRET` に 32 バイト以上の値を設定してください。

### パスキー登録やログインが失敗する

`PASSKEY_RP_ID` と `PASSKEY_RP_ORIGIN` が、ブラウザでアクセスしているホストと合っているか確認してください。パスキーは HTTPS か `localhost` で使うことが前提です。

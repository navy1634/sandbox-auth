# api

Gin で動く認証 API サーバーです。Google OAuth の callback、共通アカウント ID を持つ Cookie セッションの発行、パスキー登録とパスキーログインを担当します。

## 環境変数

`api/.env.template` を `api/.env.local` としてコピーし、必要な値を設定してください。

`AUTH_SECRET` は Cookie セッションの署名に使うため、32 バイト以上の推測されにくい文字列にしてください。

認証完了後のデフォルト戻り先は `DEFAULT_REDIRECT_URL` で指定します。未指定の場合は、認証 web 内の `http://localhost:3000/mypage` を使います。別リポジトリの本体アプリへ戻す場合は、`ALLOWED_REDIRECT_URLS` に許可する URL をカンマ区切りで指定してください。`redirect_to` には、この許可リストに含まれる任意の戻り先を指定できます。

```txt
DEFAULT_REDIRECT_URL=http://localhost:3000/mypage
ALLOWED_REDIRECT_URLS=http://localhost:3000/mypage,http://localhost:3100,https://app.example.com
```

`mise run dev` で起動する場合は `api/.env.local` の値が使われます。`compose.yml` の `SANDBOX_ALLOWED_REDIRECT_URLS` は Docker Compose 起動時だけ有効です。

Google OAuth を使う場合は、Google Cloud Console の OAuth callback URL に次の URL を登録してください。

```txt
http://localhost:8080/auth/google/callback
```

## ローカル起動

```txt
mise run dev
```

環境変数を変更した場合は、起動中の API を止めてから `mise run dev` を実行し直してください。

通常の `go run` で起動する場合は、次のコマンドを使います。

```txt
mise run run
```

## migration

```txt
mise run migrate:up
mise run migrate:down
```

`accounts` に共通アカウント ID と WebAuthn user handle、`auth_identities` に OAuth の認証情報、`webauthn_credentials` と `webauthn_sessions` にパスキーの credential と ceremony session を保存します。

## エンドポイント

| メソッド | パス | 内容 |
| -------- | ---- | ---- |
| `GET` | `/health` | ヘルスチェックを返します。 |
| `GET` | `/me` | ログイン中のユーザー情報を返します。 |
| `POST` | `/auth/logout` | セッション Cookie を削除します。 |
| `GET` | `/auth/:provider/login` | OAuth ログインを開始します。`redirect_to` で認証後の戻り先を指定できます。 |
| `GET` | `/auth/:provider/callback` | OAuth callback を処理します。 |
| `POST` | `/passkeys/register/options` | パスキー登録用の公開鍵オプションを返します。 |
| `POST` | `/passkeys/register/verify` | パスキー登録結果を検証します。 |
| `POST` | `/passkeys/login/options` | パスキーログイン用の公開鍵オプションを返します。`redirect_to` で認証後の戻り先を指定できます。 |
| `POST` | `/passkeys/login/verify` | パスキーログイン結果を検証し、`redirectTo` を返します。 |

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

### 認証後に `/mypage` へ戻ってしまう

`redirect_to` の origin が `ALLOWED_REDIRECT_URLS` に含まれていない場合、API は `DEFAULT_REDIRECT_URL` に戻します。例えば次の URL で認証を開始する場合、

```txt
http://localhost:3000/login?redirect_to=http%3A%2F%2Flocalhost%3A3100%2Fdashboard
```

`api/.env.local` に `http://localhost:3100` を含めてから API を再起動してください。

```txt
ALLOWED_REDIRECT_URLS=http://localhost:3000/mypage,http://localhost:3100
```

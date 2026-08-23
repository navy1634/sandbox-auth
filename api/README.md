# api

Gin で動く認証 API サーバーです。Google OAuth の callback、共通アカウント ID を持つ Cookie セッションの発行、パスキー登録とパスキーログイン、外部アプリ向け OIDC Provider を担当します。外部アプリとの接続には OIDC を使います。

## 環境変数

`api/.env.template` を `api/.env.local` としてコピーし、必要な値を設定してください。

認証 web 自身の認証完了後のデフォルト戻り先は `DEFAULT_REDIRECT_URL` で指定します。未指定の場合は、認証 web 内の `http://localhost:3000/mypage` を使います。認証 web の `/login` やパスキー API から別の画面へ戻す場合は、`ALLOWED_REDIRECT_URLS` に許可する URL をカンマ区切りで指定し、`redirect_to` を使います。外部アプリはこの戻り先機能を使わず、下記の OIDC Provider に接続してください。

```txt
DEFAULT_REDIRECT_URL=http://localhost:3000/mypage
ALLOWED_REDIRECT_URLS=http://localhost:3000/mypage,http://localhost:3100,https://app.example.com
```

`mise run dev` で起動する場合は `api/.env.local` の値が使われます。`compose.yml` の `SANDBOX_ALLOWED_REDIRECT_URLS` は Docker Compose 起動時だけ有効です。

リバースプロキシ配下で起動する場合は、`TRUSTED_PROXIES` に信頼するプロキシの IP または CIDR をカンマ区切りで指定してください。未指定の場合は、`X-Forwarded-For` などの forwarded header を使わず、直接接続元の IP を使います。

```txt
TRUSTED_PROXIES=10.0.0.0/8,192.0.2.10
```

### OIDC Provider

`OIDC_CLIENTS` を設定すると、外部アプリから接続できるOIDC Providerを有効にします。起動時にクライアント情報をDBへ同期し、実行時のクライアント設定とredirect URIはDBを参照します。設定から削除したクライアントはDB上で無効化します。Issuer URLは、外部アプリからAPIへ到達できる公開URLを指定してください。KubernetesのIngress配下では `/api` まで含めます。

```txt
OIDC_ISSUER_URL=https://auth.example.com/api
OIDC_CLIENTS=[{"client_id":"external-app","client_secret":"replace-with-a-random-secret","redirect_uris":["https://app.example.com/oidc/callback"]}]
OIDC_SIGNING_KEY_FILE=/run/secrets/oidc-signing-key.pem
OIDC_KEY_ID=oidc-key-2026-01
```

`OIDC_SIGNING_KEY_FILE` には2048ビット以上のRSA秘密鍵を指定します。環境変数で渡す場合は、PEMの改行を `\n` として `OIDC_SIGNING_KEY` に設定できます。秘密鍵と `OIDC_CLIENTS` はSecret管理機能へ保存してください。クライアントシークレット、認可コード、アクセストークンはDBへ平文保存しません。

ProviderはAuthorization Code FlowとPKCE（S256）だけを受け付けます。外部アプリはDiscovery URLへアクセスして設定を取得してください。

```txt
GET https://auth.example.com/api/.well-known/openid-configuration
```

ログインセッション、認可トランザクション、認可コード、アクセストークンはPostgreSQLへ保存するため、複数レプリカで同じDBを共有できます。CookieとOIDCのopaque値はDBへ平文保存しません。OIDC subjectはアカウントへ不変値として保存され、署名鍵を変更しても変わりません。署名鍵は、既発行IDトークンの検証期間を考慮して運用してください。

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

`accounts` に共通アカウント ID と不変のOIDC subject、`auth_sessions` にハッシュ化したログインセッション、`auth_identities` に内部ログイン用OAuthの認証情報、`webauthn_credentials` と `webauthn_sessions` にパスキーのcredentialとceremony sessionを保存します。OIDCクライアント、redirect URI、認可トランザクション、認可コード、アクセストークンはライフサイクルが異なるため専用テーブルへ分けて保存します。期限切れのOIDC一時データは通常のOIDC処理で削除します。

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
| `GET` | `/.well-known/openid-configuration` | OIDC Discovery文書を返します。 |
| `GET` | `/.well-known/jwks.json` | IDトークン検証用の公開鍵を返します。 |
| `GET` | `/oidc/authorize` | OIDC認可要求を受け付けます。 |
| `GET` | `/oidc/authorize/resume` | OIDC認可後に内部ログイン画面へ戻すための内部 callback です。外部アプリから直接呼び出しません。 |
| `POST` | `/oidc/token` | 認可コードをTokenへ交換します。 |
| `GET` / `POST` | `/oidc/userinfo` | Bearer TokenからUserInfoを返します。 |

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

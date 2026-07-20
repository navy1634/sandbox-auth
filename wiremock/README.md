# wiremock

フロントエンドを API サーバーなしで確認するための WireMock スタブです。Google OAuth、セッション、プロフィール更新、パスキー登録、パスキーログインで使う API レスポンスを返します。

## 起動

ルートディレクトリで次のコマンドを実行します。

```txt
docker compose --profile mock up --build web wiremock
```

WireMock は `http://localhost:8080` で起動します。web のデフォルト接続先も `http://localhost:8080` なので、追加の環境変数は不要です。OAuth 完了後の戻り先は、単独確認用の `http://localhost:3000/mypage` です。

## スタブ

スタブ定義は `wiremock/mappings` にあります。ファイル名の先頭に数字を付けて、認証フローの順序や関連を追いやすくしています。

| 対象 | 内容 |
| ---- | ---- |
| `health` | ヘルスチェックを返します。 |
| `me` | Cookie の有無に応じてログイン状態を返します。 |
| `auth-google` | Google OAuth のログイン開始と callback を模擬します。 |
| `auth-logout` | セッション Cookie の削除を模擬します。 |
| `account-profile` | プロフィール更新を模擬します。 |
| `passkeys-login` | パスキーログインの options と verify を模擬します。 |
| `passkeys-register` | パスキー登録の options と verify を模擬します。 |

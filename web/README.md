# web

Next.js の認証フロントエンドです。Google OAuth ログイン、プロフィール登録、パスキー登録、パスキーログインの画面を提供し、認証完了後は別リポジトリの本体アプリへ戻します。

## 主な画面

| パス        | 内容                                                      |
| ----------- | --------------------------------------------------------- |
| `/`         | トップページです。                                        |
| `/login`    | Google OAuth とパスキーでログインできます。               |
| `/register` | Google ログイン後にプロフィールとパスキーを登録できます。 |
| `/mypage`   | リポジトリ単独で認証後の戻り先を確認できます。            |

## バックエンド連携

API の接続先は `NEXT_PUBLIC_API_BASE_URL`、認証完了後のデフォルト戻り先は `NEXT_PUBLIC_DEFAULT_REDIRECT_URL` で指定します。未指定の場合は、同じオリジンの `/mypage` を使います。

```txt
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
NEXT_PUBLIC_DEFAULT_REDIRECT_URL=http://localhost:3000/mypage
```

Docker Compose で起動する場合は、ルートディレクトリの `SANDBOX_API_URL` が `NEXT_PUBLIC_API_BASE_URL` に、`SANDBOX_DEFAULT_REDIRECT_URL` が `NEXT_PUBLIC_DEFAULT_REDIRECT_URL` に渡されます。

## ローカル起動

```txt
pnpm install
pnpm dev
```

起動後は `http://localhost:3000` にアクセスしてください。

## コマンド

| コマンド               | 内容                                     |
| ---------------------- | ---------------------------------------- |
| `pnpm dev`             | Next.js を開発起動します。               |
| `pnpm build`           | Next.js をビルドします。                 |
| `pnpm start`           | ビルド済みの Next.js を起動します。      |
| `pnpm lint`            | JavaScript と CSS の lint を実行します。 |
| `pnpm lint:js`         | oxlint を実行します。                    |
| `pnpm lint:css`        | stylelint を実行します。                 |
| `pnpm format`          | oxfmt で整形します。                     |
| `pnpm typecheck`       | TypeScript の型チェックを実行します。    |
| `pnpm storybook`       | Storybook を起動します。                 |
| `pnpm build-storybook` | Storybook をビルドします。               |

# web

Next.js のフロントエンドです。Google OAuth ログイン、プロフィール登録、パスキー登録、パスキーログインの画面を提供します。

## 主な画面

| パス | 内容 |
| ---- | ---- |
| `/` | トップページです。 |
| `/login` | Google OAuth とパスキーでログインできます。 |
| `/register` | Google ログイン後にプロフィールとパスキーを登録できます。 |
| `/mypage` | ログイン中のユーザー情報、プロフィール更新、パスキー登録を確認できます。 |

## バックエンド連携

API の接続先は `NEXT_PUBLIC_API_BASE_URL` で指定します。

```txt
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

Docker Compose で起動する場合は、ルートディレクトリの `SANDBOX_API_URL` が `NEXT_PUBLIC_API_BASE_URL` に渡されます。

## ローカル起動

```txt
pnpm install
pnpm dev
```

起動後は `http://localhost:3000` にアクセスしてください。

## コマンド

| コマンド | 内容 |
| -------- | ---- |
| `pnpm dev` | Next.js を開発起動します。 |
| `pnpm build` | Next.js をビルドします。 |
| `pnpm start` | ビルド済みの Next.js を起動します。 |
| `pnpm lint` | JavaScript と CSS の lint を実行します。 |
| `pnpm lint:js` | oxlint を実行します。 |
| `pnpm lint:css` | stylelint を実行します。 |
| `pnpm format` | oxfmt で整形します。 |
| `pnpm typecheck` | TypeScript の型チェックを実行します。 |
| `pnpm storybook` | Storybook を起動します。 |
| `pnpm build-storybook` | Storybook をビルドします。 |

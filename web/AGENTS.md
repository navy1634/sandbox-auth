# AGENTS.md

## 基本方針

- 回答は日本語の敬語で書いてください。
- 変更前に対象ファイルを読み、既存の書き方に合わせてください。
- 不要な整形、コメント削除、命名変更は避けてください。
- パッケージ操作、スクリプト実行、検証は pnpm を使ってください。

## プロジェクト概要

- Next.js 16、React 19 の App Router 構成です。
- 認証 web はローカルでは `http://localhost:3000` 固定で扱います。`pnpm dev` は 3000 番ポートを明示して起動します。
- 3000 番ポートが使用中の場合は、別ポートで作業を続けず、先に 3000 番を使っているプロセスを止めてください。API の `FRONTEND_URL`、OAuth callback、`redirect_to` の確認がずれます。
- アプリケーションコードは `src` 配下に集約されています。
- TypeScript の import alias は `@/*` が `./src/*` を指します。
- 静的アセットは Next.js の規約どおり、ルート直下の `public` に置きます。
- Storybook は `@storybook/nextjs-vite` を使います。

## 主要ディレクトリ

- `src/app` は Next.js App Router のルートです。
- `src/app/layout.tsx` は全体レイアウトで、Header と Footer を読み込みます。
- `src/app/page.tsx` はトップページです。
- `src/components` はアプリ内コンポーネントを置く場所です。
- `src/components/*/*.stories.tsx` と `src/components/*/*.mdx` は Storybook 用です。
- `.storybook/main.ts` は Storybook の探索対象を `src/components` に向けています。
- `public` は SVG などの公開静的ファイルを置く場所です。

## import ルール

- `src` 配下のアプリコードからコンポーネントを参照するときは `@/components/...` を使ってください。
- `@/src/...` という import は使わないでください。
- Storybook の型は `@storybook/nextjs-vite` から import してください。
- コンポーネントの props 型は、原則としてコンポーネント側に置いてください。Storybook 側に実装の型を依存させないでください。

## スクリプト

- 開発サーバーは `pnpm dev` で起動します。
- `pnpm dev` は `next dev --hostname 0.0.0.0 --port 3000` を実行します。
- 本番ビルドは `pnpm build` で確認します。
- lint は `pnpm lint` を使います。
- JavaScript と TypeScript の lint は `pnpm lint:js` で実行します。
- CSS lint は `pnpm lint:css` で実行します。
- format は `pnpm format` を使います。
- Storybook は `pnpm storybook` で起動します。
- Storybook の静的ビルドは `pnpm build-storybook` で確認します。

## lint と format

- `pnpm lint` は通る状態を目標にしてください。
- JavaScript と TypeScript の lint は `oxlint` を使います。
- format は `oxfmt` を使います。
- CSS lint は `stylelint` と `stylelint-config-standard` を使います。
- `stylelint` では既存の Storybook 由来の BEM 風クラス名を許容するため、`selector-class-pattern` を無効にしています。
- `oxfmt --check .` は既存ファイルの整形差分を広く検出する可能性があります。作業範囲外の一括整形は避けてください。

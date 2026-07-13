# web

Next.js のフロントエンドです。

## バックエンド連携

ログイン確認ページは、Gin バックエンドの API を使います。

```txt
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

Docker Compose でリモート端末から使う場合は、ルートディレクトリで `SANDBOX_API_URL` を指定します。

```txt
SANDBOX_API_URL=http://192.0.2.10:8080 docker compose up --build
```

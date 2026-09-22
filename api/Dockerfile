FROM golang:1.27.1-bookworm@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195 AS builder
WORKDIR /app
COPY . .
RUN --mount=type=cache,target=~/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build go build -o /app/bin/api ./src

FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
RUN groupadd --system api && useradd --system --gid api --home-dir /app --shell /usr/sbin/nologin api
COPY --from=builder --chown=api:api /app/bin/api /app/api
USER api
EXPOSE 8080
CMD ["/app/api"]

FROM golang:1.27-bookworm@sha256:d9b6a1beb9474adc2a1ce842573639b44b2a0dd49b6669f51d3821baf9ec37c9 AS builder
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

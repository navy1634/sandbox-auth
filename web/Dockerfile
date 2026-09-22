FROM node:26-alpine@sha256:dbaa92e5758cbbcf85d65d5403fdb530fe3442cbe8c6dbfb7ef23365450d5070 AS base
ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
RUN npm install -g pnpm@11.15.1

# Dependencies
FROM base AS deps
WORKDIR /app
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

# Build
FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN pnpm build
RUN pnpm prune --prod

# Runtime
FROM base AS runner
WORKDIR /app
ENV NODE_ENV=production
RUN chown node:node /app
COPY --from=builder --chown=node:node /app/package.json ./
COPY --from=builder --chown=node:node /app/public ./public
COPY --from=builder --chown=node:node /app/.next ./.next
COPY --from=builder --chown=node:node /app/node_modules ./node_modules
COPY --from=builder --chown=node:node /app/next.config.mjs ./
COPY --from=builder --chown=node:node /app/next-env.d.ts ./
USER node

EXPOSE 3000

CMD ["./node_modules/.bin/next", "start"]

FROM node:24.11-alpine AS builder
LABEL org.opencontainers.image.source=https://github.com/impactscope-organization/wobsongo/wa-bot
RUN corepack enable && corepack prepare pnpm@11.13.0 --activate
WORKDIR /app
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY . .
RUN pnpm run build
FROM node:24.11-alpine
RUN corepack enable && corepack prepare pnpm@11.13.0 --activate
WORKDIR /app
RUN addgroup -g 1001 -S nodejs \
    && adduser -S botuser -u 1001 -G nodejs
COPY --from=builder /app/dist ./dist
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --prod --frozen-lockfile
RUN mkdir -p /app/auth_info_baileys
RUN chown -R botuser:nodejs /app
USER botuser
EXPOSE 3000
CMD ["node", "dist/server.js"]
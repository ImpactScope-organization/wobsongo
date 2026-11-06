FROM node:24.11-alpine AS builder
LABEL org.opencontainers.image.source=https://github.com/impactscope-organization/wobsongo/frontend
RUN corepack enable && corepack prepare pnpm@11.5.3 --activate
WORKDIR /app
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml .npmrc ./
RUN pnpm install --frozen-lockfile
COPY . .
RUN pnpm run build

FROM node:24.11-alpine
RUN corepack enable && corepack prepare pnpm@11.5.3 --activate
WORKDIR /app
COPY --from=builder /app/build ./build
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml .npmrc ./
RUN pnpm install --prod --frozen-lockfile --ignore-scripts
RUN chown -R node:node /app
USER node
ENV NODE_ENV=production
EXPOSE 3000
CMD ["node", "build"]
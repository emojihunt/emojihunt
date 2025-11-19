FROM oven/bun:1.3-slim AS build

WORKDIR /build
COPY . .
RUN bun install

ENV NODE_ENV="production"
RUN bun run build

FROM oven/bun:1.3-slim
COPY --from=build /build/.output /app

USER root
WORKDIR /app
ENTRYPOINT ["bun", "server/index.mjs"]

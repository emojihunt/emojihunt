FROM oven/bun:1.2-slim AS build

WORKDIR /build
COPY . .
RUN bun install

ENV NODE_ENV="production"
RUN bun run build

WORKDIR .output/server
RUN rm -rf node_modules/ && \
    bun build --compile --minify --sourcemap index.mjs \
        --outfile /build/emojiweb  --target bun-linux-x64-musl

FROM alpine
RUN apk add --no-cache tzdata libstdc++
COPY --from=build /build/emojiweb /bin/emojiweb

USER root
ENTRYPOINT ["emojiweb"]

FROM golang:1.21-alpine AS build

WORKDIR /build
COPY . .
RUN apk add --no-cache build-base
# Flags are a workaround for https://github.com/mattn/go-sqlite3/issues/1164
RUN CGO_CFLAGS="-D_LARGEFILE64_SOURCE" go build -ldflags="-w -s" .

FROM alpine
RUN apk add --no-cache tzdata
COPY --from=build /build/emojihunt /bin/huntbot

USER root
WORKDIR /state
ENTRYPOINT ["huntbot"]

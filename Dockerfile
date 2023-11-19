FROM golang:1.21-alpine AS build

WORKDIR /build
COPY . .
RUN apk add --no-cache build-base
RUN go build -ldflags="-w -s" .

FROM alpine
RUN apk add --no-cache tzdata
COPY --from=build /build/emojihunt /bin/huntbot

USER root
WORKDIR /state
ENTRYPOINT ["huntbot"]

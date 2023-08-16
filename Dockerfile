FROM golang:1.21-alpine AS build

WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" .

FROM alpine
COPY --from=build /build/emojihunt /bin/huntbot

USER root
WORKDIR /state
ENTRYPOINT ["huntbot"]

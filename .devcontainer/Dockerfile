# syntax = btidor/syntax

FROM mcr.microsoft.com/vscode/devcontainers/base:0-jammy

ADD --apt sqlite3

# Install Go
RUN (curl -sL https://golang.org/dl/go1.21.4.linux-amd64.tar.gz | tar xzC /usr/local) && \
    (echo 'PATH=$PATH:/usr/local/go/bin' >> /home/vscode/.profile) && \
    GOPATH=/usr/local/go PATH=$PATH:/usr/local/go/bin go install \
         github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Install flyctl
ENV FLYCTL_INSTALL=/usr/local
RUN (curl https://fly.io/install.sh | sh)

# syntax = btidor/syntax

FROM mcr.microsoft.com/vscode/devcontainers/base:noble

COPY *.sources /etc/apt/sources.list.d/
ADD --apt ngrok nodejs sqlite3

# Install Go
RUN (curl -sL https://golang.org/dl/go1.23.4.linux-amd64.tar.gz | tar xzC /usr/local) && \
    (echo 'PATH=$PATH:/usr/local/go/bin' >> /home/vscode/.profile) && \
    GOPATH=/usr/local/go PATH=$PATH:/usr/local/go/bin go install \
         github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Install CLIs
RUN (curl -fsSL https://fly.io/install.sh | FLYCTL_INSTALL=/usr/local sh)
RUN (curl -fsSL https://bun.sh/install | BUN_INSTALL=/usr/local bash)
RUN npm install -g vercel

FROM mcr.microsoft.com/vscode/devcontainers/base:0-jammy

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get upgrade -y

# Install Go
RUN (curl -sL https://golang.org/dl/go1.21.3.linux-amd64.tar.gz | tar xzC /usr/local) && \
    (echo 'PATH=$PATH:/usr/local/go/bin' >> /home/vscode/.profile)

# Install flyctl
ENV FLYCTL_INSTALL=/usr/local
RUN (curl https://fly.io/install.sh | sh)

ARG VARIANT="jammy"
FROM mcr.microsoft.com/vscode/devcontainers/base:0-${VARIANT}

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get upgrade -y

# Install Go
RUN (curl -sL https://golang.org/dl/go1.19.5.linux-amd64.tar.gz | tar xzC /usr/local) && \
    (echo 'PATH=$PATH:/usr/local/go/bin' >> /home/vscode/.profile)

# Install Google Cloud SDK
RUN (curl -sL https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-412.0.0-linux-x86_64.tar.gz | tar xzC /usr/local) && \
    (ln -s /usr/local/google-cloud-sdk/bin/gcloud /usr/local/bin)

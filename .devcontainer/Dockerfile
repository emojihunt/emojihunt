# See here for image contents: https://github.com/microsoft/vscode-dev-containers/tree/v0.209.6/containers/ubuntu/.devcontainer/base.Dockerfile

# [Choice] Ubuntu version (use hirsuite or bionic on local arm64/Apple Silicon): hirsute, focal, bionic
ARG VARIANT="hirsute"
FROM mcr.microsoft.com/vscode/devcontainers/base:0-${VARIANT}

# [Optional] Uncomment this section to install additional OS packages.
# RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
#     && apt-get -y install --no-install-recommends <your-package-list-here>

# Install Go
RUN (curl -sL https://golang.org/dl/go1.17.5.linux-amd64.tar.gz | tar xzC /usr/local) && \
    (echo 'PATH=$PATH:/usr/local/go/bin' >> /home/vscode/.profile)

# Install Google Cloud SDK
RUN (curl -sL https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-367.0.0-linux-x86_64.tar.gz | tar xzC /usr/local) && \
    (ln -s /usr/local/google-cloud-sdk/bin/gcloud /usr/local/bin)

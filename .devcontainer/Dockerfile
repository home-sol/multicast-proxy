ARG GO_VERSION="1.20"
FROM mcr.microsoft.com/vscode/devcontainers/go:${GO_VERSION}-bullseye


RUN apt update \
    && apt install -y \
    libpcap-dev \
    && rm -rf /var/lib/apt/lists/*
    

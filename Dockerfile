FROM ubuntu:22.04

RUN apt update \
    && apt install -y \
        libpcap-dev \
    && rm -rf /var/lib/apt/lists/*

COPY multicast-proxy /usr/bin/multicast-proxy

CMD ["multicast-proxy", "serve"]
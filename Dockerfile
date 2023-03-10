FROM alpine
COPY multicast-proxy /usr/bin/multicast-proxy
ENTRYPOINT /usr/bin/multicast-proxy
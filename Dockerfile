FROM alpine
MAINTAINER xujie xujieasd@gmail.com
RUN apk add --no-cache \
    iptables \
    ipset \
    conntrack-tools \
    curl \
    bash
COPY enn-policy /

ENTRYPOINT ["/enn-policy"]
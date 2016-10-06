FROM alpine:3.4

MAINTAINER Eric Chiang <eric.chiang@coreos.com>

RUN apk add --update ca-certificates 

COPY _output/bin/dex /usr/local/bin/dex

ENTRYPOINT ["/usr/local/bin/dex"]

CMD ["version"]

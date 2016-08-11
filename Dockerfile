FROM alpine:latest

MAINTAINER Eric Chiang <eric.chiang@coreos.com>

RUN apk add --update ca-certificates 

COPY bin/dex /dex

ENTRYPOINT ["/dex"]

CMD ["version"]

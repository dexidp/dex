FROM alpine:latest

MAINTAINER Eric Chiang <eric.chiang@coreos.com>

RUN apk add --update ca-certificates 

COPY bin/poke /poke

ENTRYPOINT ["/poke"]

CMD ["version"]

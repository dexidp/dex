FROM golang:1.8-alpine

MAINTAINER Travis Vachon <travis@circleci.com>
MAINTAINER Lewis Liu <lewis@circleci.com>
MAINTAINER Alexander Mann <alex@circleci.com>

RUN apk update
RUN apk add wget

ENV DOCKERIZE_VERSION v0.3.0
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz

ADD . /go/src/github.com/coreos/dex/compose
WORKDIR /go/src/github.com/coreos/dex/compose
ENTRYPOINT ["/go/src/github.com/coreos/dex/compose/docker-entrypoint.sh"]
EXPOSE 5556 5557
CMD ["start"]

FROM golang:1.8

MAINTAINER Travis Vachon <travis@circleci.com>
MAINTAINER Lewis Liu <lewis@circleci.com>

ENV DEX_HOME /go/src/github.com/coreos/dex
ENV DEX_VERSION 2.2.5

RUN apt-get update -y && \
    apt-get install sqlite3 -y

RUN apt-get update && apt-get install -y wget

ENV DOCKERIZE_VERSION v0.3.0
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz

RUN mkdir -p $DEX_HOME

RUN curl -L https://github.com/coreos/dex/archive/v$DEX_VERSION.tar.gz | tar zx && \
    cp -R dex-$DEX_VERSION/* $DEX_HOME && \
    rm -rf dex-$DEX_VERSION

RUN cd $DEX_HOME && \
    make

WORKDIR $DEX_HOME

COPY config.yml $DEX_HOME
COPY init.sh $DEX_HOME

RUN chmod +x init.sh

EXPOSE 5556 5557

FROM golang:1.13-alpine as builder

ENV BASE_DIR /go/src/github.com/dexidp/dex
ENV GO111MODULE on

RUN apk add --update git gcc libc-dev

WORKDIR ${BASE_DIR}

COPY . ${BASE_DIR}/

RUN go mod tidy -v
RUN go mod verify
RUN go build -v -o ${BASE_DIR}/bin/dex ${BASE_DIR}/cmd/dex

FROM alpine:3.10

ENV BASE_DIR /go/src/github.com/dexidp/dex
LABEL source=git@github.com:kyma-incubator/dex.git
# Dex connectors, such as GitHub and Google logins require root certificates.
# Proper installations should manage those certificates, but it's a bad user
# experience when this doesn't work out of the box.
#
# OpenSSL is required so wget can query HTTPS endpoints for health checking.
RUN apk add --update ca-certificates openssl

USER 1001:1001
COPY --from=builder ${BASE_DIR}/bin/dex /usr/local/bin/dex

COPY ./licenses/ /licenses

# Import frontend assets and set the correct CWD directory so the assets
# are in the default path.
COPY web /web
WORKDIR /

ENTRYPOINT ["dex"]

CMD ["version"]

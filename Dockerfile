FROM golang:1.15.6-alpine3.12

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT=""

WORKDIR /go/src/github.com/dexidp/dex

ENV GOOS=${TARGETOS} \
  GOARCH=${TARGETARCH} \
  GOARM=${TARGETVARIANT}

RUN apk add --no-cache --update alpine-sdk

ARG GOPROXY

COPY go.mod go.sum ./
COPY api/v2/go.mod api/v2/go.sum ./api/v2/
RUN go mod download

COPY . .

RUN make release-binary

FROM alpine:3.12

WORKDIR /

# Dex connectors, such as GitHub and Google logins require root certificates.
# Proper installations should manage those certificates, but it's a bad user
# experience when this doesn't work out of the box.
#
# OpenSSL is required so wget can query HTTPS endpoints for health checking.
RUN apk add --no-cache --update ca-certificates openssl

USER 1001:1001

COPY --from=0 /go/bin/dex /usr/local/bin/dex

# Copy module dependencies for CVE scanning / dependency analysis.
COPY go.mod go.sum                 /opt/dex/dependencies/
COPY api/v2/go.mod api/v2/go.sum   /opt/dex/dependencies/api/v2/

# Import frontend assets and set the correct CWD directory so the assets
# are in the default path.
COPY web web

ENTRYPOINT ["dex"]

CMD ["version"]

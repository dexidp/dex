FROM golang:1.15.7-alpine3.12

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT=""

WORKDIR /usr/local/src/dex

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

FROM alpine:3.13.0

WORKDIR /

# Dex connectors, such as GitHub and Google logins require root certificates.
# Proper installations should manage those certificates, but it's a bad user
# experience when this doesn't work out of the box.
#
# OpenSSL is required so wget can query HTTPS endpoints for health checking.
RUN apk add --no-cache --update ca-certificates openssl

RUN mkdir -p /var/dex
RUN chown -R 1001:1001 /var/dex

# Copy module files for CVE scanning / dependency analysis.
COPY --from=0 /usr/local/src/dex/go.mod /usr/local/src/dex/go.sum /usr/local/src/dex/
COPY --from=0 /usr/local/src/dex/api/v2/go.mod /usr/local/src/dex/api/v2/go.sum /usr/local/src/dex/api/v2/

COPY --from=0 /go/bin/dex /usr/local/bin/dex

USER 1001:1001

# Import frontend assets and set the correct CWD directory so the assets
# are in the default path.
COPY web web

ENTRYPOINT ["dex"]

CMD ["version"]

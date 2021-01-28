FROM golang:1.15.7-alpine3.13 AS builder

WORKDIR /usr/local/src/dex

RUN apk add --no-cache --update alpine-sdk

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT=""
ARG GOMPLATE_VERSION=v3.9.0

ENV GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT}

ARG GOPROXY

RUN wget -O /usr/local/bin/gomplate \
  "https://github.com/hairyhenderson/gomplate/releases/download/${GOMPLATE_VERSION}/gomplate_${GOOS:-linux}-${GOARCH:-amd64}${GOARM}" \
  && chmod +x /usr/local/bin/gomplate

COPY go.mod go.sum ./
COPY api/v2/go.mod api/v2/go.sum ./api/v2/
RUN go mod download

COPY . .

RUN make release-binary

FROM alpine:3.13.1

# Dex connectors, such as GitHub and Google logins require root certificates.
# Proper installations should manage those certificates, but it's a bad user
# experience when this doesn't work out of the box.
#
# OpenSSL is required so wget can query HTTPS endpoints for health checking.

RUN apk add --no-cache --update ca-certificates openssl

RUN mkdir -p /var/dex
RUN chown -R 1001:1001 /var/dex

RUN mkdir -p /etc/dex
COPY examples/config-example.tmpl /etc/dex/config.tmpl
RUN chown -R 1001:1001 /etc/dex

# Copy module files for CVE scanning / dependency analysis.
COPY --from=builder /usr/local/src/dex/go.mod /usr/local/src/dex/go.sum /usr/local/src/dex/
COPY --from=builder /usr/local/src/dex/api/v2/go.mod /usr/local/src/dex/api/v2/go.sum /usr/local/src/dex/api/v2/

COPY --from=builder /usr/local/bin/gomplate /usr/local/bin/gomplate
COPY --from=builder /go/bin/dex /usr/local/bin/dex

USER 1001:1001

# Import frontend assets and set the correct CWD directory so the assets
# are in the default path.
COPY --from=builder /usr/local/src/dex/web /web

USER 1001:1001

COPY entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]
CMD ["serve", "/etc/dex/config.tmpl"]

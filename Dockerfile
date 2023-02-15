ARG BASE_IMAGE=alpine

FROM golang:1.20.1-alpine3.16 AS builder

WORKDIR /usr/local/src/dex

RUN apk add --no-cache --update alpine-sdk ca-certificates openssl

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT=""

ENV GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT}

ARG GOPROXY

COPY go.mod go.sum ./
COPY api/v2/go.mod api/v2/go.sum ./api/v2/
RUN go mod download

COPY . .

RUN make release-binary

FROM alpine:3.17.2 AS stager

RUN mkdir -p /var/dex
RUN mkdir -p /etc/dex
COPY config.docker.yaml /etc/dex/

FROM alpine:3.17.2 AS gomplate

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

ENV GOMPLATE_VERSION=v3.11.3

RUN wget -O /usr/local/bin/gomplate \
    "https://github.com/hairyhenderson/gomplate/releases/download/${GOMPLATE_VERSION}/gomplate_${TARGETOS:-linux}-${TARGETARCH:-amd64}${TARGETVARIANT}" \
    && chmod +x /usr/local/bin/gomplate

# For Dependabot to detect base image versions
FROM alpine:3.17.2 AS alpine
FROM gcr.io/distroless/static:latest AS distroless

FROM $BASE_IMAGE

# Dex connectors, such as GitHub and Google logins require root certificates.
# Proper installations should manage those certificates, but it's a bad user
# experience when this doesn't work out of the box.
#
# See https://go.dev/src/crypto/x509/root_linux.go for Go root CA bundle locations.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --from=stager --chown=1001:1001 /var/dex /var/dex
COPY --from=stager --chown=1001:1001 /etc/dex /etc/dex

# Copy module files for CVE scanning / dependency analysis.
COPY --from=builder /usr/local/src/dex/go.mod /usr/local/src/dex/go.sum /usr/local/src/dex/
COPY --from=builder /usr/local/src/dex/api/v2/go.mod /usr/local/src/dex/api/v2/go.sum /usr/local/src/dex/api/v2/

COPY --from=builder /go/bin/dex /usr/local/bin/dex
COPY --from=builder /go/bin/docker-entrypoint /usr/local/bin/docker-entrypoint
COPY --from=builder /usr/local/src/dex/web /srv/dex/web

COPY --from=gomplate /usr/local/bin/gomplate /usr/local/bin/gomplate

USER 1001:1001

ENTRYPOINT ["/usr/local/bin/docker-entrypoint"]
CMD ["dex", "serve", "/etc/dex/config.docker.yaml"]

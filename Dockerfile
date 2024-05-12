ARG BASE_IMAGE=alpine

FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.4.0@sha256:0cd3f05c72d6c9b038eb135f91376ee1169ef3a330d34e418e65e2a5c2e9c0d4 AS xx

FROM --platform=$BUILDPLATFORM golang:1.22.3-alpine3.18@sha256:45319271acc6318e717a16a8f79539cffbee77cebd0602b32f4e55c26db9f78e AS builder

COPY --from=xx / /

RUN apk add --update alpine-sdk ca-certificates openssl clang lld

ARG TARGETPLATFORM

RUN xx-apk --update add musl-dev gcc

# lld has issues building static binaries for ppc so prefer ld for it
RUN [ "$(xx-info arch)" != "ppc64le" ] || XX_CC_PREFER_LINKER=ld xx-clang --setup-target-triple

RUN xx-go --wrap

WORKDIR /usr/local/src/dex

ARG GOPROXY

ENV CGO_ENABLED=1

COPY go.mod go.sum ./
COPY api/v2/go.mod api/v2/go.sum ./api/v2/
RUN go mod download

COPY . .

# Propagate Dex version from build args to the build environment
ARG VERSION
RUN make release-binary

RUN xx-verify /go/bin/dex && xx-verify /go/bin/docker-entrypoint

FROM alpine:3.19.1@sha256:c5b1261d6d3e43071626931fc004f70149baeba2c8ec672bd4f27761f8e1ad6b AS stager

RUN mkdir -p /var/dex
RUN mkdir -p /etc/dex
COPY config.docker.yaml /etc/dex/

FROM alpine:3.19.1@sha256:c5b1261d6d3e43071626931fc004f70149baeba2c8ec672bd4f27761f8e1ad6b AS gomplate

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

ENV GOMPLATE_VERSION=v3.11.7

RUN wget -O /usr/local/bin/gomplate \
  "https://github.com/hairyhenderson/gomplate/releases/download/${GOMPLATE_VERSION}/gomplate_${TARGETOS:-linux}-${TARGETARCH:-amd64}${TARGETVARIANT}" \
  && chmod +x /usr/local/bin/gomplate

# For Dependabot to detect base image versions
FROM alpine:3.19.1@sha256:c5b1261d6d3e43071626931fc004f70149baeba2c8ec672bd4f27761f8e1ad6b AS alpine
FROM gcr.io/distroless/static-debian12:nonroot@sha256:42c88655018248be5b8ed88261fca620f6d0dd55996fbb591de4969f250120e9 AS distroless

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

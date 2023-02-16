ARG BASE_IMAGE=alpine

#
# --- Artifact: CA certificates and configs ---
#
FROM alpine:3.17.2 AS stager

RUN apk add --no-cache --update ca-certificates

RUN mkdir -p /var/dex
RUN mkdir -p /etc/dex

COPY config.docker.yaml /etc/dex/
COPY go.mod go.sum /usr/local/src/dex/
COPY api/v2/go.mod api/v2/go.sum /usr/local/src/dex/api/v2/

#
# --- Artifact: Gomplate ---
#
FROM alpine:3.17.2 AS gomplate

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

ENV GOMPLATE_VERSION=v3.11.4

RUN wget -O /usr/local/bin/gomplate \
    "https://github.com/hairyhenderson/gomplate/releases/download/${GOMPLATE_VERSION}/gomplate_${TARGETOS:-linux}-${TARGETARCH:-amd64}${TARGETVARIANT}" \
    && chmod +x /usr/local/bin/gomplate

#
# --- Final image: Dex ---
#
# For Dependabot to detect base image versions
FROM alpine:3.17.2 AS alpine

FROM gcr.io/distroless/static:latest AS distroless

FROM $BASE_IMAGE

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

# Dex connectors, such as GitHub and Google logins require root certificates.
# Proper installations should manage those certificates, but it's a bad user
# experience when this doesn't work out of the box.
#
# See https://go.dev/src/crypto/x509/root_linux.go for Go root CA bundle locations.
COPY --from=stager /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --from=stager --chown=1001:1001 /var/dex /var/dex
COPY --from=stager --chown=1001:1001 /etc/dex /etc/dex

# Copy module files for CVE scanning / dependency analysis.
COPY --from=stager --chown=1001:1001 /usr/local/src/dex/ /usr/local/src/dex/

COPY --chown=1001:1001 ./web /srv/dex/web
COPY --chown=1001:1001 ./bin/dex-${TARGETOS:-linux}-${TARGETARCH:-amd64}${TARGETVARIANT} /usr/local/bin/dex
COPY --chown=1001:1001 ./bin/docker-entrypoint-${TARGETOS:-linux}-${TARGETARCH:-amd64}${TARGETVARIANT} /usr/local/bin/docker-entrypoint

COPY --from=gomplate /usr/local/bin/gomplate /usr/local/bin/gomplate

USER 1001:1001

ENTRYPOINT ["/usr/local/bin/docker-entrypoint"]
CMD ["dex", "serve", "/etc/dex/config.docker.yaml"]

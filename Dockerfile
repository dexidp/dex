FROM alpine:3.4

MAINTAINER Ed Rooth <ed.rooth@coreos.com>
MAINTAINER Eric Chiang <eric.chiang@coreos.com>
MAINTAINER Rithu John <rithu.john@coreos.com>

# Dex connectors, such as GitHub and Google logins require root certificates.
# Proper installations should manage those certificates, but it's a bad user
# experience when this doesn't work out of the box.
#
# OpenSSL is required so wget can query HTTPS endpoints for health checking.
RUN apk add --update ca-certificates openssl

COPY _output/bin/dex /usr/local/bin/dex

# Import frontend assets and set the correct CWD directory so the assets
# are in the default path.
COPY web /web
WORKDIR /

ENTRYPOINT ["dex"]

CMD ["version"]

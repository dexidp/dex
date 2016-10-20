FROM alpine

MAINTAINER eric.chiang@coreos.com

# groll installs soelim, which is required by the build.

RUN apk add --update alpine-sdk openssl-dev db-dev groff

ADD assets/openldap-2.4.44 /openldap-2.4.44

WORKDIR /openldap-2.4.44

RUN ./configure

RUN make depend

RUN make

RUN make install

RUN apk del groff alpine-sdk

ADD scripts/entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]

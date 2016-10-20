#!/bin/sh -e

# Provide sane defaults for these values.
DOMAIN=${LDAP_DOMAIN:-"dc=example,dc=com"}
ROOT_CN=${LDAP_ROOT_CN:-"cn=Manager"}
ROOT_PW=${LDAP_ROOT_PW:-"secret"}
LOG_LEVEL=${LDAP_LOG_LEVEL:-"any"}

ROOT_DN="$ROOT_CN,$DOMAIN"

cat <<EOF > /usr/local/etc/openldap/slapd.ldif
# Global config
dn: cn=config
objectClass: olcGlobal
cn: config

# Schema definition
dn: cn=schema,cn=config
objectClass: olcSchemaConfig
cn: schema

include: file:///usr/local/etc/openldap/schema/core.ldif

# Default frontend configuration.
dn: olcDatabase=frontend,cn=config
objectClass: olcDatabaseConfig
objectClass: olcFrontendConfig
olcDatabase: frontend

# Template in RootDN values and RootPW.
dn: olcDatabase=mdb,cn=config
objectClass: olcDatabaseConfig
objectClass: olcMdbConfig
olcDatabase: mdb
OlcDbMaxSize: 1073741824
olcSuffix: $DOMAIN
olcRootDN: $ROOT_DN
olcRootPW: $ROOT_PW
olcDbDirectory: /usr/local/var/openldap-data
olcDbIndex: objectClass eq
EOF

mkdir -p /usr/local/etc/cn=config

/usr/local/sbin/slapadd \
    -n 0 \
    -F /usr/local/etc/cn=config \
    -l /usr/local/etc/openldap/slapd.ldif

# Begin slapd with `-d` so it attaches rather than running it as a daemon process.
/usr/local/libexec/slapd \
    -d $LOG_LEVEL \
    -F /usr/local/etc/cn=config

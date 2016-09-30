#!/bin/sh -e

DOMAIN=${LDAP_DOMAIN:-"dc=example,dc=com"}
ROOT_PW=${LDAP_ROOT_PW:-"secret"}
LOG_LEVEL=${LDAP_LOG_LEVEL:-"any"}

cat <<EOF > /usr/local/etc/openldap/slapd.ldif
dn: olcDatabase=mdb,cn=config 
objectClass: olcDatabaseConfig 
objectClass: olcMdbConfig 
olcDatabase: mdb 
OlcDbMaxSize: 1073741824 
olcSuffix: $DOMAIN
olcRootDN: cn=Manager,$DOMAIN
olcRootPW: $ROOT_PW 
olcDbDirectory: /usr/local/var/openldap-data 
olcDbIndex: objectClass eq
EOF

mkdir -p /usr/local/etc/cn=config

/usr/local/sbin/slapadd \
	-F /usr/local/etc/cn=config \
	-l /usr/local/etc/openldap/slapd.ldif

/usr/local/libexec/slapd \
	-d $LOG_LEVEL \
	-F /usr/local/etc/cn=config

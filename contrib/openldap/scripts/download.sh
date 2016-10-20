#!/bin/bash -e
# USAGE: scripts/download.sh

wget -O /tmp/openldap-2.4.44.tgz ftp://ftp.openldap.org/pub/OpenLDAP/openldap-release/openldap-2.4.44.tgz
sha512sum -c scripts/openldap-2.4.44.tgz.sha512
mv /tmp/openldap-2.4.44.tgz assets/openldap-2.4.44.tgz
tar -zxvf assets/openldap-2.4.44.tgz -C assets

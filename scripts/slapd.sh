#!/bin/bash -e

if ! [[ "$0" =~ "scripts/slapd.sh" ]]; then
	echo "This script must be run in a toplevel dex directory"
	exit 255
fi

command -v slapd >/dev/null 2>&1 || { 
    echo >&2 "OpenLDAP not installed. Install using one of the following commands:

   brew install openldap

   sudo dnf -y install openldap-servers openldap-clients 

   sudo apt-get install slapd ldap-utils

   Note: certain OpenLDAP packages may include AppArmor or SELinux configurations which prevent actions this script takes, such as referencing config files outside of its default config directory.
"; exit 1;
}

TEMPDIR=$( mktemp -d )

trap "{ rm -r $TEMPDIR ; exit 255; }" EXIT

CONFIG_DIR=$PWD/connector/ldap/testdata

# Include the schema files in the connector test directory. Installing OpenLDAP installs
# these in /etc somewhere, but the path isn't reliable across installs. Easier to ship
# the schema files directly.
for config in $( ls $CONFIG_DIR/*.schema ); do
    echo "include $config" >> $TEMPDIR/config
done

DATA_DIR=$TEMPDIR/data
mkdir $DATA_DIR

# Config template copied from:
# http://www.zytrax.com/books/ldap/ch5/index.html#step1-slapd
cat << EOF >> $TEMPDIR/config
# MODULELOAD definitions
# not required (comment out) before version 2.3
moduleload back_bdb.la

database bdb
suffix "dc=example,dc=org"

# root or superuser
rootdn "cn=admin,dc=example,dc=org"
rootpw admin
# The database directory MUST exist prior to running slapd AND 
# change path as necessary
directory	$DATA_DIR

# Indices to maintain for this directory
# unique id so equality match only
index	uid	eq
# allows general searching on commonname, givenname and email
index	cn,gn,mail eq,sub
# allows multiple variants on surname searching
index sn eq,sub
# sub above includes subintial,subany,subfinal
# optimise department searches
index ou eq
# if searches will include objectClass uncomment following
# index objectClass eq
# shows use of default index parameter
index default eq,sub
# indices missing - uses default eq,sub
index telephonenumber

# other database parameters
# read more in slapd.conf reference section
cachesize 10000
checkpoint 128 15
EOF

SLAPD_PID=""
trap "kill $SLAPD_PID" SIGINT

# Background the LDAP daemon so we can run an LDAP add command.
slapd \
    -d any \
    -h "ldap://localhost:10389/" \
    -f $TEMPDIR/config &
SLAPD_PID=$!

# Wait for server to come up.
time sleep 1

# Seed the initial set of users. Edit these values to change the initial
# set of users.
ldapadd \
    -x \
    -D "cn=admin,dc=example,dc=org" \
    -w admin \
    -H ldap://localhost:10389/ \
    -f $PWD/examples/config-ldap.ldif 

# Wait for slapd to exit.
wait $SLAPD_PID

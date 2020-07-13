#!/bin/bash
#
# Start an OpenLDAP container and populate it with example entries.
# https://github.com/dexidp/dex/blob/master/Documentation/connectors/ldap.md
#
# Usage:
#   slapd.sh          Kill a possibly preexisting "ldap" container, start a new one, and populate the directory.
#   slapd.sh --keep   Same, but keep the container if it is already running.
#
set -eu
cd -- "$(dirname "$0")/.."

run_cmd() {
	echo ">" "$@" >&2
	"$@"
}

keep_running=
if [ $# -gt 0 ] && [ "$1" = "--keep" ]; then
	keep_running=1
fi

if [ -z "$keep_running" ] || [ "$(docker inspect --format="{{.State.Running}}" ldap 2> /dev/null)" != "true" ]; then
	echo "LDAP container not running, or running and --keep not specified."
	echo "Removing old LDAP container (if any)..."
	run_cmd docker rm --force ldap || true
	echo "Starting LDAP container..."
	# Currently the most popular OpenLDAP image on Docker Hub. Comes with the latest version OpenLDAP 2.4.50.
	run_cmd docker run -p 389:389 -p 636:636 -v $PWD:$PWD --name ldap --detach osixia/openldap:1.4.0

	tries=1
	max_tries=10
	echo "Waiting for LDAP container ($tries/$max_tries)..."
	# Wait until expected line "structuralObjectClass: organization" shows up.
	# Seems to work more reliably than waiting for exit code 0. That would be:
	#   while ! docker exec ldap slapcat -b "dc=example,dc=org" > /dev/null 2>&1; do
	while [[ ! "$(docker exec ldap slapcat -b "dc=example,dc=org" 2>/dev/null)" =~ organization ]]; do
		((++tries))
		if [ "$tries" -gt "$max_tries" ]; then
			echo "ERROR: Timeout waiting for LDAP container."
			exit 1
		fi
		sleep 1
		echo "Waiting for LDAP container ($tries/$max_tries)..."
	done
fi

echo "Adding example entries to directory..."
run_cmd docker exec ldap ldapadd \
	-x \
	-D "cn=admin,dc=example,dc=org" \
	-w admin \
	-H ldap://localhost:389/ \
	-f $PWD/examples/config-ldap.ldif

echo "OK."

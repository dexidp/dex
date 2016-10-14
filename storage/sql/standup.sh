#!/bin/bash

if [ "$EUID" -ne 0 ]
  then echo "Please run as root"
  exit
fi

function usage {
  cat << EOF >> /dev/stderr
Usage: sudo ./standup.sh [create|destroy] [postgres|mysql|cockroach]

This is a script for standing up test databases. It uses systemd to daemonize
rkt containers running on a local loopback IP.

The general workflow is to create a daemonized container, use the output to set
the test environment variables, run the tests, then destroy the container.

	sudo ./standup.sh create postgres
	# Copy environment variables and run tests.
	go test -v -i # always install test dependencies
	go test -v
	sudo ./standup.sh destroy postgres

EOF
  exit 2
}

function main {
  if [ "$#" -ne 2 ]; then
    usage
    exit 2
  fi

  case "$1" in
  "create")
     case "$2" in
     "postgres")
        create_postgres;;
     "mysql")
        create_mysql;;
     *)
       usage
       exit 2
       ;;
     esac        
     ;;
  "destroy")
     case "$2" in
     "postgres")
        destroy_postgres;;
     "mysql")
        destroy_mysql;;
     *)
       usage
       exit 2
       ;;
     esac        
     ;;
  *)
    usage
    exit 2
    ;;
  esac
}

function wait_for_file {
  while [ ! -f $1 ]; do
    sleep 1
  done
}

function wait_for_container {
  while [ -z "$( rkt list --full | grep $1 | grep running )" ]; do
    sleep 1
  done
}

function create_postgres {
  UUID_FILE=/tmp/dex-postgres-uuid
  if [ -f $UUID_FILE ]; then
    echo "postgres database already exists, try ./standup.sh destroy postgres"
    exit 2
  fi

  echo "Starting postgres. To view progress run:"
  echo ""
  echo "  journalctl -fu dex-postgres"
  echo ""
  systemd-run --unit=dex-postgres \
      rkt run --uuid-file-save=$UUID_FILE --insecure-options=image docker://postgres:9.6

  wait_for_file $UUID_FILE

  UUID=$( cat $UUID_FILE )
  wait_for_container $UUID
  HOST=$( rkt list --full | grep "$UUID" | awk '{ print $NF }' | sed -e 's/default:ip4=//g' )
  echo "To run tests export the following environment variables:"
  echo ""
  echo "  export DEX_POSTGRES_DATABASE=postgres; export DEX_POSTGRES_USER=postgres; export DEX_POSTGRES_PASSWORD=postgres; export DEX_POSTGRES_HOST=$HOST:5432"
  echo ""
}

function destroy_postgres {
  UUID_FILE=/tmp/dex-postgres-uuid
  systemctl stop dex-postgres
  rkt rm --uuid-file=$UUID_FILE
  rm $UUID_FILE
}


main $@

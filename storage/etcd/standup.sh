#!/bin/bash

if [ "$EUID" -ne 0 ]
  then echo "Please run as root"
  exit
fi

function usage {
  cat << EOF >> /dev/stderr
Usage: sudo ./standup.sh [create|destroy] [etcd]

This is a script for standing up test databases. It uses systemd to daemonize
rkt containers running on a local loopback IP.

The general workflow is to create a daemonized container, use the output to set
the test environment variables, run the tests, then destroy the container.

	sudo ./standup.sh create etcd
	# Copy environment variables and run tests.
	go test -v -i # always install test dependencies
	go test -v
	sudo ./standup.sh destroy etcd

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
     "etcd")
        create_etcd;;
     *)
       usage
       exit 2
       ;;
     esac
     ;;
  "destroy")
     case "$2" in
     "etcd")
        destroy_etcd;;
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

function create_etcd {
  UUID_FILE=/tmp/dex-etcd-uuid
  if [ -f $UUID_FILE ]; then
    echo "etcd database already exists, try ./standup.sh destroy etcd"
    exit 2
  fi

  echo "Starting etcd . To view progress run:"
  echo ""
  echo "  journalctl -fu dex-etcd"
  echo ""
  UNIFIED_CGROUP_HIERARCHY=no \
  systemd-run --unit=dex-etcd \
      rkt run --uuid-file-save=$UUID_FILE --insecure-options=image \
	  --net=host \
	  docker://quay.io/coreos/etcd:v3.2.9

  wait_for_file $UUID_FILE

  UUID=$( cat $UUID_FILE )
  wait_for_container $UUID
  echo "To run tests export the following environment variables:"
  echo ""
  echo "  export DEX_ETCD_ENDPOINTS=http://localhost:2379"
  echo ""
}

function destroy_etcd {
  UUID_FILE=/tmp/dex-etcd-uuid
  systemctl stop dex-etcd
  rkt rm --uuid-file=$UUID_FILE
  rm $UUID_FILE
}


main $@

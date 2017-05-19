#!/bin/bash

set -ex

if [ "$1" = "start" ]
then
  cd ..
  make
  dockerize -wait tcp://db:5432
  ./bin/dex serve compose/config.yml
fi

exec "$@"

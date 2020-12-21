#!/bin/sh -e

### Usage: /entrypoint.sh <command> <args>
command=$1

if [ "$command" == "serve" ]; then
  file="$2"
  dockerize -template "$file" | dex serve -
else
  dex $@
fi

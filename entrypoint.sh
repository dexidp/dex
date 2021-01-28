#!/bin/sh -e

### Usage: /entrypoint.sh <command> <args>
set -e
command=$1

if [ "$command" == "serve" ]; then
  file="$2"
  gomplate -f "$file" -o "/etc/dex/config.yaml";
  exec dex serve "/etc/dex/config.yaml"
else
  exec dex $@
fi

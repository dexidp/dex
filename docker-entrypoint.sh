#!/bin/sh -e

### Usage: /docker-entrypoint.sh <command> <args>
### * If command equals to "serve", config file for serving will be preprocessed using gomplate and saved to tmp dir.
###   Example: docker-entrypoint.sh serve config.yaml = dex serve /tmp/dex-config.yaml-ABCDEFG
### * If command is not in the list of known dex commands, it will be executed bypassing entrypoint.
###   Example: docker-entrypoint.sh echo "Hello!" = echo "Hello!"

command=$1

case "$command" in
  serve)
    for file_candidate in $@ ; do
      if test -f "$file_candidate"; then
        tmpfile=$(mktemp /tmp/dex.config.yaml-XXXXXX)
        gomplate -f "$file_candidate" -o "$tmpfile"

        args="${args} ${tmpfile}"
      else
        args="${args} ${file_candidate}"
      fi
    done
    exec dex $args
    ;;
  --help|-h|version)
    exec dex $@
    ;;
  *)
    exec $@
    ;;
esac

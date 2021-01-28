#!/bin/sh -e

### Usage: /docker-entrypoint.sh <command> <args>
command=$1

case "$command" in
  serve)
    for file_candidate in $@ ; do
      if test -f "$file_candidate"; then
        tmpfile=$(mktemp /tmp/dex.config.yaml-XXXXXX)
        gomplate -f "$file_candidate" -o "$tmpfile"
        echo "config rendered successfully into the tmp file ${tmpfile}"

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

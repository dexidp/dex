#!/bin/sh -e

### Usage: /docker-entrypoint.sh <command> <args>
function main() {
  executable=$1
  command=$2

  if [[ "$executable" != "dex" ]] && [[ "$executable" != "$(which dex)" ]]; then
    exec $@
  fi

  if [[ "$command" != "serve" ]]; then
    exec $@
  fi

  for tpl_candidate in $@ ; do
    case "$tpl_candidate" in
      *.tpl|*.tmpl|*.yaml)
        tmp_file=$(mktemp /tmp/dex.config.yaml-XXXXXX)
        gomplate -f "$tpl_candidate" -o "$tmp_file"

        args="${args} ${tmp_file}"
        ;;
      *)
        args="${args} ${tpl_candidate}"
        ;;
    esac
  done
  exec $args
}

main $@

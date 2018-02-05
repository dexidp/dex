#!/bin/bash -e

DIFF=$( git diff . )
if [ "$DIFF" != "" ]; then
    echo "$DIFF" >&2
    exit 1
fi

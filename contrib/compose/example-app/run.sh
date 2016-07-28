#!/bin/bash -e
sleep 20

# load the variables from the command
eval "$(/opt/dex/bin/dexctl --db-url='postgres://postgres:postgres@pg:5432/dex?sslmode=disable' new-client http://front:5555/callback)"

# runs the example app
/opt/dex/bin/example-app --client-id=$DEX_APP_CLIENT_ID --client-secret=$DEX_APP_CLIENT_SECRET --discovery=http://dex-worker:5556 --redirect-url="http://front:5555/callback"

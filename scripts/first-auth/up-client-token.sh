#!/bin/bash

# define usefull variable
if [ -z "$DEX_BIN_PATH" ]
then
    DEX_BIN_PATH="../../bin"
fi

if [ -z "$DEX_CONFIG_PATH" ]
then
    DEX_CONFIG_PATH="../.."
fi

# certs files
CA_CRT="$DEX_CONFIG_PATH/ca.crt"
CA_KEY="$DEX_CONFIG_PATH/ca.key"
CLIENT_CRT="$DEX_CONFIG_PATH/client.crt"
CLIENT_KEY="$DEX_CONFIG_PATH/client.key"
SERVER_KEY="$DEX_CONFIG_PATH/server.key"

# cmd variables
CMD="first-auth upClientToken"
read -p "tokenID: " tokenID
read -p 'clientID: ' clientID
read -p 'expire in (days): ' days

ARGS="--tokenID=$tokenID --clientID=$clientID --days=$days"
EXEC="$CMD --ca-crt=$CA_CRT --client-crt=$CLIENT_CRT --client-key=$CLIENT_KEY $ARGS"
$DEX_BIN_PATH/$EXEC

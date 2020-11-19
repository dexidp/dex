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
CMD="first-auth upAclToken"
read -p 'tokenID: ' tokenID
read -p 'description: ' desc
read -p 'max user: ' maxUser
read -p 'clientTokens: ' clientTokens

Description=""
for elem in ${desc[@]};
do
if [ -z $Description ]
then
        Description="${elem}"
else 
        Description="$Description,${elem}"
fi
done
ARGS="--tokenID=$tokenID --desc=$Description --maxUser=$maxUser --clientTokens=$clientTokens"
EXEC="$CMD --ca-crt=$CA_CRT --client-crt=$CLIENT_CRT --client-key=$CLIENT_KEY $ARGS"
$DEX_BIN_PATH/$EXEC

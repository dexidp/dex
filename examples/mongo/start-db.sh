#!/bin/bash

docker-compose up -d

sleep 5

docker exec mongo1 /scripts/rs-init.sh
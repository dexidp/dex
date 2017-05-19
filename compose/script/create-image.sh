#!/bin/bash

set -ex
CONTAINER_NAME=circleci/dex-service
SHORT_SHA=$(git rev-parse --short HEAD)
DOCKER_TAG="0.1.${CIRCLE_BUILD_NUM}-${SHORT_SHA}"
DOCKER_PATH="${CONTAINER_NAME}:${DOCKER_TAG}"
AWS_URL="183081753049.dkr.ecr.us-east-1.amazonaws.com/${DOCKER_PATH}"

docker build --tag ${DOCKER_PATH} .
docker tag ${DOCKER_PATH} ${AWS_URL}
docker push ${AWS_URL}

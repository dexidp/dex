#!/bin/bash

set -e

dockerize -wait tcp://db:5432
./bin/dex serve config.yml

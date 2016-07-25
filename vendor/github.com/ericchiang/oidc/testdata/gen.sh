#!/bin/bash

set -e 

for i in $(seq 1 4); do
   openssl ecparam -out ecdsa_521_${i}.pem -name secp521r1 -genkey -noout
done

for i in $(seq 1 4); do
   openssl genrsa -out rsa_2048_${i}.pem 2048
done

for i in $(seq 1 4); do
   openssl genrsa -out rsa_4096_${i}.pem 4096
done

#!/bin/bash -e

# Stolen from the coreos/matchbox repo.

echo "
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name

[req_distinguished_name]

[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.101 = localhost
" > openssl.config

openssl genrsa -out testdata/ca.key 2048
openssl genrsa -out testdata/server.key 2048

openssl req \
    -x509 -new -nodes \
    -key testdata/ca.key \
    -days 10000 -out testdata/ca.crt \
    -subj "/CN=ldap-tests"

openssl req \
	-new \
	-key testdata/server.key \
	-out testdata/server.csr \
	-subj "/CN=localhost" \
	-config openssl.config

openssl x509 -req \
	-in testdata/server.csr \
	-CA testdata/ca.crt \
	-CAkey testdata/ca.key \
	-CAcreateserial \
	-out testdata/server.crt \
	-days 10000 \
	-extensions v3_req \
	-extfile openssl.config

rm testdata/server.csr
rm testdata/ca.srl
rm openssl.config

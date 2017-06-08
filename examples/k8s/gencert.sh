#!/bin/bash

mkdir -p ssl

cat << EOF > ssl/req.cnf
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name

[req_distinguished_name]

[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = dex.example.com
EOF


# generate ca-key (random) (the private key)
# openssl genrsa -out ssl/ca-key.pem 2048

# uses ca-key.pem to generate ca.pem (the public key)
openssl req -x509 -new -nodes -key $1 -days 10 -out ssl/ca.pem -subj "/CN=$3"


# generate key.pem (random/symetrical key to establish SSL)
openssl genrsa -out ssl/key.pem 2048

# use key.pem AND req.cnf to generate csr.pem (result PART-A)
openssl req -new -key ssl/key.pem -out ssl/csr.pem -subj "/CN=$3" -config ssl/req.cnf

# uses csr.pem and ca.pem to generate cert.pem (result PART-B) (SSL cert including public key)
openssl x509 -req -in ssl/csr.pem -CA ssl/ca.pem -CAkey $1 -CAcreateserial -out ssl/cert.pem -days 10 -extensions v3_req -extfile ssl/req.cnf

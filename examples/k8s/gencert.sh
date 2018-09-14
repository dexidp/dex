#!/bin/bash

CLUSTER_PRIVATE_KEY=$1
CLUSTER_CA=$2
COMMON_NAME=$3

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
DNS.1 = $COMMON_NAME
EOF

# generate ca-key (random) (the private key "Root CA")
# openssl genrsa -out ssl/ca-key.pem 2048

# uses ca-key.pem to generate ca.pem (the public key, that pair with the Private Key "Root CA")
# openssl req -x509 -new -nodes -key ROOT_CA -days 10 -out ssl/ca.pem -subj "/CN=$COMMON_NAME"


# generate key.pem (random/symetrical key to establish SSL)
openssl genrsa -out ssl/key.pem 2048

# use key.pem AND req.cnf to generate csr.pem (result PART-A, X509 cert signing request)
openssl req -new -key ssl/key.pem -out ssl/csr.pem -subj "/CN=$COMMON_NAME" -config ssl/req.cnf

# uses csr.pem and ca.pem to generate cert.pem (result PART-B) (SSL cert including public key)
openssl x509 -req -in ssl/csr.pem -CA $CLUSTER_CA -CAkey $CLUSTER_PRIVATE_KEY -CAcreateserial -out ssl/cert.pem -days 1000 -extensions v3_req -extfile ssl/req.cnf

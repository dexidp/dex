#!/bin/bash -ex

# Always run from the testdata directory
cd "$(dirname "$0")"

# Uncomment these commands to regenerate the CA files.
#
# openssl req \
#     -nodes \
#     -newkey rsa:2048 \
#     -keyout ca.key \
#     -new -x509 -days 7300 \
#     -extensions v3_ca \
#     -out ca.crt \
#     -subj "/O=DEX/CN=coreos.com"
# 
# openssl req \
#     -nodes \
#     -newkey rsa:2048 \
#     -keyout bad-ca.key \
#     -new -x509 -days 7300 \
#     -extensions v3_ca \
#     -out bad-ca.crt \
#     -subj "/O=BAD/CN=coreos.com"

# Sign these files using xmlsec1.
#
# Templates MUST have a <Signature> element already embedded in them so
# xmlsec1 can know where to embed the signature.
#
# See: https://sgros.blogspot.com/2013/01/signing-xml-document-using-xmlsec1.html

xmlsec1 --sign --privkey-pem ca.key,ca.crt --output good-resp.xml good-resp.tmpl
xmlsec1 --sign --privkey-pem ca.key,ca.crt --output bad-status.xml bad-status.tmpl

# Sign a specific sub element, not just the root.
#
# Values match up to the <Response URI="#(ID)"> element in the documents.
xmlsec1 --sign --privkey-pem ca.key,ca.crt  \
    --id-attr:ID Assertion \
    --output assertion-signed.xml assertion-signed.tmpl

xmlsec1 --sign --privkey-pem ca.key,ca.crt \
    --id-attr:ID Assertion \
    --output two-assertions-first-signed.xml \
    two-assertions-first-signed.tmpl


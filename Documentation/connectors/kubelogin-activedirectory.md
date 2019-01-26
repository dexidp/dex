# Integration kubelogin and Active Directory

## Overview

kubelogin is helper tool for kubernetes and oidc integration.
It makes easy to login Open ID Provider.
This document describes how dex work with kubelogin and Active Directory.

examples/config-ad-kubelogin.yaml is sample configuration to integrate Active Directory and kubelogin.

## Precondition

1. Active Directory
You should have Active Directory or LDAP has Active Directory compatible schema such as samba ad.
You may have user objects and group objects in AD. Please ensure TLS is enabled.

2. Install kubelogin
Download kubelogin from https://github.com/int128/kubelogin/releases.
Install it to your terminal.

## Getting started

### Generate certificate and private key

Create OpenSSL conf req.conf as follow:

```
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
```

Please replace dex.example.com to your favorite hostname.
Generate certificate and private key by following command.

```console
$ openssl req -new -x509 -sha256 -days 3650 -newkey rsa:4096 -extensions v3_req -out openid-ca.pem -keyout openid-key.pem -config req.cnf -subj "/CN=kube-ca" -nodes
$ ls openid*
openid-ca.pem openid-key.pem
```

### Modify dex config

Modify following host, bindDN and bindPW in examples/config-ad-kubelogin.yaml.

```yaml
connectors:
- type: ldap
  name: OpenLDAP
  id: ldap
  config:
    host: ldap.example.com:636

    # No TLS for this setup.
    insecureNoSSL: false
    insecureSkipVerify: true

    # This would normally be a read-only user.
    bindDN: cn=Administrator,cn=users,dc=example,dc=com
    bindPW: admin0!
```

### Run dex

```
$ bin/dex serve examples/config-ad-kubelogin.yaml
```

### Configure kubernetes with oidc

Copy openid-ca.pem to /etc/ssl/certs/openid-ca.pem on master node.

Use the following flags to point your API server(s) at dex. `dex.example.com` should be replaced by whatever DNS name or IP address dex is running under.

```
--oidc-issuer-url=https://dex.example.com:32000/dex
--oidc-client-id=kubernetes
--oidc-ca-file=/etc/ssl/certs/openid-ca.pem
--oidc-username-claim=email
--oidc-groups-claim=groups
```

Then restart API server(s).


See https://kubernetes.io/docs/reference/access-authn-authz/authentication/ for more detail.

### kubelogin

Create context for dex authentication:

```console
$ kubectl config set-context oidc-ctx --cluster=cluster.local --user=test
$ kubectl config set-credentials test \
  --auth-provider=oidc \
  --auth-provider-arg=idp-issuer-url=https://dex.example.com:32000/dex \
  --auth-provider-arg=client-id=kubernetes \
  --auth-provider-arg=client-secret=ZXhhbXBsZS1hcHAtc2VjcmV0 \
  --auth-provider-arg=idp-certificate-authority-data=$(base64 -w 0 openid-ca.pem) \
  --auth-provider-arg=extra-scopes="offline_access openid profile email group"
$ kubectl config use-context oidc-ctx
```

Please confirm idp-issuer-url, client-id, client-secret and idp-certificate-authority-data value is same as config-ad-kubelogin.yaml's value.

Then run kubelogin:

```console
$ kubelogin
```

Access http://localhost:8000 by web browser and login with your AD account (eg. test@example.com) and password.
After login and grant, you have following token in ~/.kube/config:

```
        id-token: eyJhbGciOiJSUzICuU4dCcilDDWlw2lfr8mg...
        refresh-token: ChlxY2EzeGhKEB4492EzecdKJOElECK...
```


Similar to etcd's [tls-setup](https://github.com/coreos/etcd/tree/master/hack/tls-setup), this demonstrates using Cloudflare's [cfssl](https://github.com/cloudflare/cfssl) to easily generate certificates for an dex server.

Defaults generate an ECDSA-384 root and leaf certificates for `localhost`.

**Instructions**

1. Install git, go, and make
2. Run `make` to generate the certs

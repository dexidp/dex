# Kubernetes authentication through dex

## Overview

This document covers setting up the [Kubernetes OpenID Connect token authenticator plugin][k8s-oidc] with dex.

Token responses from OpenID Connect providers include a signed JWT called an ID Token. ID Tokens contain names, emails, unique identifiers, and in dex's case, a set of groups that can be used to identify the user. OpenID Connect providers, like dex, publish public keys; the Kubernetes API server understands how to use these to verify ID Tokens.

The authentication flow looks like:

1. OAuth2 client logs a user in through dex.
2. That client uses the returned ID Token as a bearer token when talking to the Kubernetes API.
3. Kubernetes uses dex's public keys to verify the ID Token.
4. A claim designated as the username (and optionally group information) will be associated with that request.

Username and group information can be combined with Kubernetes [authorization plugins][k8s-authz], such as roles based access control (RBAC), to enforce policy.

## Configuring the OpenID Connect plugin

Configuring the API server to use the OpenID Connect [authentication plugin][k8s-oidc] requires:

* Deploying an API server with specific flags.
* Dex is running on HTTPS.
  * Custom CA files must be accessible by the API server (likely through volume mounts).
* Dex is accessible to both your browser and the Kubernetes API server.

Use the following flags to point your API server(s) at dex. `dex.example.com` should be replaced by whatever DNS name or IP address dex is running under.

```
--oidc-issuer-url=https://dex.example.com:32000
--oidc-client-id=example-app
--oidc-ca-file=/etc/kubernetes/ssl/openid-ca.pem
--oidc-username-claim=email
--oidc-groups-claim=groups
```

Additional notes:

* The API server configured with OpenID Connect flags doesn't require dex to be available upfront.
  * Other authenticators, such as client certs, can still be used.
  * Dex doesn't need to be running when you start your API server.
* Kubernetes only trusts ID Tokens issued to a single client.
  * As a work around dex allows clients to [trust other clients][trusted-peers] to mint tokens on their behalf.
* If a claim other than "email" is used for username, for example "sub", it will be prefixed by `"(value of --oidc-issuer-url)#"`. This is to namespace user controlled claims which may be used for privilege escalation.

## Deploying dex on Kubernetes

The dex repo contains scripts for running dex on a Kubernetes cluster with authentication through GitHub. The dex service is exposed using a [node port][node-port] on port 32000. This likely requires a custom `/etc/hosts` entry pointed at one of the cluster's workers.

Because dex uses `ThirdPartyResources` to store state, no external database is needed. For more details see the [storage documentation](storage.md#kubernetes-third-party-resources).

There are many different ways to spin up a Kubernetes development cluster, each with different host requirements and support for API server reconfiguration. At this time, this guide does not have copy-pastable examples, but can recommend the following methods for spinning up a cluster:

* [coreos-kubernetes][coreos-kubernetes] repo for vagrant and VirtualBox users.
* [coreos-baremetal][coreos-baremetal] repo for Linux QEMU/KVM users.

To run dex on Kubernetes perform the following steps:

1. Generate TLS assets for dex.
2. Spin up a Kubernetes cluster with the appropriate flags and CA volume mount.
3. Create a secret containing your [GitHub OAuth2 client credentials][github-oauth2].
4. Deploy dex.

The TLS assets can be created using the following command:

```
$ cd examles/k8s
$ ./gencert.sh
```

The created `ssl/ca.pem` must then be mounted into your API server deployment. Once the cluster is up and correctly configured, use kubectl to add the serving certs as secrets.

```
$ kubectl create secret tls dex.example.com.tls --cert=ssl/cert.pem --key=ssl/key.pem
```

Then create a secret for the GitHub OAuth2 client.

```
$ kubectl create secret \
    generic github-client \
    --from-literal=client-id=$GITHUB_CLIENT_ID \
    --from-literal=client-secret=$GITHUB_CLIENT_SECRET
```

Finally, create the dex deployment, configmap, and node port service.

```
$ kubectl create -f dex.yaml
```

__Caveats:__ No health checking is configured because dex does its own TLS termination complicating the setup. This is a known issue and can be tracked [here][dex-healthz].

## Logging into the cluster

The `example-app` can be used to log into the cluster and get an ID Token. To build the app, you can run `make` in the root of the repo and it will build the `example-app` binary in the repo's `bin` directory. To build the `example-app` requires at least a 1.7 version of Go. 

```
$ ./bin/example-app --issuer https://dex.example.com:32000 --issuer-root-ca examples/k8s/ssl/ca.pem
```

Please note that the `example-app` will listen at http://127.0.0.1:5555 and can be changed with the `--listen` flag.

Once the example app is running, choose the GitHub option and grant access to dex to view your profile.

The default redirect uri is http://127.0.0.1:5555/callback and can be changed with the `--redirect-uri` flag and should correspond with your configmap.

The printed ID Token can then be used as a bearer token to authenticate against the API server.

```
$ token='(id token)'
$ curl -H "Authorization: Bearer $token" -k https://( API server host ):443/api/v1/nodes
```

[k8s-authz]: http://kubernetes.io/docs/admin/authorization/
[k8s-oidc]: http://kubernetes.io/docs/admin/authentication/#openid-connect-tokens
[trusted-peers]: https://godoc.org/github.com/coreos/dex/storage#Client
[coreos-kubernetes]: https://github.com/coreos/coreos-kubernetes/
[coreos-baremetal]: https://github.com/coreos/coreos-baremetal/
[dex-healthz]: https://github.com/coreos/dex/issues/682
[github-oauth2]: https://github.com/settings/applications/new
[node-port]: http://kubernetes.io/docs/user-guide/services/#type-nodeport 
[coreos-kubernetes]: https://github.com/coreos/coreos-kubernetes
[coreos-baremetal]: https://github.com/coreos/coreos-baremetal

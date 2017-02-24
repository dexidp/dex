# Storage options

Dex requires persisting state to perform various tasks such as track refresh tokens, preventing replays, and rotating keys. This document is a summary of the storage configurations supported by dex.

Storage breaches are serious as they can affect applications that rely on dex. Dex saves sensitive data in its backing storage, including signing keys and bcrypt'd passwords. As such, transport security and database ACLs should both be used, no matter which storage option is chosen.

## Kubernetes third party resources

__NOTE:__ Dex requires Kubernetes version 1.4+.

Kubernetes third party resources are a way for applications to create new resources types in the Kubernetes API. This allows dex to run on top of an existing Kubernetes cluster without the need for an external database. While this storage may not be appropriate for a large number of users, it's extremely effective for many Kubernetes use cases.

The rest of this section will explore internal details of how dex uses `ThirdPartyResources`. __Admins should not interact with these resources directly__, except when debugging. These resources are only designed to store state and aren't meant to be consumed by humans. For modifying dex's state dynamically see the [API documentation](api.md).

The `ThirdPartyResource` type acts as a description for the new resource a user wishes to create. The following an example of a resource managed by dex:

```
kind: ThirdPartyResource
apiVersion: extensions/v1beta1
metadata:
  name: o-auth2-client.oidc.coreos.com
versions:
  - name: v1
description: "An OAuth2 client."
```

Once the `ThirdPartyResource` is created, custom resources can be created at a namespace level (though there will be a gap between the `ThirdPartyResource` being created and the API server accepting the custom resource). While most fields are user defined, the API server still respects the common `ObjectMeta` and `TypeMeta` values. For example names are still restricted to a small set of characters, and the `resourceVersion` field can be used for an [atomic compare and swap][k8s-api].

The following is an example of a custom `OAuth2Client` resource:

```
# Standard Kubernetes resource fields
kind: OAuth2Client
apiVersion: oidc.coreos.com/v1
metadata:
  namespace: foobar
  name: ( opaque hash )

# Custom fields defined by dex.
clientID: "aclientid"
clientSecret: "clientsecret"
redirectURIs:
- "https://app.example.com/callback"
```

The `ThirdPartyResource` type and the custom resources can be queried, deleted, and edited like any other resource using `kubectl`.

```
kubectl get thirdpartyresources # list third party resources registered on the clusters
kubectl get --namespace=foobar oauth2clients # list oauth2 clients in a given namespace
```

To reduce administrative overhead, dex creates and manages its own third party resources and may create new ones during upgrades. While not strictly required we feel this is important for reasonable updates. Though, as a result, dex requires access to the non-namespaced `ThirdPartyResource` type. For example, clusters using RBAC authorization would need to create the following roles and bindings:

```
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1alpha1
metadata:
  name: dex
rules:
  - apiGroups: ["oidc.coreos.com"] # API group created by dex
    resources: ["*"]
    verbs: ["*"]
    nonResourceURLs: []
  - apiGroups: ["extensions"]
    resources: ["thirdpartyresources"]
    verbs: ["create"] # To manage its own resources identity must be able to create thirdpartyresources.
    nonResourceURLs: []
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1alpha1
metadata:
  name: dex
subjects:
  - kind: ServiceAccount
    name: dex                 # Service account assigned to the dex pod.
    namespace: demo-namespace # The namespace dex is running in.
roleRef:
  kind: ClusterRole
  name: identity
  apiVersion: rbac.authorization.k8s.io/v1alpha1
```

The storage configuration is extremely limited since installations running outside a Kubernetes cluster would likely prefer a different storage option. An example configuration for dex running inside Kubernetes:

```
storage:
  type: kubernetes
  config:
    inCluster: true
```

Dex determines the namespace it's running in by parsing the service account token automatically mounted into its pod.

## SQL

Dex supports two flavors of SQL, SQLite3 and Postgres. MySQL and CockroachDB may be added at a later time.

Migrations are performed automatically on the first connection to the SQL server (it does not support rolling back). Because of this dex requires privileges to add and alter the tables for its database.

__NOTE:__ Previous versions of dex required symmetric keys to encrypt certain values before sending them to the database. This feature has not yet been ported to dex v2. If it is added later there may not be a migration path for current v2 users.

### SQLite3

SQLite3 is the recommended storage for users who want to stand up dex quickly. It is __not__ appropriate for real workloads.

The SQLite3 configuration takes a single argument, the database file.

```
storage:
  type: sqlite3
  config:
    file: /var/dex/dex.db
```

Because SQLite3 uses file locks to prevent race conditions, if the ":memory:" value is provided dex will automatically disable support for concurrent database queries.

### Postgres

When using Postgres, admins may want to dedicate a database to dex for the following reasons:

1. Dex requires privileged access to its database because it performs migrations.
2. Dex's database table names are not configurable; when shared with other applications there may be table name clashes.

```
CREATE DATABASE dex_db;
CREATE USER dex WITH PASSWORD '66964843358242dbaaa7778d8477c288';
GRANT ALL PRIVILEGES ON DATABASE dex_db TO dex;
```

An example config for Postgres setup using these values:

```
storage:
  type: postgres
  config:
    database: dex_db
    user: dex
    password: 66964843358242dbaaa7778d8477c288
    ssl:
      mode: verify-ca
      caFile: /etc/dex/postgres.ca
```

The SSL "mode" corresponds to the `github.com/lib/pq` package [connection options][psql-conn-options]. If unspecified, dex defaults to the strictest mode "verify-full".

## Adding a new storage options

Each storage implementation bears a large ongoing maintenance cost and needs to be updated every time a feature requires storing a new type. Bugs often require in depth knowledge of the backing software, and much of this work will be done by developers who are not the original author. Changes to dex which add new storage implementations are not merged lightly.

### New storage option references

Those who still want to construct a proposal for a new storage should review the following packages:

* `github.com/coreos/dex/storage`: Interface definitions which the storage must implement. __NOTE:__ This package is not stable.
* `github.com/coreos/dex/storage/conformance`: Conformance tests which storage implementations must pass.

### New storage option requirements

Any proposal to add a new implementation must address the following:

* Integration testing setups (Travis and developer workstations).
* Transactional requirements: atomic deletes, updates, etc.
* Is there an established and reasonable Go client?

[issues-transaction-tests]: https://github.com/coreos/dex/issues/600
[k8s-api]: https://github.com/kubernetes/kubernetes/blob/master/docs/devel/api-conventions.md#concurrency-control-and-consistency
[psql-conn-options]: https://godoc.org/github.com/lib/pq#hdr-Connection_String_Parameters

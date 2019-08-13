# Storage options

Dex requires persisting state to perform various tasks such as track refresh tokens, preventing replays, and rotating keys. This document is a summary of the storage configurations supported by dex.

Storage breaches are serious as they can affect applications that rely on dex. Dex saves sensitive data in its backing storage, including signing keys and bcrypt'd passwords. As such, transport security and database ACLs should both be used, no matter which storage option is chosen.

## Etcd

Dex supports persisting state to [etcd v3](https://github.com/coreos/etcd).

An example etcd configuration is using these values:

```
storage:
  type: etcd
  config:
    # list of etcd endpoints we should connect to
    endpoints:
      - http://localhost:2379
    namespace: my-etcd-namespace/
```

Etcd storage can be customized further using the following options:

* `endpoints`: list of etcd endpoints we should connect to
* `namespace`: etcd namespace to be set for the connection. All keys created by
  etcd storage will be prefixed with the namespace. This is useful when you
  share your etcd cluster amongst several applications. Another approach for
  setting namespace is to use [etcd proxy](https://coreos.com/etcd/docs/latest/op-guide/grpc_proxy.html#namespacing)
* `username`: username for etcd authentication
* `password`: password for etcd authentication
* `ssl`: ssl setup for etcd connection
  * `serverName`: ensures that the certificate matches the given hostname the
    client is connecting to.
  * `caFile`: path to the ca
  * `keyFile`: path to the private key
  * `certFile`: path to the certificate

## Kubernetes custom resource definitions (CRDs)

Kubernetes [custom resource definitions](crd) are a way for applications to create new resources types in the Kubernetes API.

The Custom Resource Definition (CRD) API object was introduced in Kubernetes version 1.7 to replace the Third Party Resource (TPR) extension. CRDs allow dex to run on top of an existing Kubernetes cluster without the need for an external database. While this storage may not be appropriate for a large number of users, it's extremely effective for many Kubernetes use cases.

The rest of this section will explore internal details of how dex uses CRDs. __Admins should not interact with these resources directly__, except while debugging. These resources are only designed to store state and aren't meant to be consumed by end users. For modifying dex's state dynamically see the [API documentation](api.md).

The following is an example of the AuthCode resource managed by dex:

```
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: 2017-09-13T19:56:28Z
  name: authcodes.dex.coreos.com
  resourceVersion: "288893"
  selfLink: /apis/apiextensions.k8s.io/v1beta1/customresourcedefinitions/authcodes.dex.coreos.com
  uid: a1cb72dc-98bd-11e7-8f6a-02d13336a01e
spec:
  group: dex.coreos.com
  names:
    kind: AuthCode
    listKind: AuthCodeList
    plural: authcodes
    singular: authcode
  scope: Namespaced
  version: v1
status:
  acceptedNames:
    kind: AuthCode
    listKind: AuthCodeList
    plural: authcodes
    singular: authcode
  conditions:
  - lastTransitionTime: null
    message: no conflicts found
    reason: NoConflicts
    status: "True"
    type: NamesAccepted
  - lastTransitionTime: 2017-09-13T19:56:28Z
    message: the initial names have been accepted
    reason: InitialNamesAccepted
    status: "True"
    type: Established
```

Once the `CustomResourceDefinition` is created, custom resources can be created and stored at a namespace level. The CRD type and the custom resources can be queried, deleted, and edited like any other resource using `kubectl`.

dex requires access to the non-namespaced `CustomResourceDefinition` type. For example, clusters using RBAC authorization would need to create the following roles and bindings:
```
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: dex
rules:
- apiGroups: ["dex.coreos.com"] # API group created by dex
  resources: ["*"]
  verbs: ["*"]
- apiGroups: ["apiextensions.k8s.io"]
  resources: ["customresourcedefinitions"]
  verbs: ["create"] # To manage its own resources identity must be able to create customresourcedefinitions.
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: dex
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: dex
subjects:
- kind: ServiceAccount
  name: dex                 # Service account assigned to the dex pod.
  namespace: dex-namespace  # The namespace dex is running in.

```


### Removed: Kubernetes third party resources(TPRs)

TPR support in dex has been removed.  The last version to support TPR
is [v2.17.0](https://github.com/dexidp/dex/tree/v2.17.0)

If you are currently running dex using TPRs, you will need to [migrate to CRDs](https://github.com/dexidp/dex/blob/v2.17.0/Documentation/storage.md#migrating-from-tprs-to-crds)
before you upgrade to a post v2.17 dex.  The script mentioned in the instructions can be [found here](https://github.com/dexidp/dex/blob/v2.17.0/scripts/dump-tprs)


### Configuration

The storage configuration is extremely limited since installations running outside a Kubernetes cluster would likely prefer a different storage option. An example configuration for dex running inside Kubernetes:

```
storage:
  type: kubernetes
  config:
    inCluster: true
```

Dex determines the namespace it's running in by parsing the service account token automatically mounted into its pod.

## SQL

Dex supports two flavors of SQL: SQLite3 and Postgres.

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

### MySQL

Dex requires MySQL 5.7 or later version. When using MySQL, admins may want to dedicate a database to dex for the following reasons:

1. Dex requires privileged access to its database because it performs migrations.
2. Dex's database table names are not configurable; when shared with other applications there may be table name clashes.

```
CREATE DATABASE dex_db;
CREATE USER dex IDENTIFIED BY '66964843358242dbaaa7778d8477c288';
GRANT ALL PRIVILEGES ON dex_db.* TO dex;
```

An example config for MySQL setup using these values:

```
storage:
  type: mysql
  config:
    database: dex_db
    user: dex
    password: 66964843358242dbaaa7778d8477c288
    ssl:
      mode: custom
      caFile: /etc/dex/mysql.ca
```

The SSL "mode" corresponds to the `github.com/go-sql-driver/mysql` package [connection options][mysql-conn-options]. If unspecified, dex defaults to the strictest mode "true".

## Adding a new storage options

Each storage implementation bears a large ongoing maintenance cost and needs to be updated every time a feature requires storing a new type. Bugs often require in depth knowledge of the backing software, and much of this work will be done by developers who are not the original author. Changes to dex which add new storage implementations are not merged lightly.

### New storage option references

Those who still want to construct a proposal for a new storage should review the following packages:

* `github.com/dexidp/dex/storage`: Interface definitions which the storage must implement. __NOTE:__ This package is not stable.
* `github.com/dexidp/dex/storage/conformance`: Conformance tests which storage implementations must pass.

### New storage option requirements

Any proposal to add a new implementation must address the following:

* Integration testing setups (Travis and developer workstations).
* Transactional requirements: atomic deletes, updates, etc.
* Is there an established and reasonable Go client?

[issues-transaction-tests]: https://github.com/dexidp/dex/issues/600
[k8s-api]: https://github.com/kubernetes/kubernetes/blob/master/docs/devel/api-conventions.md#concurrency-control-and-consistency
[psql-conn-options]: https://godoc.org/github.com/lib/pq#hdr-Connection_String_Parameters
[mysql-conn-options]: https://github.com/go-sql-driver/mysql#tls
[crd]: https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/

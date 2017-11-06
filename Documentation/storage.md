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

__NOTE:__ CRDs are only supported by Kubernetes version 1.7+.

Kubernetes [custom resource definitions](crd) are a way for applications to create new resources types in the Kubernetes API. The Custom Resource Definition (CRD) API object was introduced in Kubernetes version 1.7 to replace the Third Party Resource (TPR) extension. CRDs allows dex to run on top of an existing Kubernetes cluster without the need for an external database. While this storage may not be appropriate for a large number of users, it's extremely effective for many Kubernetes use cases.

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


## Kubernetes third party resources(TPRs)

__NOTE:__ TPRs will be deprecated by Kubernetes version 1.8.

The default behavior of dex from release v2.7.0 onwards is to utitlize CRDs to manage its custom resources. If users would like to use dex with a Kubernetes version lower than 1.7, they will have to force dex to use TPRs instead of CRDs by setting the `UseTPR` flag in the storage configuration as shown below:

```
storage:
  type: kubernetes
  config:
    kubeConfigFile: kubeconfig
    useTPR: true
```

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

## Migrating from TPRs to CRDs

This section descibes how users can migrate storage data in dex when upgrading from an older version of kubernetes (lower than 1.7). This involves creating new CRDs and moving over the data from TPRs.
The flow of the migration process is as follows:
1. Stop running old version of Dex (lower than v2.7.0).
2. Create new CRDs by running the following command:
   ```
   kubectl apply -f scripts/manifests/crds/
   ```
   Note that the newly created CRDs have `dex.coreos.com` as their group and will not conflict with the existing TPR resources which have `oidc.coreos.com` as the group.
3. Migrate data from existing TPRs to CRDs by running the following commands for each of the TPRs:
   1. Export `DEX_NAMESPACE` to be the namespace in which the TPRs exist and run the following script to store TPR definition in a temporary yaml file:
      ```
      export DEX_NAMESPACE="<namespace-value>"
      ./scripts/dump-tprs > out.yaml
      ```
   2. Update `out.yaml` to change the apiVersion to `apiVersion: dex.coreos.com/v1` and delete the `resourceVersion` field.
      ```
      sed 's/oidc.coreos.com/dex.coreos.com/' out.yaml
      ```
      ```
      sed 's/resourceVersion: ".*"//' out.yaml
      ```
   3. Create the resource object using the following command:
      ```
      kubectl apply -f out.yaml
      ```
   4. Confirm that the resource got created using the following get command:
      ```
      kubectl get --namespace=tectonic-system <TPR-name>.dex.coreos.com  -o yaml
      ```
4. Update to new version of Dex (v2.7.0 or higher) which will use CRDs instead of TPRs.

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
[crd]: https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/

# Deploying dex on Kubernetes

This document will allow you to set up dex in your Kubernetes cluster; the example configuration files are generally useful, but will need to be modified to meet the needs of your deployment. The places that are likely to need modification will be called out as often as possible in this document.

## Prerequisites and Assumptions

The document assumes that you already have a cluster with at least one worker up and running. The easiest way to bring up a small cluster for experimentation is the [coreos-kubernetes single node](coreos-kubernetes-single-node) Vagrant installer.

The other assumption is that your Kubernetes cluster will be routable on `172.17.4.99`  (which is what it will be if you use [coreos-kubernetes single node][coreos-kubernetes-single-node], and the issuer URL for your dex installation is `http://172.17.4.99:30556`; in production installations you will need to make sure that you are serving on https and you will likely want to use a hostname rather than an IP address.

[coreos-kubernetes-single-node](https://github.com/coreos/coreos-kubernetes/blob/master/single-node/README.md)

## Start Postgres

Dex needs a database to store information; these commands will create a Postgres service that dex can use. Note that this configuration is not suitable for production - if the container is destroyed, the data is gone forever. 

In production you should have a sufficiently fault-tolerant Postgres deployment on a persistent volume with backup.

```
kubectl create -f postgres-rc.yaml
kubectl create -f postgres-service.yaml
```

## Create your secrets.

dex needs a secret key for encrypting private keys in the database. These can be stored as [Kubernetes secrets][k8s-secrets].

[k8s-secrets]: http://kubernetes.io/v1.0/docs/user-guide/secrets.html

```
kubectl create -f dex-secrets.yaml
```

## Start the Overlord

Start the overlord. This will also initialize your database the first time it's run, and perform migrations when new versions are installed.

```
kubectl create -f dex-overlord-rc.yaml
kubectl create -f dex-overlord-service.yaml
```

Note: this will make the admin API available to any pod in the cluster. This API is very powerful, and allows the creation of admin users who can perform any action in dex, including creating, modifying and deleting other users. This will be fixed soon by requirng some sort of authentication.

## Add a Connector

This is bit of a hack; right now the only way to add connectors and register
your first client is to use the `dexctl` tool talking directly to the
database. Because the database is only routable inside the cluster, we do it
inside a pod via `kubectl exec`. (note that if your DB is not running on the cluster, you can run the dexctl command directly against your database.)

The other hacky thing is that this needs to happen before the workers start because workers do not (yet!) respond dynamically to connector configuration changes.

First, start a shell session on the overlord pod.
```
DEX_OVERLORD_POD=$(kubectl get pod -l=app=dex,role=overlord -o template --template "{{ (index .items 0).metadata.name }}")

kubectl exec -ti $DEX_OVERLORD_POD -- sh
```

Once we're on the pod, we create a connectors file and upload it to dex.

```
DEX_CONNECTORS_FILE=$(mktemp  /tmp/dex-conn.XXXXXX)
cat << EOF > $DEX_CONNECTORS_FILE
[
	{
		"type": "local",
		"id": "local"
	}
]
EOF

/opt/dex/bin/dexctl --db-url=postgres://postgres@dex-postgres.default:5432/postgres?sslmode=disable set-connector-configs $DEX_CONNECTORS_FILE
exit
```

## Start the Worker

Start the worker. The worker is exposed as an external service so that end-users can access it.

```
kubectl create -f dex-worker-rc.yaml
kubectl create -f dex-worker-service.yaml
```

## [Create a client](https://github.com/coreos/dex#registering-clients)

We then `eval` that which creates the shell variables `DEX_APP_CLIENT_ID` and `DEX_APP_CLIENT_SECRET`

```
eval "$(kubectl exec $DEX_OVERLORD_POD -- /opt/dex/bin/dexctl --db-url=postgres://postgres@dex-postgres.default:5432/postgres?sslmode=disable new-client http://127.0.0.1:5555/callback )"
```

## Build and Run the Example App

First, go to the root of the dex repo:

```
cd ../..
```

Now, build and run the example app.

```
./build
./bin/example-app --client-id=$DEX_APP_CLIENT_ID --client-secret=$DEX_APP_CLIENT_SECRET --discovery=http://172.17.4.99:30556
```

Now you can register and log-in to your example app: Go to http://127.0.0.1:5555

## Debugging


### psql

Here's how to get psql session.
```
DEX_PSQL_POD=$(kubectl get pod -l=app=postgres -o template --template "{{ (index .items 0).metadata.name }}")
kubectl exec $DEX_PSQL_POD -ti  -- psql postgres://postgres@dex-postgres.default:5432/postgres?sslmode=disable
```



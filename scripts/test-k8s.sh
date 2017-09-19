#!/bin/bash -e

TEMPDIR=$( mktemp -d )

cat << EOF > $TEMPDIR/kubeconfig
apiVersion: v1
kind: Config
clusters:
- name: local
  cluster:
    server: http://localhost:8080
users:
- name: local
  user:
contexts:
- context:
    cluster: local
    user: local
EOF

cleanup () {
    docker rm -f $( cat $TEMPDIR/etcd )
    docker rm -f $( cat $TEMPDIR/kube-apiserver )
    rm -rf $TEMPDIR
}

trap "{ CODE=$?; cleanup ; exit $CODE; }" EXIT

docker run \
    --cidfile=$TEMPDIR/etcd \
    -d \
    --net=host \
    gcr.io/google_containers/etcd:3.1.10 \
    etcd

docker run \
    --cidfile=$TEMPDIR/kube-apiserver \
    -d \
    -v $TEMPDIR:/var/run/kube-test:ro \
    --net=host \
    gcr.io/google_containers/kube-apiserver-amd64:v1.7.4 \
    kube-apiserver \
    --etcd-servers=http://localhost:2379 \
    --service-cluster-ip-range=10.0.0.1/16 \
    --insecure-bind-address=0.0.0.0 \
    --insecure-port=8080

until $(curl --output /dev/null --silent --head --fail http://localhost:8080/healthz); do
    printf '.'
    sleep 1
done
echo "API server ready"

export DEX_KUBECONFIG=$TEMPDIR/kubeconfig
go test -v -i ./storage/kubernetes
go test -v ./storage/kubernetes

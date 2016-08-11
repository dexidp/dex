# Running dex as the Kubernetes

```
kubectl create -f thirdpartyresources.yaml
kubectl create configmap dex-config --from-file=config.yaml=config-k8s.yaml
kubectl create -f deployment.yaml
```

```
kubectl create -f https://raw.githubusercontent.com/kubernetes/contrib/master/ingress/controllers/nginx/rc.yaml
./gencert.sh
kubectl create secret tls dex.example.com.tls --cert=ssl/cert.pem --key=ssl/key.pem
kubectl create -f dex-ingress.yaml
```

```
kubectl create -f client.yaml
../../bin/example-app --issuer https://dex.example.com --issuer-root-ca ssl/ca.pem
```

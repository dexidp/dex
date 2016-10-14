# Running dex as the Kubernetes authenticator

Running dex as the Kubernetes authenticator requires.

* dex is running on HTTPS.
* Your browser can navigate to dex at the same address Kubernetes refers to it as.

To accomplish this locally, these scripts assume you're using the single host
vagrant setup provided by the [coreos-kubernetes](
https://github.com/coreos/coreos-kubernetes) repo with a couple of changes (a
complete diff is provided at the bottom of this document). Namely that:

* The API server isn't running on host port 443.
* The virtual machine has a populated `/etc/hosts`

The following entry must be added to your host's `/etc/hosts` file as well as
the VM. 

```
172.17.4.99        dex.example.com
```

In the future this document will provide instructions for a more general
Kubernetes installation.

Once you have Kubernetes configured, set up the ThirdPartyResources and a
ConfigMap for dex to use. These run dex as a deployment with configuration and
storage, allowing it to get started. 

```
kubectl create configmap dex-config --from-file=config.yaml=config-k8s.yaml
kubectl create -f deployment.yaml
```

To get dex running at an HTTPS endpoint, create an ingress controller, some
self-signed TLS assets and an ingress rule for dex. These TLS assest should
normally be provided by an actual CA (public or internal).

```
kubectl create -f https://raw.githubusercontent.com/kubernetes/contrib/master/ingress/controllers/nginx/rc.yaml
./gencert.sh
kubectl create secret tls dex.example.com.tls --cert=ssl/cert.pem --key=ssl/key.pem
kubectl create -f dex-ingress.yaml
```

To test that the everything has been installed correctly. Configure a client
with some credentials, and run the `example-app` (run `make` at the top level
of this repo if you haven't already). The second command will error out if your
example-app can't find dex.

```
kubectl create -f client.yaml
../../bin/example-app --issuer https://dex.example.com --issuer-root-ca ssl/ca.pem
```

Navigate to `127.0.0.1:5555` and try to login. You should be redirected to
`dex.example.com` with lots of TLS errors. Proceed around them, authorize the
`example-app`'s OAuth2 client and you should be redirected back to the
`example-app` with valid OpenID Connect credentials.

Finally, to configure Kubernetes to use dex as its authenticator, copy
`ssl/ca.pem` to `/etc/kubernetes/ssl/openid-ca.pem` onto the VM and update the
API server's manifest at `/etc/kubernetes/manifests/kube-apiserver.yaml` to add
the following flags.

```
--oidc-issuer-url=https://dex.example.com
--oidc-client-id=example-app
--oidc-ca-file=/etc/kubernetes/ssl/openid-ca.pem
--oidc-username-claim=email
--oidc-groups-claim=groups
```

Kick the API server by killing its Docker container, and when it comes up again
it should be using dex. Login again through the `example-app` and you should be
able to use the provided token as a bearer token to hit the Kubernetes API.

## Changes to coreos-kubernetes

The following is a diff to the [coreos-kubernetes](https://github.com/coreos/coreos-kubernetes)
repo that accomplishes the required changes.

```diff
diff --git a/single-node/user-data b/single-node/user-data
index f419f09..ed42055 100644
--- a/single-node/user-data
+++ b/single-node/user-data
@@ -80,6 +80,15 @@ function init_flannel {
 }
 
 function init_templates {
+    local TEMPLATE=/etc/hosts
+    if [ ! -f $TEMPLATE ]; then
+        echo "TEMPLATE: $TEMPLATE"
+        mkdir -p $(dirname $TEMPLATE)
+        cat << EOF > $TEMPLATE
+172.17.4.99		dex.example.com
+EOF
+    fi
+
     local TEMPLATE=/etc/systemd/system/kubelet.service
     if [ ! -f $TEMPLATE ]; then
         echo "TEMPLATE: $TEMPLATE"
@@ -195,7 +204,7 @@ spec:
     - --etcd-servers=${ETCD_ENDPOINTS}
     - --allow-privileged=true
     - --service-cluster-ip-range=${SERVICE_IP_RANGE}
-    - --secure-port=443
+    - --secure-port=8443
     - --advertise-address=${ADVERTISE_IP}
     - --admission-control=NamespaceLifecycle,LimitRanger,ServiceAccount,ResourceQuota
     - --tls-cert-file=/etc/kubernetes/ssl/apiserver.pem
@@ -211,8 +220,8 @@ spec:
       initialDelaySeconds: 15
       timeoutSeconds: 15
     ports:
-    - containerPort: 443
-      hostPort: 443
+    - containerPort: 8443
+      hostPort: 8443
       name: https
     - containerPort: 8080
       hostPort: 8080
```

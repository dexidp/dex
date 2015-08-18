# dex

## Getting Started

**Warning**: Hacks Ahead.

You must be running cluster wide DNS for this to work.

Install your dockercfg. There is no nice way to do this:

```
ssh worker
cat > /proc/$(pgrep kubelet)/cwd/.dockercfg
```

Start postgres

```
kubectl create -f postgres-rc.yaml
kubectl create -f postgres-service.yaml
```

Run dex and setup services

```
for i in dex-overlord-rc.yaml dex-overlord-service.yaml dex-worker-rc.yaml dex-worker-service.yaml; do 
	kubectl create -f ${i}
done
```

curl http://$(kubectl describe service dex-worker | grep '^IP:' | awk '{print $2}'):5556

5. [Register your first client](https://github.com/coreos/dex#registering-clients)

## Debugging

You can use a port forward from the target host to debug the database

IP=$(kubectl describe service dex-postgres | grep '^IP:' | awk '{print $2}')
ssh -F ssh-config -L 5432:${IP}:5432 w1
psql -h localhost -w -U postgres

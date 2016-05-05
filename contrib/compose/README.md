# Running dex in Docker compose

This demo intends to demonstrate Dex on a development workstation. It is only intended for local test setups and should not be used for production environments. I recommend to use k8s in production.

This setup is composed of 4 containers:
- dex-worker
- dex-overlord
- postgres
- example app

Notes:
- I needed to use some hacks to get the timing of all components right (https://github.com/docker/compose/issues/374). You will see `sh -c 'sleep x ;` to enforce the right order. Nevertheless it may happen that the startup fails for some components, since the execution time may depend on the host machine.
- production use would need some proper secrets management

## Build and Run the Example App

```
# build the binaries locally (required for the example app)
./go-docker ./build
# copy the example app, so that the Dockerfile is able to reach it
cp -r bin contrib/compose/example-app
cd contrib/compose
# start the containers
docker-compose up
```

To reach the contains from you local machine, you need to update the `/etc/hosts`:

```
127.0.0.1 front
127.0.0.1 dex-worker
```

Now open http://front:5555/ and register a new user.

## Dex URLs

 - http://front:5555 example app
 - http://front:5556/health dex-worker health check
 - http://front:5556/.well-known/openid-configuration
 - http://front:5557/health dex-overlord health check

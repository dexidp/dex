# Dev Guide


## Building

To build using the go binary on your host, use the `./build` script.

You can also use a copy of `go` hosted inside a docker container if you prefix your command with `go-docker`, as in: `./go-docker ./build`

## Docker Build and Push

Once binaries are compiled you can build and push a dex image to quay.io. Before doing this step binaries must be built above using one of the build tools.

```
export DOCKER_USER=<<your user>>
export DOCKER_PASSWORD=<<your password>>
./build-docker-push
```

## Rebuild API from JSON schema

Go API bindings are generated from a JSON Discovery file.
To regenerate run:

```
./schema/generator
```

For updating generator dependencies see docs in: `schema/generator_import.go`.

## Runing Tests

Run all tests: `./test`

Single package only: `PKG=<pkgname> ./test`

Functional tests: `./test-functional`

Run with docker:

```
./go-docker ./test
./go-docker ./test-functional
```


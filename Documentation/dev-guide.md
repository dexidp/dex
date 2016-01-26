# Dev Guide

## No DB mode

When you are working on dex it's convenient to use the `--no-db` flag. This starts up dex in a mode which uses an in-memory datastore for persistence. It also does not rotate keys, so no overlord is required.

In this mode you provide the binary with paths to files for connectors, users, and emailer. There are example files you can use inside of `static/fixtures` named *"connectors.json.sample"*, *"users.json.sample"*, and *"emailer.json.sample"*, respectively.

You can rename these to the equivalent without the *".sample"* suffix since the defaults point to those locations:

```console
mv static/fixtures/connectors.json.sample static/fixtures/connectors.json
mv static/fixtures/users.json.sample static/fixtures/users.json
mv static/fixtures/emailer.json.sample static/fixtures/emailer.json
```

Starting dex is then as simple as:

```console
bin/dex-worker  --no-db
```

***Do not use this flag in production*** - it's not thread safe and data is destroyed when the process dies. In addition, there is no key rotation.

Note: If you want to test out the registration flow, you need to enable that feature by passing `--enable-registration=true` as well.

## Building

To build using the go binary on your host, use the `./build` script.

You can also use a copy of `go` hosted inside a Docker container if you prefix your command with `go-docker`, as in: `./go-docker ./build`

## Docker Build and Push

Once binaries are compiled you can build and push a dex image to quay.io. Before doing this step binaries must be built above using one of the build tools.

```console
export DOCKER_USER=<<your user>>
export DOCKER_PASSWORD=<<your password>>
./build-docker-push
```

By default the script pushes to `quay.io/coreos/dex`; if you want to push to a different repository, override the `DOCKER_REGISTRY` and `DOCKER_REPO` environment variables.

## Rebuild API from JSON schema

Go API bindings are generated from a JSON Discovery file.
To regenerate run:

```console
schema/generator
```

For updating generator dependencies see docs in: `schema/generator_import.go`.

## Running Tests

To run all tests (except functional) use the `./test` script;

If you want to test a single package only, use `PKG=<pkgname> ./test`

The functional tests require a database; create a database (eg. `createdb dex_func_test`) and then pass it as an environment variable to the functional test script, eg.  `DEX_TEST_DSN=postgres://localhost/dex_func_test?sslmode=disable ./test-functional`

To run these tests with Docker is a little trickier; you need to have a container running Postgres, and then you need to link that container to the container running your tests:


```console
# Run the Postgres docker container, which creates a db called "postgres"
docker run --name dex_postgres -d postgres

# The host name in the DSN is "postgres"; that works because that is what we
# will alias the link as, which causes Docker to modify /etc/hosts with a "postgres"
# entry.
export DEX_TEST_DSN=postgres://postgres@postgres/postgres?sslmode=disable

# Run the test container, linking it to the Postgres container.
DOCKER_LINKS=dex_postgres:postgres DOCKER_ENV=DEX_TEST_DSN ./go-docker ./test-functional

# Remove the container after the tests are run.
docker rm -f dex_postgres
```

## Vendoring dependencies

dex uses [godep](https://github.com/tools/godep) for vendoring external dependencies. This section details how to add and update those dependencies.

Before continuing, please ensure you have the **latest version** of godep available in your PATH.

```
go get -u github.com/tools/godep
```

### Preparing your GOPATH

Godep assumes code uses the [standard Go directory layout](https://golang.org/doc/code.html#Organization) with the GOPATH environment variable. Developers who use a different workflow (for instance, prefer working from `~/src/dex`) should see [rkt's documentation](https://github.com/coreos/rkt/blob/master/Documentation/hacking.md#having-the-right-directory-layout-ie-gopath) for workarounds.

Godep determines depdencies using the GOPATH, not the vendored code in the Godeps directory. The first step is to "restore" your GOPATH to match the vendored state of dex. From dex's top level directory run:

```
godep restore -v
```

Next, continue to either *Adding a new package* or *Updating an existing package*.

### Adding a new package

After adding a new `import` to dex source, godep will automatically detect it and update the vendored code. Once code changes are finalized, bring the dependency into your GOPATH and save the state:

```
go get github.com/mavricknz/ldap # Replace with your dependency.
godep save ./...
```

Note that dex does **not** rewrite import paths like other CoreOS projects.

## Updating an existing package

After restoring your GOPATH, update the dependency in your GOPATH to the version you wish to check in.

```
cd $GOPATH/src/github.com/lib/pq # Replace with your dependency.
git checkout origin master
```

Then, move to dex's top level directory and run:

```
godep update github.com/lib/pq
```

To update a group of packages, use the `...` notation.

```
godep update github.com/coreos/go-oidc/...
```

## Finalizing your change

Use git to ensure the Godeps directory has updated only your target packages.

Changes to the Godeps directory should be added as a separate commit from other changes for readability:

```
git status      # make sure things look reasonable
git add Godeps
git commit -m "Godeps: updated postgres driver"

# continue working

git add .
git commit -m "dirname: this is my actual change"
```

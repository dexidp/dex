# Dev Guide

## No DB mode

When you are working on dex it's convenient to use the `--no-db` flag. This starts up dex in a mode which uses an in-memory datastore for persistence. It also does not rotate keys, so no overlord is required.

In this mode you provide the binary with paths to files for clients, connectors, users, and emailer. There are example files you can use inside of `static/fixtures` named *"clients.json.sample"*, *"connectors.json.sample"*, *"users.json.sample"*, and *"emailer.json.sample"*, respectively.

You can rename these to the equivalent without the *".sample"* suffix since the defaults point to those locations:

```console
cp static/fixtures/clients.json.sample static/fixtures/clients.json
cp static/fixtures/connectors.json.sample static/fixtures/connectors.json
cp static/fixtures/users.json.sample static/fixtures/users.json
cp static/fixtures/emailer.json.sample static/fixtures/emailer.json
```

Starting dex is then as simple as:

```console
bin/dex-worker  --no-db
```

***Do not use this flag in production*** - data is destroyed when the process dies and there is no key rotation.

Note: If you want to test out the registration flow, you need to enable that feature by passing `--enable-registration=true` as well.

## Building

To build using the go binary on your host, use the `./build` script. Note that __dex cannot be cross compiled__ due to cgo dependencies.

Docker can be used to build Linux binaries using the `./build-docker` script. Either for Linux user without Go on their host, or for users wishing to cross-compile dex.

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

dex uses [glide](https://github.com/Masterminds/glide) for vendoring external dependencies. This section details how to add and update those dependencies.

Before continuing, please ensure you have the **latest version** of glide available in your PATH.

```
go get -u github.com/Masterminds/glide
```

### Adding a new package

After adding a new `import` to dex source, use `glide get` to add the dependency to the `glide.yaml` and `glide.lock` files.

```
glide get -u -v -s github.com/godbus/dbus
```

Note that __all of these flags are manditory__. This should add an entry to the glide files, add the package to the `vendor` directory, and remove nested `vendor` directories and version control information.

## Updating an existing package

To update an existing package, edit the `glide.yaml` file to the desired verison (most likely a git hash), and run `glide update`.

```
{{ edit the entry in glide.yaml }}
glide update -u -v -s github.com/lib/pq
```

Like `glide get` all flags are manditory. If the update was successful, `glide.lock` will have been updated to reflect the changes to `glide.yaml` and the package will have been updated in `vendor`.

## Finalizing your change

Use git to ensure the `vendor` directory has updated only your target packages, and that no other entries in `glide.yaml` and `glide.lock` have changed.

Changes to the Godeps directory should be added as a separate commit from other changes for readability:

```
git status      # make sure things look reasonable
git add vendor
git commit -m "vendor: updated postgres driver"

# continue working

git add .
git commit -m "dirname: this is my actual change"
```

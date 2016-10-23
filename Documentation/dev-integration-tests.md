# Running integration tests

## Kubernetes

Kubernetes tests will only run if the `DEX_KUBECONFIG` environment variable is set.

```
$ export DEX_KUBECONFIG=~/.kube/config
$ go test -v -i ./storage/kubernetes
$ go test -v ./storage/kubernetes
```

Because third party resources creation isn't synchronized it's expected that the tests fail the first time. Fear not, and just run them again.

## Postgres

Running database tests locally require:

* A systemd based Linux distro.
* A recent version of [rkt](https://github.com/coreos/rkt) installed.

The `standup.sh` script in the SQL directory is used to run databases in containers with systemd daemonizing the process.

```
$ sudo ./storage/sql/standup.sh create postgres
Starting postgres. To view progress run

  journalctl -fu dex-postgres

Running as unit dex-postgres.service.
To run tests export the following environment variables:

  export DEX_POSTGRES_DATABASE=postgres; export DEX_POSTGRES_USER=postgres; export DEX_POSTGRES_PASSWORD=postgres; export DEX_POSTGRES_HOST=172.16.28.3:5432

```

Exporting the variables will cause the database tests to be run, rather than skipped.

```
$ # sqlite3 takes forever to compile, be sure to install test dependencies
$ go test -v -i ./storage/sql
$ go test -v ./storage/sql
```

When you're done, tear down the unit using the `standup.sh` script.

```
$ sudo ./storage/sql/standup.sh destroy postgres
```

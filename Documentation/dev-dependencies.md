# Managing dependencies

## Go modules

Dex uses [Go modules][go-modules] to manage its [`vendor` directory][go-vendor]. Go 1.11 or higher is recommended. While Go 1.12 is expected to finalize the Go modules feature, with Go 1.11 you should [activate the Go modules feature][go-modules-activate] before interacting with Go modules.

Here is one way to activate the Go modules feature with Go 1.11.

```
export GO111MODULE=on  # manually active module mode
```

You should become familiar with [module-aware `go get`][module-aware-go-get] as it can be used to add version-pinned dependencies out of band of the typical `go mod tidy -v` workflow.

## Adding dependencies

To add a new dependency to dex or update an existing one:

1. Make changes to dex's source code importing the new dependency.
2. You have at least three options as to how to update `go.mod` to reflect the new dependency:
  * Run `go mod tidy -v`. This is a good option if you do not wish to immediately pin to a specific Semantic Version or commit.
  * Run, for example, `go get <package-name>@<commit-hash>`. This is a good option when you want to immediately pin to a specific Semantic Version or commit.
  * Manually update `go.mod`.  If one of the two options above doesn't suit you, do this -- but very carefully.
3. Create a git commit to reflect your code (not vendor) changes. See below for tips on composing commits.
4. Once `go.mod` describes the desired state and you've create a commit for that change, run `make revendor` to update `go.mod`, `go.sum` and `vendor`. This calls `go mod tidy -v`, `go mod vendor -v` and `go mod verify`.
5. Create a git commit to reflect the changes made by `make revendor`. Again, see below for tips on composing commits.

## Composing commits

When composing commits make sure that updates to `vendor` are in a separate commit from the main changes. GitHub's UI makes commits with a large number of changes unreviewable.

Commit histories should look like the following:

```
connector/ldap: add a LDAP connector
vendor: revendor
```

[go-modules]: https://github.com/golang/go/wiki/Modules
[go-modules-activate]: https://github.com/golang/go/wiki/Modules#how-to-install-and-activate-module-support
[go-vendor]: https://golang.org/cmd/go/#hdr-Vendor_Directories
[module-aware-go-get]: https://tip.golang.org/cmd/go/#hdr-Module_aware_go_get

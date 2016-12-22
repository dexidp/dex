# Managing dependencies

Dex uses [glide][glide] and [glide-vc][glide-vc] to manage its [`vendor` directory][go-vendor]. A recent version of these are preferred but dex doesn't require any bleeding edge features. Either install these tools using `go get` or take an opportunity to update to a more recent version.

```
go get -u github.com/Masterminds/glide
go get -u github.com/sgotti/glide-vc
```

To add a new dependency to dex or update an existing one:

* Make changes to dex's source code importing the new dependency.
* Edit `glide.yaml` to include the new dependency at a given commit SHA or change a SHA.
* Add all transitive dependencies of the package to prevent unpinned packages.

Tests will fail if transitive dependencies aren't included. 

Once `glide.yaml` describes the desired state use `make` to update `glide.lock` and `vendor`. This calls both `glide` and `glide-vc` with the set of flags that dex requires.

```
make revendor
```

When composing commits make sure that updates to `vendor` are in a separate commit from the main changes. GitHub's UI makes commits with a large number of changes unreviewable.

Commit histories should look like the following:

```
connector/ldap: add a LDAP connector
vendor: revendor
```

[glide]: https://github.com/Masterminds/glide
[glide-vc]: https://github.com/sgotti/glide-vc
[go-vendor]: https://golang.org/cmd/go/#hdr-Vendor_Directories

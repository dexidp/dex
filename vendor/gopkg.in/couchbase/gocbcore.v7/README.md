# Couchbase Go Core

This package provides the underlying Couchbase IO for the gocb project.
If you are looking for the Couchbase Go SDK, you are probably looking for
[gocb](https://github.com/couchbase/gocb).


## Branching Strategy
The gocbcore library maintains a branch for each major revision of its API.
These branches are introduced just prior to any API breaking changes with a
internal version code of `vX-dev`.  Once a version is fully assembled and
prepared to ship, the version will be updated to reflect a specific full
version number (ie `vX.0.0`), and a tag created for that version number.

An example of this is the v6 branch was created just prior to a breaking
change.  The v6 branch will contain all changes that have a new API until
a release is made (and tag v6.0.0 is created).  At which point, any future
API changes will be made on branch v7.


## License
Copyright 2017 Couchbase Inc.

Licensed under the Apache License, Version 2.0.

See
[LICENSE](https://github.com/couchbase/gocbcore/blob/master/LICENSE)
for further details.

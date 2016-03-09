# dex 0.4 Roadmap

These are the roadmap items for the dex team over the 0.4 release cycle (in no particular order).

## Groups

Start work on groups.

* Add groups (#175)

## Refresh tokens

Finish work on refresh token revocation.

* API endpoints for revoking refresh tokens (#261)

## dexctl rework

Deprecating dexctl’s --db-url flag. Achieve feature parity between existing commands and the bootstrapping API, then have all dexctl actions go through that.

* Overarching issue of deprecating --db-url flag (#298)
* Add client registration to bootstrapping API (#326)
* Set connector configs through bootstrapping API (#360)

## Further server side cleanups

Establish idioms for handling HTTP requests, create a storage interface for backends, and continue to improve --no-db mode.

* Improve server code and storage interfaces (#278)
* Fix client secrets encoding in --no-db mode (#337)
* Easier specification of passwords in --no-db mode (#340)

# Proposal: user objects for revoking refresh tokens and merging accounts

Certain operations require tracking users the have logged in through the server
and storing them in the backend. Namely, allowing end users to revoke refresh
tokens and merging existing accounts with upstream providers.

While revoking refresh tokens is relatively easy, merging accounts is a
difficult problem. What if display names or emails are different? What happens
to a user with two remote identities with the same upstream service? Should
this be presented differently for a user with remote identities for different
upstream services? This proposal only covers a minimal merging implementation
by guaranteeing that merged accounts will always be presented to clients with
the same user ID.

This proposal defines the following objects and methods to be added to the
storage package to allow user information to be persisted.

```go
// User is an end user which has logged in to the server.
//
// Users do not hold additional data, such as emails, because claim information
// is always supplied by an upstream provider during the auth flow. The ID is
// the only information from this object which overrides the claims produced by
// connectors.
//
// Clients which wish to associate additional data with a user must do so on
// their own. The server only guarantees that IDs will be constant for an end
// user, no matter what backend they use to login.
type User struct {
	// A string which uniquely identifies the user for the server. This overrides
	// the ID provided by the connector in the ID Token claims.
	ID string

	// A list of clients who have been issued refresh tokens for this user.
	//
	// When a refresh token is redeemed, the server will check this field to
	// ensure that the client is still on this list. To revoke a client,
	// remove it from here.
	AuthorizedClients []AuthorizedClient

	// A set of remote identities which are able to login as this user.
	RemoteIdentities []RemoteIdentity
}

// AuthorizedClient is a client that has a refresh token out for this user.
type AuthorizedClient struct {
	// The ID of the client.
	ClientID string
	// The last time a token was refreshed.
	LastRefreshed time.Time
}

// RemoteIdentity is the smallest amount of information that identifies a user
// with a remote service. It indicates which remote identities should be able
// to login as a specific user.
//
// RemoteIdentity contains an username so an end user can be displayed this
// object and reason about what upstream profile it represents. It is not used
// to cache claims, such as groups or emails, because these are always provided
// by the upstream identity system during login.
type RemoteIdentity struct {
	// The ID of the connector used to login the user.
	ConnectorID string
	// A string which uniquely identifies the user with the remote system.
	ConnectorUserID stirng

	// Optional, human readable name for this remote identity. Only used when
	// displaying the remote identity to the end user (e.g. when merging
	// accounts). NOT used for determining ID Token claims.
	Username string
}
```

`UserID` fields will be added to the `AuthRequest`, `AuthCode` and `RefreshToken`
structs. When a user logs in successfully through a connector
[here](https://github.com/coreos/dex/blob/95a61454b522edd6643ced36b9d4b9baa8059556/server/handlers.go#L227),
the server will attempt to either get the user, or create one if none exists with
the remote identity.

`AuthorizedClients` serves two roles. First is makes displaying the set of
clients a user is logged into easy. Second, because we don't assume multi-object
transactions, we can't ensure deleting all refresh tokens a client has for a
user. Between listing the set of refresh tokens and deleting a token, a client
may have already redeemed the token and created a new one.

When an OAuth2 client exchanges a code for a token, the following steps are
taken to populate the `AuthorizedClients`:

1. Get token where the user has authorized the `offline_access` scope.
1. Update the user checking authorized clients. If client is not in the list,
add it.
1. Create a refresh token and return the token.

When a OAuth2 client attempts to renew a refresh token, the server ensures that
the token hasn't been revoked.

1. Check authorized clients and update the `LastRefreshed` timestamp. If client
isn't in list error out and delete the refresh token.
1. Continue renewing the refresh token.

When the end user revokes a client, the following steps are used to.

1. Update the authorized clients by removing the client from the list. This
atomic action causes any renew attempts to fail.
1. Iterate through list of refresh tokens and garbage collect any tokens issued
by the user for the client. This isn't atomic, but exists so a user can
re-authorize a client at a later time without authorizing old refresh tokens.

This is clunky due to the lack of multi-object transactions. E.g. we can't delete
all the refresh tokens at once because we don't have that guarantee.

Merging accounts becomes extremely simple. Just add another remote identity to
the user object.

We hope to provide a web interface that a user can login to to perform these
actions. Perhaps using a well known client issued exclusively for the server.

The new `User` object requires adding the following methods to the storage
interface, and (as a nice side effect) deleting the `ListRefreshTokens()` method.

```go
type Storage interface {
	// ...

	CreateUser(u User) error

	DeleteUser(id string) error

	GetUser(id string) error
	GetUserByRemoteIdentity(connectorID, connectorUserID string) (User, error)

	// Updates are assumed to be atomic.
	//
	// When a UpdateUser is called, if clients are removed from the
	// AuthorizedClients list, the underlying storage SHOULD clean up refresh
	// tokens issued for the removed clients. This allows backends with
	// multi-transactional capabilities to utilize them, while key-value stores
	// only guarantee best effort.
	UpdateUser(id string, updater func(old User) (User, error)) error
}
```

Importantly, this will be the first object which has a secondary index.
The Kubernetes client will simply list all the users in memory then iterate over
them to support this (possibly followed by a "watch" based optimization). SQL
implementations will have an easier time.

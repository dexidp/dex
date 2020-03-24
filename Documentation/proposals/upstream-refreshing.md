# Proposal: upstream refreshing

## TL;DR

Today, if a user deletes their GitHub account, dex will keep allowing clients to
refresh tokens on that user's behalf because dex never checks back in with
GitHub.

This is a proposal to change the connector package so the dex can check back
in with GitHub.

## The problem

When dex is federating to an upstream identity provider (IDP), we want to ensure
claims being passed onto clients remain fresh. This includes data such as Google
accounts display names, LDAP group membership, account deactivations. Changes to
these on an upstream IDP should always be reflected in the claims dex passes to
its own clients.

Refresh tokens make this complicated. When refreshing a token, unlike normal
logins, dex doesn't have the opportunity to prompt for user interaction. For
example, if dex is proxying to a LDAP server, it won't have the user's username
and passwords.

Dex can't do this today because connectors have no concept of checking back in
with an upstream provider (with the sole exception of groups). They're only 
called during the initial login, and never consulted when dex needs to mint a
new refresh token for a client. Additionally, connectors aren't actually aware
of the scopes being requested by the client, so they don't know when they should
setup the ability to check back in and have to treat every request identically.

## Changes to the connector package

The biggest changes proposed impact the connector package and connector
implementations.

1. Connectors should be consulted when dex attempts to refresh a token.
2. Connectors should be aware of the scopes requested by the client.

The second bullet is important because of the first. If a client isn't
requesting a refresh token, the connector shouldn't do the extra work, such as
requesting additional upstream scopes.

to address the first point, a top level `Scopes` object will be added to the
connector package to express the scopes requested by the client. The
`CallbackConnector` and `PasswordConnector` will be updated accordingly.

```go
// Scopes represents additional data requested by the clients about the end user.
type Scopes struct{
	// The client has requested a refresh token from the server.
	OfflineAccess bool

	// The client has requested group information about the end user.
	Groups bool
}

// CallbackConnector is an interface implemented by connectors which use an OAuth
// style redirect flow to determine user information.
type CallbackConnector interface {
	// The initial URL to redirect the user to.
	//
	// OAuth2 implementations should request different scopes from the upstream
	// identity provider based on the scopes requested by the downstream client.
	// For example, if the downstream client requests a refresh token from the
	// server, the connector should also request a token from the provider.
	LoginURL(s Scopes, callbackURL, state string) (string, error)

	// Handle the callback to the server and return an identity.
	HandleCallback(s Scopes, r *http.Request) (identity Identity, state string, err error)
}

// PasswordConnector is an interface implemented by connectors which take a
// username and password.
type PasswordConnector interface {
	Login(s Scopes, username, password string) (identity Identity, validPassword bool, err error)
}
```

The existing `GroupsConnector` plays two roles.

1. The connector only attempts to grab groups when the downstream client requests it.
2. Allow group information to be refreshed.

The first issue is remedied by the added `Scopes` struct. This proposal also
hopes to generalize the need of the second role by adding a more general
`RefreshConnector`:

```go
type Identity struct {
	// Existing fields...

	// Groups are added to the identity object, since connectors are now told
	// if they're being requested.

	// The set of groups a user is a member of.
	Groups []string
}


// RefreshConnector is a connector that can update the client claims.
type RefreshConnector interface {
	// Refresh is called when a client attempts to claim a refresh token. The
	// connector should attempt to update the identity object to reflect any
	// changes since the token was last refreshed.
	Refresh(s Scopes, identity Identity) (Identity, error)

	// TODO(ericchiang): Should we allow connectors to indicate that the user has
	// been delete	or an upstream token has been revoked? This would allow us to
	// know when we should remove the downstream refresh token, and when there was
	// just a server error, but might be hard to determine for certain protocols.
	// Might be safer to always delete the downstream token if the Refresh()
	// method returns an error.
}
```

## Example changes to the "passwordDB" connector

The `passwordDB` connector is the internal connector maintained by the server. 
As an example, these are the changes to that connector if this change was
accepted.

```go
func (db passwordDB) Login(s connector.Scopes, username, password string) (connector.Identity, bool, error) {
	// No change to existing implementation. Scopes can be ignored since we'll
	// always have access to the password objects.

}

func (db passwordDB) Refresh(s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	// If the user has been deleted, the refresh token will be rejected.
	p, err := db.s.GetPassword(identity.Email)
	if err != nil {
		if err == storage.ErrNotFound {
			return connector.Identity{}, errors.New("user not found")
		}
		return connector.Identity{}, fmt.Errorf("get password: %v", err)
	}

	// User removed but a new user with the same email exists.
	if p.UserID != identity.UserID {
		return connector.Identity{}, errors.New("user not found")
	}

	// If a user has updated their username, that will be reflected in the
	// refreshed token.
	identity.Username = p.Username
	return identity, nil
}
```

## Caveats

Certain providers, such as Google, will only grant a single refresh token for each
client + end user pair. The second time one's requested, no refresh token is
returned. This means refresh tokens must be stored by dex as objects on an
upstream identity rather than part of a downstream refresh even.

Right now `ConnectorData` is too general for this since it is only stored with a
refresh token and can't be shared between sessions. This should be rethought in
combination with the [`user-object.md`](./user-object.md) proposal to see if
there are reasonable ways for us to do this.

This isn't a problem for providers like GitHub because they return the same
refresh token every time. We don't need to track a token per client.

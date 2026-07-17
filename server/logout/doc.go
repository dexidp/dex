// Package logout implements OIDC RP-Initiated Logout: the /logout and
// /logout/callback endpoints, upstream (connector) single-logout, refresh-token
// revocation and session teardown.
//
// It is a terminal flow, independent of the login/consent/issue sequence: it
// ends a session rather than advancing one. It requires sessions and mounts its
// endpoints only when a session config is present.
package logout

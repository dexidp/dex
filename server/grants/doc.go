// Package grants implements the OAuth2 token endpoint (/token). It defines a
// Grant abstraction — one handler per grant_type — and a Handler that
// dispatches a token request to the grant registered for its grant_type,
// authenticating the client first when the grant requires it.
package grants

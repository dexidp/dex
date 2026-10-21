// Package oauth2 holds the OAuth2/OIDC protocol vocabulary shared across the
// server: the error codes returned in error responses, grant types, response
// types, token-exchange token types, device-flow poll statuses, PKCE methods,
// and the special redirect URIs, plus the OAuth2 error-response writer. Keeping
// them in one place lets the per-domain handler packages reference the protocol
// without depending on the server package.
package oauth2

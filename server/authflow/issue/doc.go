// Package issue writes the OAuth2 authorization response once the interactive
// flow completes: it mints the auth code and, for implicit/hybrid flows, the
// access and ID tokens, then redirects the browser back to the client (or
// renders the out-of-band page).
//
// It is one of the shared flow steps (alongside mfa and consent): the browser
// login flow and, conceptually, any other front-channel flow funnel through it
// to hand control back to the client. One self-selecting handler per
// response_type populates the response, mirroring fosite's
// AuthorizeEndpointHandler model.
package issue

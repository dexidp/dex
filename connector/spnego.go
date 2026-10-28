package connector

import (
	"context"
	"net/http"
)

// Handled indicates whether the SPNEGO-aware connector handled the request.
type Handled bool

// SPNEGOAware is an optional extension for connectors that can authenticate
// users via Kerberos SPNEGO on the initial GET to the password login endpoint.
//
// If handled is true and ident is non-nil, the caller should complete the
// OAuth flow as with a successful password login. If handled is true and
// ident is nil, the implementation has already written an appropriate
// response (e.g., 401 with WWW-Authenticate: Negotiate) and the caller should
// return without rendering the password form. If handled is false, proceed
// with the legacy password form flow.
type SPNEGOAware interface {
	TrySPNEGO(ctx context.Context, s Scopes, w http.ResponseWriter, r *http.Request) (*Identity, Handled, error)
}

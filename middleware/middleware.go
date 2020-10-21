// Package middleware defines interfaces for pluggable identity middleware.
package middleware

import (
	"context"

	"github.com/dexidp/dex/connector"
)

// Middleware is a mechanism for allowing customisation of responses returned
// by a remote identity service.
//
// Each configured connector can have a stack of Middleware components; when
// the connector returns successfully with an Identity, this will be passed to
// the Middleware at the top of the stack, which can inspect the identity and
// take any required actions.  Assuming that Middleware component succeeds
// and returns an Identity, the returned identity will be passed to the next
// Middleware in the chain until all middleware modules have processed the
// identity.
//
// Once the connector specific middleware has finished executing, there is also
// a global middleware chain that runs on the results.
type Middleware interface {
	Process(ctx context.Context, identity connector.Identity) (connector.Identity, error)
}

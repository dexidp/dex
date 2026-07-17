// Package log carries per-request logging attributes (request ID, client IP)
// through the context, from the server's middleware down into the handlers and
// the CLI logger. It is a neutral, low-level home so the server, the auth flow
// and its sub-packages, and the CLI can all reference the same keys.
package log

import (
	"context"

	"github.com/google/uuid"
)

type key string

const (
	RequestKeyRequestID key = "request_id"
	RequestKeyRemoteIP  key = "client_remote_addr"
)

// WithRequestID attaches a fresh request ID to the context.
func WithRequestID(ctx context.Context) context.Context {
	return context.WithValue(ctx, RequestKeyRequestID, uuid.NewString())
}

// WithRemoteIP attaches the client's remote IP to the context.
func WithRemoteIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, RequestKeyRemoteIP, ip)
}

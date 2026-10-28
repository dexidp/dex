package reqctx

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

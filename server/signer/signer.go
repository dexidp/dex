package signer

import (
	"context"

	"github.com/go-jose/go-jose/v4"
)

// Signer is an interface for signing payloads and retrieving validation keys.
type Signer interface {
	// Sign signs the provided payload.
	Sign(ctx context.Context, payload []byte) (string, error)
	// ValidationKeys returns the current public keys used for signature validation.
	ValidationKeys(ctx context.Context) ([]*jose.JSONWebKey, error)
	// Algorithm returns the signing algorithm used by this signer.
	Algorithm(ctx context.Context) (jose.SignatureAlgorithm, error)
	// Start starts any background tasks required by the signer (e.g., key rotation).
	Start(ctx context.Context)
}

package signer

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage/memory"
)

func newTestLocalSigner(t *testing.T) *localSigner {
	t.Helper()

	logger := slog.New(slog.DiscardHandler)
	s := memory.New(logger)
	r := &keyRotator{
		Storage:  s,
		strategy: defaultRotationStrategy(time.Hour, time.Hour),
		now:      time.Now,
		logger:   logger,
	}

	return &localSigner{
		storage: s,
		rotator: r,
		logger:  logger,
	}
}

func TestLocalSignerAlgorithm(t *testing.T) {
	ls := newTestLocalSigner(t)

	// Algorithm should return RS256 even before keys are rotated (empty storage).
	alg, err := ls.Algorithm(context.Background())
	require.NoError(t, err)
	assert.Equal(t, jose.RS256, alg)
}

func TestLocalSignerSignAndValidate(t *testing.T) {
	ls := newTestLocalSigner(t)
	ctx := context.Background()

	// Rotate keys so we have a signing key.
	require.NoError(t, ls.rotator.rotate())

	payload := []byte(`{"sub":"test-user"}`)
	signed, err := ls.Sign(ctx, payload)
	require.NoError(t, err)
	assert.NotEmpty(t, signed)

	// Validation keys should be available.
	keys, err := ls.ValidationKeys(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, keys)
}

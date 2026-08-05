package tokens

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func TestLookupRefreshTokenUsesConnectorExpiryOverride(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)
	store := memory.New(logger)

	// t0 is far in the future so nothing accidentally ages on wall time.
	t0 := time.Date(2050, 1, 1, 0, 0, 0, 0, time.UTC)
	now := func() time.Time { return t0.Add(2 * time.Minute) }

	e := NewExpiryPolicy(time.Hour, NewRefreshStrategy(true, 0, 0, 0, now))
	require.NoError(t, e.Upsert("strict", &storage.ConnectorExpiry{
		RefreshTokens: &storage.ConnectorRefreshExpiry{ValidIfNotUsedFor: "1m"},
	}))

	for id, connID := range map[string]string{"r1": "strict", "r2": "lax"} {
		require.NoError(t, store.CreateRefresh(ctx, storage.RefreshToken{
			ID:          id,
			Token:       "token",
			ClientID:    "client",
			ConnectorID: connID,
			CreatedAt:   t0,
			LastUsed:    t0,
		}))
	}

	// The strict connector's override has aged its token out; the lax one
	// runs on the global strategy, which never expires it.
	_, err := LookupRefreshToken(ctx, store, e, logger, nil, &internal.RefreshToken{RefreshId: "r1", Token: "token"})
	require.ErrorIs(t, err, ErrRefreshTokenExpired)

	got, err := LookupRefreshToken(ctx, store, e, logger, nil, &internal.RefreshToken{RefreshId: "r2", Token: "token"})
	require.NoError(t, err)
	require.Equal(t, "r2", got.ID)
}

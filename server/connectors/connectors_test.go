package connectors

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

// stubConn is a trivial connector.Connector (which is interface{}) tagged with
// the version it was resolved from, so tests can tell reopens apart.
type stubConn struct{ version string }

func newTestCache(t *testing.T) (*Cache, storage.Storage, *int) {
	t.Helper()
	store := memory.New(slog.New(slog.DiscardHandler))
	calls := 0
	resolve := func(c storage.Connector) (connector.Connector, error) {
		calls++
		return stubConn{version: c.ResourceVersion}, nil
	}
	return NewCache(store, resolve), store, &calls
}

func TestGetOpensAndCaches(t *testing.T) {
	ctx := context.Background()
	cache, store, calls := newTestCache(t)

	require.NoError(t, store.CreateConnector(ctx, storage.Connector{
		ID: "c1", Type: "mock", ResourceVersion: "1", GrantTypes: []string{"authorization_code"},
	}))

	got, err := cache.Get(ctx, "c1")
	require.NoError(t, err)
	require.Equal(t, 1, *calls)
	require.Equal(t, "mock", got.Type)
	require.Equal(t, "1", got.ResourceVersion)
	require.Equal(t, []string{"authorization_code"}, got.GrantTypes)
	require.Equal(t, stubConn{version: "1"}, got.Connector)

	// Second Get hits the cache; the connector is not resolved again.
	got2, err := cache.Get(ctx, "c1")
	require.NoError(t, err)
	require.Equal(t, 1, *calls)
	require.Equal(t, got, got2)
}

func TestGetReopensOnVersionChange(t *testing.T) {
	ctx := context.Background()
	cache, store, calls := newTestCache(t)

	require.NoError(t, store.CreateConnector(ctx, storage.Connector{ID: "c1", Type: "mock", ResourceVersion: "1"}))
	_, err := cache.Get(ctx, "c1")
	require.NoError(t, err)
	require.Equal(t, 1, *calls)

	// A stored resource-version bump must invalidate the cached entry.
	require.NoError(t, store.UpdateConnector(ctx, "c1", func(old storage.Connector) (storage.Connector, error) {
		old.ResourceVersion = "2"
		return old, nil
	}))

	got, err := cache.Get(ctx, "c1")
	require.NoError(t, err)
	require.Equal(t, 2, *calls)
	require.Equal(t, "2", got.ResourceVersion)
	require.Equal(t, stubConn{version: "2"}, got.Connector)
}

func TestGetNotFound(t *testing.T) {
	ctx := context.Background()
	cache, _, calls := newTestCache(t)

	_, err := cache.Get(ctx, "missing")
	require.Error(t, err)
	require.Equal(t, 0, *calls)
}

func TestOpenResolveErrorNotCached(t *testing.T) {
	store := memory.New(slog.New(slog.DiscardHandler))
	cache := NewCache(store, func(storage.Connector) (connector.Connector, error) {
		return nil, errors.New("boom")
	})

	_, err := cache.Open(storage.Connector{ID: "c1"})
	require.Error(t, err)

	_, ok := cache.Cached("c1")
	require.False(t, ok)
	require.Equal(t, 0, cache.Len())
}

func TestSetCachedCloseLen(t *testing.T) {
	cache, _, _ := newTestCache(t)

	require.Equal(t, 0, cache.Len())
	_, ok := cache.Cached("c1")
	require.False(t, ok)

	cache.Set("c1", Connector{Type: "mock", ResourceVersion: "1", Connector: stubConn{version: "1"}})
	cache.Set("c2", Connector{Type: "ldap"})
	require.Equal(t, 2, cache.Len())

	got, ok := cache.Cached("c1")
	require.True(t, ok)
	require.Equal(t, "mock", got.Type)

	cache.Close("c1")
	_, ok = cache.Cached("c1")
	require.False(t, ok)
	require.Equal(t, 1, cache.Len())

	// Closing an unknown id is a no-op.
	cache.Close("nope")
	require.Equal(t, 1, cache.Len())
}

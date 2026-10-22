package connectors

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

// stubConn is a trivial connector.Connector (which is interface{}) tagged with
// the version it was resolved from, so tests can tell reopens apart.
type stubConn struct{ version string }

// lifecycleStub records lifecycle callbacks so tests can assert they fired.
type lifecycleStub struct {
	version string
	started int
	closed  int
}

func (l *lifecycleStub) Start(ctx context.Context) error {
	l.started++
	return nil
}

func (l *lifecycleStub) Close() {
	l.closed++
}

func newTestCache(t *testing.T, ctx context.Context) (*Cache, storage.Storage, *int) {
	t.Helper()
	store := memory.New(slog.New(slog.DiscardHandler))
	calls := 0
	resolve := func(c storage.Connector) (connector.Connector, error) {
		calls++
		return stubConn{version: c.ResourceVersion}, nil
	}
	return NewCache(ctx, store, resolve), store, &calls
}

func TestGetOpensAndCaches(t *testing.T) {
	ctx := context.Background()
	cache, store, calls := newTestCache(t, ctx)

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
	cache, store, calls := newTestCache(t, ctx)

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
	cache, _, calls := newTestCache(t, ctx)

	_, err := cache.Get(ctx, "missing")
	require.Error(t, err)
	require.Equal(t, 0, *calls)
}

func TestOpenResolveErrorNotCached(t *testing.T) {
	ctx := context.Background()
	store := memory.New(slog.New(slog.DiscardHandler))
	cache := NewCache(ctx, store, func(storage.Connector) (connector.Connector, error) {
		return nil, errors.New("boom")
	})

	_, err := cache.Open(storage.Connector{ID: "c1"})
	require.Error(t, err)

	_, ok := cache.Cached("c1")
	require.False(t, ok)
	require.Equal(t, 0, cache.Len())
}

func TestSetCachedCloseLen(t *testing.T) {
	ctx := context.Background()
	cache, _, _ := newTestCache(t, ctx)

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

func TestLifecycleStartAndClose(t *testing.T) {
	ctx := context.Background()
	store := memory.New(slog.New(slog.DiscardHandler))

	var current *lifecycleStub
	cache := NewCache(ctx, store, func(c storage.Connector) (connector.Connector, error) {
		current = &lifecycleStub{version: c.ResourceVersion}
		return current, nil
	})

	require.NoError(t, store.CreateConnector(ctx, storage.Connector{ID: "c1", Type: "mock", ResourceVersion: "1"}))

	// Open starts the lifecycle.
	_, err := cache.Open(storage.Connector{ID: "c1", Type: "mock", ResourceVersion: "1"})
	require.NoError(t, err)
	require.Equal(t, 1, current.started)
	require.Equal(t, 0, current.closed)

	// Reopening the same ID stops the previous instance and starts the new one.
	first := current
	_, err = cache.Open(storage.Connector{ID: "c1", Type: "mock", ResourceVersion: "2"})
	require.NoError(t, err)
	require.Equal(t, 1, first.closed)
	require.NotSame(t, first, current)
	require.Equal(t, 1, current.started)

	// Cache.Close stops the lifecycle.
	cache.Close("c1")
	require.Equal(t, 1, current.closed)
}

func TestLifecycleNotRequired(t *testing.T) {
	ctx := context.Background()
	cache, _, _ := newTestCache(t, ctx)

	// stubConn does not implement LifecycleConnector; Open/Close must not panic.
	cache.Set("c1", Connector{Type: "mock", Connector: stubConn{version: "1"}})
	cache.Close("c1")
}

// TestLifecycleConcurrentOpenSameID verifies that concurrent opens of the same
// connector ID start a single lifecycle instance: the losing goroutine must not
// leak a started instance that nothing will ever close.
func TestLifecycleConcurrentOpenSameID(t *testing.T) {
	ctx := context.Background()
	store := memory.New(slog.New(slog.DiscardHandler))

	var (
		mu        sync.Mutex
		instances []*lifecycleStub
	)
	cache := NewCache(ctx, store, func(c storage.Connector) (connector.Connector, error) {
		inst := &lifecycleStub{version: c.ResourceVersion}
		mu.Lock()
		instances = append(instances, inst)
		mu.Unlock()
		return inst, nil
	})

	conn := storage.Connector{ID: "c1", Type: "mock", ResourceVersion: "1"}

	const n = 8
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = cache.Open(conn)
		}()
	}
	wg.Wait()

	// Exactly one instance may have been started; every started instance that is
	// not the cached one must have been closed.
	cached, ok := cache.Cached("c1")
	require.True(t, ok)
	cachedLifecycle, ok := cached.Connector.(*lifecycleStub)
	require.True(t, ok)

	mu.Lock()
	created := append([]*lifecycleStub(nil), instances...)
	mu.Unlock()

	var started, unclosed int
	for _, inst := range created {
		if inst.started == 1 {
			started++
		}
		if inst.started > inst.closed {
			unclosed++
		}
	}

	// Serialization guarantees only the winner is live.
	require.Equal(t, 1, started, "expected exactly one started instance")
	require.Equal(t, 1, unclosed, "expected exactly one live (started, unclosed) instance")
	require.Same(t, cachedLifecycle, func() *lifecycleStub {
		for _, inst := range created {
			if inst.started == 1 && inst.closed == 0 {
				return inst
			}
		}
		return nil
	}())
	require.Equal(t, 1, cachedLifecycle.started)
}

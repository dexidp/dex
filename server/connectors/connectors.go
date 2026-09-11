package connectors

import (
	"context"
	"fmt"
	"sync"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/storage"
)

// Connector is a connector with resource version metadata.
type Connector struct {
	Type            string
	ResourceVersion string
	Connector       connector.Connector
	GrantTypes      []string
}

// ResolveFunc builds the underlying connector implementation for a stored
// connector. The server injects it so that connector construction (the local
// password DB and the connector-config registry) stays in the server package.
type ResolveFunc func(storage.Connector) (connector.Connector, error)

// Cache resolves connectors from storage and keeps the opened instances in
// memory, refreshing an entry when its stored resource version changes. It is
// the sole owner of the connector map and its mutex.
type Cache struct {
	mu      sync.Mutex
	conns   map[string]Connector
	storage storage.Storage
	resolve ResolveFunc
	ctx     context.Context

	// openLocks serializes Open calls per connector ID so concurrent misses for
	// the same ID resolve and start a single lifecycle-managed instance rather
	// than leaking one. Entries are created on first use and never removed; the
	// number of connector IDs is small and bounded.
	openLocks map[string]*sync.Mutex
}

// NewCache returns an empty cache backed by the given storage and resolver.
// ctx bounds the lifetime of any connector background work started via Open.
func NewCache(ctx context.Context, storage storage.Storage, resolve ResolveFunc) *Cache {
	return &Cache{
		conns:     make(map[string]Connector),
		storage:   storage,
		resolve:   resolve,
		ctx:       ctx,
		openLocks: make(map[string]*sync.Mutex),
	}
}

// openLock returns the per-ID mutex that serializes Open calls for connID.
func (c *Cache) openLock(connID string) *sync.Mutex {
	c.mu.Lock()
	defer c.mu.Unlock()
	l, ok := c.openLocks[connID]
	if !ok {
		l = &sync.Mutex{}
		c.openLocks[connID] = l
	}
	return l
}

// Open builds the connector for the given stored connector and records it in the
// cache, replacing any existing entry for the same ID.
func (c *Cache) Open(conn storage.Connector) (Connector, error) {
	// Serialize per ID so concurrent opens of the same connector resolve and
	// start a single instance; otherwise the loser of a concurrent race would
	// leak its started background work with no handle to stop it.
	lock := c.openLock(conn.ID)
	lock.Lock()
	defer lock.Unlock()

	// Re-check under the per-ID lock: a concurrent opener may have already
	// built and cached exactly this connector. Return it rather than starting
	// a second lifecycle instance.
	c.mu.Lock()
	existing, ok := c.conns[conn.ID]
	c.mu.Unlock()
	if ok && existing.ResourceVersion == conn.ResourceVersion && existing.Type == conn.Type {
		return existing, nil
	}

	impl, err := c.resolve(conn)
	if err != nil {
		return Connector{}, fmt.Errorf("failed to open connector: %v", err)
	}

	// Stop any lifecycle-managed instance previously cached under this ID.
	c.mu.Lock()
	prev, hadPrev := c.conns[conn.ID]
	c.mu.Unlock()
	if hadPrev {
		if lc, ok := prev.Connector.(connector.LifecycleConnector); ok {
			lc.Close()
		}
	}

	if lc, ok := impl.(connector.LifecycleConnector); ok {
		if err := lc.Start(c.ctx); err != nil {
			return Connector{}, fmt.Errorf("failed to start connector %s: %v", conn.ID, err)
		}
	}

	opened := Connector{
		Type:            conn.Type,
		ResourceVersion: conn.ResourceVersion,
		Connector:       impl,
		GrantTypes:      conn.GrantTypes,
	}

	c.mu.Lock()
	c.conns[conn.ID] = opened
	c.mu.Unlock()

	return opened, nil
}

// Set records an already-opened connector under the given id, replacing any
// existing entry. It is used to inject connectors that are not built from stored
// config (for example the built-in local connector, or mocks in tests).
func (c *Cache) Set(id string, conn Connector) {
	c.mu.Lock()
	c.conns[id] = conn
	c.mu.Unlock()
}

// Cached returns the connector currently held for id without consulting storage.
func (c *Cache) Cached(id string) (Connector, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	conn, ok := c.conns[id]
	return conn, ok
}

// Len reports the number of cached connectors.
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.conns)
}

// Close stops any lifecycle-managed work and removes the connector from the
// in-memory cache.
func (c *Cache) Close(id string) {
	c.mu.Lock()
	conn, ok := c.conns[id]
	if ok {
		delete(c.conns, id)
	}
	c.mu.Unlock()

	if ok {
		if lc, ok := conn.Connector.(connector.LifecycleConnector); ok {
			lc.Close()
		}
	}
}

// Get returns the connector with the given id, opening (or reopening) it when it
// is missing from the cache or its stored resource version has changed.
func (c *Cache) Get(ctx context.Context, id string) (Connector, error) {
	storageConnector, err := c.storage.GetConnector(ctx, id)
	if err != nil {
		return Connector{}, fmt.Errorf("failed to get connector object from storage: %v", err)
	}

	c.mu.Lock()
	conn, ok := c.conns[id]
	c.mu.Unlock()

	if !ok || storageConnector.ResourceVersion != conn.ResourceVersion {
		// Not cached, or updated in storage since we last opened it.
		return c.Open(storageConnector)
	}

	return conn, nil
}

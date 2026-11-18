package tokens

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func newTestStore(t *testing.T) (*RefreshStore, storage.Storage) {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)
	store := memory.New(logger)
	return NewRefreshStore(store, time.Now, logger), store
}

func TestRefreshStoreCreate(t *testing.T) {
	ctx := context.Background()
	rt, store := newTestStore(t)
	auth := testAuthorization()

	first, err := rt.Create(ctx, auth)
	require.NoError(t, err)
	var firstTok internal.RefreshToken
	require.NoError(t, internal.Unmarshal(first, &firstTok))

	stored, err := store.GetRefresh(ctx, firstTok.RefreshId)
	require.NoError(t, err)
	require.Equal(t, "client-1", stored.ClientID)
	require.Equal(t, auth.ConnectorData, stored.ConnectorData)

	sess, err := store.GetOfflineSessions(ctx, "u1", "mock")
	require.NoError(t, err)
	require.Contains(t, sess.Refresh, "client-1")

	// A second create for the same client replaces the old token.
	second, err := rt.Create(ctx, auth)
	require.NoError(t, err)
	var secondTok internal.RefreshToken
	require.NoError(t, internal.Unmarshal(second, &secondTok))
	require.NotEqual(t, firstTok.RefreshId, secondTok.RefreshId)

	_, err = store.GetRefresh(ctx, firstTok.RefreshId)
	require.ErrorIs(t, err, storage.ErrNotFound)

	sess, err = store.GetOfflineSessions(ctx, "u1", "mock")
	require.NoError(t, err)
	require.Equal(t, secondTok.RefreshId, sess.Refresh["client-1"].ID)
}

func TestRefreshStoreRotate(t *testing.T) {
	ctx := context.Background()
	rt, store := newTestStore(t)

	require.NoError(t, store.CreateRefresh(ctx, storage.RefreshToken{
		ID: "r1", Token: "t1", ClientID: "client-1", ConnectorID: "mock",
		Claims: storage.Claims{UserID: "u1", Username: "alice"}, CreatedAt: time.Now(), LastUsed: time.Now(),
	}))
	require.NoError(t, store.CreateOfflineSessions(ctx, storage.OfflineSessions{
		UserID: "u1", ConnID: "mock",
		Refresh: map[string]*storage.RefreshTokenRef{"client-1": {ID: "r1", ClientID: "client-1"}},
	}))
	stored, err := store.GetRefresh(ctx, "r1")
	require.NoError(t, err)

	strategy := NewRefreshStrategy(true, 0, 0, 0, nil) // rotation on, no reuse
	ident := connector.Identity{UserID: "u1", Username: "bob", Email: "bob@example.com"}
	freshIdentity := func(context.Context) (connector.Identity, error) { return ident, nil }

	raw, gotIdent, err := rt.Rotate(ctx, &stored, &internal.RefreshToken{RefreshId: "r1", Token: "t1"}, strategy, freshIdentity)
	require.NoError(t, err)
	require.Equal(t, ident, gotIdent)

	var newTok internal.RefreshToken
	require.NoError(t, internal.Unmarshal(raw, &newTok))
	require.Equal(t, "r1", newTok.RefreshId)
	require.NotEqual(t, "t1", newTok.Token, "token should have rotated")

	after, err := store.GetRefresh(ctx, "r1")
	require.NoError(t, err)
	require.Equal(t, newTok.Token, after.Token)
	require.Equal(t, "t1", after.ObsoleteToken)
	require.Equal(t, "bob", after.Claims.Username, "claims refreshed from the identity")

	// Claiming with a token that is neither current nor obsolete is rejected.
	_, _, err = rt.Rotate(ctx, &after, &internal.RefreshToken{RefreshId: "r1", Token: "wrong"}, strategy, freshIdentity)
	require.Error(t, err)
}

func TestRefreshStoreRevoke(t *testing.T) {
	ctx := context.Background()
	rt, store := newTestStore(t)

	for _, tc := range []struct{ id, client string }{{"r1", "c1"}, {"r2", "c2"}} {
		require.NoError(t, store.CreateRefresh(ctx, storage.RefreshToken{
			ID: tc.id, ClientID: tc.client, ConnectorID: "mock", Claims: storage.Claims{UserID: "u1"},
		}))
	}
	require.NoError(t, store.CreateOfflineSessions(ctx, storage.OfflineSessions{
		UserID: "u1", ConnID: "mock", ConnectorData: []byte("cd"),
		Refresh: map[string]*storage.RefreshTokenRef{
			"c1": {ID: "r1", ClientID: "c1"},
			"c2": {ID: "r2", ClientID: "c2"},
		},
	}))

	// RevokeClients is scoped to the named clients and leaves the rest of the user's
	// tokens alone.
	rt.RevokeClients(ctx, "u1", "mock", []string{"c1", "unknown"})

	_, err := store.GetRefresh(ctx, "r1")
	require.ErrorIs(t, err, storage.ErrNotFound)
	_, err = store.GetRefresh(ctx, "r2")
	require.NoError(t, err)

	sess, err := store.GetOfflineSessions(ctx, "u1", "mock")
	require.NoError(t, err)
	require.NotContains(t, sess.Refresh, "c1")
	require.Contains(t, sess.Refresh, "c2")

	// RevokeAll is the unscoped variant. The offline session survives with its refs
	// cleared and its connector data intact.
	rt.RevokeAll(ctx, "u1", "mock")

	_, err = store.GetRefresh(ctx, "r2")
	require.ErrorIs(t, err, storage.ErrNotFound)

	sess, err = store.GetOfflineSessions(ctx, "u1", "mock")
	require.NoError(t, err)
	require.Empty(t, sess.Refresh)
	require.Equal(t, []byte("cd"), sess.ConnectorData)
}

// seedRefreshToken stores a refresh token and its offline session with the
// given last-used time, aged past the reuse window by the caller.
func seedRefreshToken(t *testing.T, store storage.Storage, lastUsed time.Time) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, store.CreateRefresh(ctx, storage.RefreshToken{
		ID: "r1", Token: "t1", ObsoleteToken: "t0", ClientID: "client-1", ConnectorID: "mock",
		Claims: storage.Claims{UserID: "u1", Username: "alice"}, CreatedAt: lastUsed, LastUsed: lastUsed,
	}))
	require.NoError(t, store.CreateOfflineSessions(ctx, storage.OfflineSessions{
		UserID: "u1", ConnID: "mock",
		Refresh: map[string]*storage.RefreshTokenRef{"client-1": {ID: "r1", ClientID: "client-1"}},
	}))
}

// blockingStorage parks the second UpdateRefreshToken call, the finalize of the
// first rotation, until released. Every other call passes through.
type blockingStorage struct {
	storage.Storage
	inGap   chan struct{}
	release chan struct{}
	updates atomic.Int32
}

// waitInGap waits until a rotation is parked between its claim and finalize.
func (b *blockingStorage) waitInGap(t *testing.T) {
	t.Helper()
	select {
	case <-b.inGap:
	case <-time.After(5 * time.Second):
		t.Fatal("rotation never reached the gap between claim and finalize")
	}
}

func (b *blockingStorage) UpdateRefreshToken(ctx context.Context, id string, updater func(r storage.RefreshToken) (storage.RefreshToken, error)) error {
	if b.updates.Add(1) == 2 {
		close(b.inGap)
		<-b.release
	}
	return b.Storage.UpdateRefreshToken(ctx, id, updater)
}

// failingFinalizeStorage fails the second UpdateRefreshToken call, the finalize
// of the first rotation, once.
type failingFinalizeStorage struct {
	storage.Storage
	failNow atomic.Bool
	updates atomic.Int32
}

func (f *failingFinalizeStorage) UpdateRefreshToken(ctx context.Context, id string, updater func(r storage.RefreshToken) (storage.RefreshToken, error)) error {
	if f.updates.Add(1) == 2 && f.failNow.CompareAndSwap(true, false) {
		return errors.New("injected finalize failure")
	}
	return f.Storage.UpdateRefreshToken(ctx, id, updater)
}

// failingOfflineSessionStorage fails the next UpdateOfflineSessions call once
// and counts UpdateRefreshToken calls.
type failingOfflineSessionStorage struct {
	storage.Storage
	failNow atomic.Bool
	updates atomic.Int32
}

func (f *failingOfflineSessionStorage) UpdateRefreshToken(ctx context.Context, id string, updater func(r storage.RefreshToken) (storage.RefreshToken, error)) error {
	f.updates.Add(1)
	return f.Storage.UpdateRefreshToken(ctx, id, updater)
}

func (f *failingOfflineSessionStorage) UpdateOfflineSessions(ctx context.Context, userID, connID string, updater func(s storage.OfflineSessions) (storage.OfflineSessions, error)) error {
	if f.failNow.CompareAndSwap(true, false) {
		return errors.New("injected offline session failure")
	}
	return f.Storage.UpdateOfflineSessions(ctx, userID, connID, updater)
}

func TestRefreshStoreRotateConcurrentRefreshReusesClaimedToken(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	clock := func() time.Time { return now }

	blocked := &blockingStorage{
		Storage: memory.New(slog.New(slog.DiscardHandler)),
		inGap:   make(chan struct{}),
		release: make(chan struct{}),
	}
	rt := NewRefreshStore(blocked, clock, slog.New(slog.DiscardHandler))
	seedRefreshToken(t, blocked.Storage, now.Add(-31*time.Second))
	strategy := NewRefreshStrategy(true, 0, 0, 30*time.Second, clock)

	var resolves atomic.Int32
	fresh := func(context.Context) (connector.Identity, error) {
		resolves.Add(1)
		return connector.Identity{UserID: "u1", Username: "bob"}, nil
	}

	stored, err := blocked.GetRefresh(ctx, "r1")
	require.NoError(t, err)
	req := &internal.RefreshToken{RefreshId: "r1", Token: "t1"}

	var winnerRaw string
	var winnerErr error
	winnerDone := make(chan struct{})
	go func() {
		defer close(winnerDone)
		winnerRaw, _, winnerErr = rt.Rotate(ctx, &stored, req, strategy, fresh)
	}()
	blocked.waitInGap(t)

	// The concurrent refresh, still holding the pre-rotation token, takes the
	// reuse path the claim opened and receives the token being minted.
	racerRaw, _, racerErr := rt.Rotate(ctx, &stored, req, strategy, fresh)

	close(blocked.release)
	<-winnerDone

	require.NoError(t, winnerErr)
	require.NoError(t, racerErr)
	var winnerTok, racerTok internal.RefreshToken
	require.NoError(t, internal.Unmarshal(winnerRaw, &winnerTok))
	require.NoError(t, internal.Unmarshal(racerRaw, &racerTok))
	require.NotEqual(t, "t1", winnerTok.Token, "token should have rotated")
	require.Equal(t, winnerTok.Token, racerTok.Token, "concurrent refresh must receive the same new token")
	require.Equal(t, int32(1), resolves.Load(), "the connector must be resolved once")
}

func TestRefreshStoreRotateConcurrentRefreshWithoutReuseWindow(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	clock := func() time.Time { return now }

	blocked := &blockingStorage{
		Storage: memory.New(slog.New(slog.DiscardHandler)),
		inGap:   make(chan struct{}),
		release: make(chan struct{}),
	}
	rt := NewRefreshStore(blocked, clock, slog.New(slog.DiscardHandler))
	seedRefreshToken(t, blocked.Storage, now.Add(-31*time.Second))
	strategy := NewRefreshStrategy(true, 0, 0, 0, clock)

	var resolves atomic.Int32
	fresh := func(context.Context) (connector.Identity, error) {
		resolves.Add(1)
		return connector.Identity{UserID: "u1", Username: "bob"}, nil
	}

	stored, err := blocked.GetRefresh(ctx, "r1")
	require.NoError(t, err)
	req := &internal.RefreshToken{RefreshId: "r1", Token: "t1"}

	var winnerErr error
	winnerDone := make(chan struct{})
	go func() {
		defer close(winnerDone)
		_, _, winnerErr = rt.Rotate(ctx, &stored, req, strategy, fresh)
	}()
	blocked.waitInGap(t)

	// Without a reuse window there is no path back onto the token: the claim
	// already rotated it, so the concurrent refresh is denied before resolving.
	_, _, racerErr := rt.Rotate(ctx, &stored, req, strategy, fresh)

	close(blocked.release)
	<-winnerDone

	require.NoError(t, winnerErr)
	require.Error(t, racerErr, "without a reuse window the concurrent refresh must be denied")
	require.Equal(t, int32(1), resolves.Load(), "only the winner resolves the connector")
}

func TestRefreshStoreRotateResolveFailureUndoesClaim(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	clock := func() time.Time { return now }

	store := memory.New(slog.New(slog.DiscardHandler))
	rt := NewRefreshStore(store, clock, slog.New(slog.DiscardHandler))
	aged := now.Add(-31 * time.Second)
	seedRefreshToken(t, store, aged)
	strategy := NewRefreshStrategy(true, 0, 0, 30*time.Second, clock)

	resolveErr := errors.New("user deleted")
	var resolves atomic.Int32
	fresh := func(context.Context) (connector.Identity, error) {
		resolves.Add(1)
		if resolveErr != nil {
			return connector.Identity{}, resolveErr
		}
		return connector.Identity{UserID: "u1", Username: "bob"}, nil
	}

	stored, err := store.GetRefresh(ctx, "r1")
	require.NoError(t, err)
	req := &internal.RefreshToken{RefreshId: "r1", Token: "t1"}

	_, _, err = rt.Rotate(ctx, &stored, req, strategy, fresh)
	require.ErrorIs(t, err, resolveErr)

	after, err := store.GetRefresh(ctx, "r1")
	require.NoError(t, err)
	require.Equal(t, "t1", after.Token, "a failed rotation must restore the token")
	require.Equal(t, "t0", after.ObsoleteToken, "a failed rotation must restore the obsolete token")
	require.Equal(t, aged, after.LastUsed, "a failed rotation must restore last-used")

	// The undo closed the reuse window, so a retry rotates again and fails
	// while the user is still gone instead of minting from stale claims.
	_, _, err = rt.Rotate(ctx, &stored, req, strategy, fresh)
	require.ErrorIs(t, err, resolveErr)

	resolveErr = nil
	raw, ident, err := rt.Rotate(ctx, &stored, req, strategy, fresh)
	require.NoError(t, err)
	require.Equal(t, "bob", ident.Username)

	var tok internal.RefreshToken
	require.NoError(t, internal.Unmarshal(raw, &tok))
	require.NotEqual(t, "t1", tok.Token)

	after, err = store.GetRefresh(ctx, "r1")
	require.NoError(t, err)
	require.Equal(t, tok.Token, after.Token)
	require.Equal(t, "t1", after.ObsoleteToken)
	require.Equal(t, "bob", after.Claims.Username)
	require.Equal(t, int32(3), resolves.Load())
}

func TestRefreshStoreRotateFinalizeFailureUndoesClaim(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	clock := func() time.Time { return now }

	failing := &failingFinalizeStorage{Storage: memory.New(slog.New(slog.DiscardHandler))}
	failing.failNow.Store(true)
	rt := NewRefreshStore(failing, clock, slog.New(slog.DiscardHandler))
	aged := now.Add(-31 * time.Second)
	seedRefreshToken(t, failing.Storage, aged)
	strategy := NewRefreshStrategy(true, 0, 0, 30*time.Second, clock)

	var resolves atomic.Int32
	fresh := func(context.Context) (connector.Identity, error) {
		resolves.Add(1)
		return connector.Identity{UserID: "u1", Username: "bob"}, nil
	}

	stored, err := failing.GetRefresh(ctx, "r1")
	require.NoError(t, err)
	req := &internal.RefreshToken{RefreshId: "r1", Token: "t1"}

	_, _, err = rt.Rotate(ctx, &stored, req, strategy, fresh)
	require.Error(t, err, "the injected finalize failure must fail the grant")

	after, err := failing.GetRefresh(ctx, "r1")
	require.NoError(t, err)
	require.Equal(t, "t1", after.Token, "a failed finalize must restore the token")
	require.Equal(t, "t0", after.ObsoleteToken, "a failed finalize must restore the obsolete token")
	require.Equal(t, aged, after.LastUsed, "a failed finalize must restore last-used")

	// Within the window the retry would have taken the reuse path had the claim
	// stood; with it undone the retry re-rotates.
	now = now.Add(time.Second)
	raw, _, err := rt.Rotate(ctx, &stored, req, strategy, fresh)
	require.NoError(t, err)
	var tok internal.RefreshToken
	require.NoError(t, internal.Unmarshal(raw, &tok))
	require.NotEqual(t, "t1", tok.Token, "in-window retry must re-rotate, not reuse the un-rotated token")

	after, err = failing.GetRefresh(ctx, "r1")
	require.NoError(t, err)
	require.Equal(t, "bob", after.Claims.Username)
	require.Equal(t, int32(2), resolves.Load())
}

func TestRefreshStoreRotateOfflineSessionFailureUndoesClaim(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	clock := func() time.Time { return now }

	failing := &failingOfflineSessionStorage{Storage: memory.New(slog.New(slog.DiscardHandler))}
	failing.failNow.Store(true)
	rt := NewRefreshStore(failing, clock, slog.New(slog.DiscardHandler))
	aged := now.Add(-31 * time.Second)
	seedRefreshToken(t, failing.Storage, aged)
	strategy := NewRefreshStrategy(true, 0, 0, 0, clock)

	fresh := func(context.Context) (connector.Identity, error) {
		return connector.Identity{UserID: "u1", Username: "bob"}, nil
	}

	stored, err := failing.GetRefresh(ctx, "r1")
	require.NoError(t, err)
	req := &internal.RefreshToken{RefreshId: "r1", Token: "t1"}

	raw, _, err := rt.Rotate(ctx, &stored, req, strategy, fresh)
	require.Error(t, err, "the injected offline session failure must fail the grant")
	require.Empty(t, raw)

	after, err := failing.GetRefresh(ctx, "r1")
	require.NoError(t, err)
	require.Equal(t, "t1", after.Token, "a failed offline session update must restore the token")
	require.Equal(t, "t0", after.ObsoleteToken, "a failed offline session update must restore the obsolete token")
	require.Equal(t, aged, after.LastUsed, "a failed offline session update must restore last-used")

	// With reuse disabled the retry only succeeds if the claim was undone.
	raw, _, err = rt.Rotate(ctx, &stored, req, strategy, fresh)
	require.NoError(t, err, "retry with the original token must succeed")
	var tok internal.RefreshToken
	require.NoError(t, internal.Unmarshal(raw, &tok))
	require.NotEqual(t, "t1", tok.Token)
}

func TestRefreshStoreRotateOfflineSessionFailureOnReusePathLeavesRow(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	clock := func() time.Time { return now }

	failing := &failingOfflineSessionStorage{Storage: memory.New(slog.New(slog.DiscardHandler))}
	rt := NewRefreshStore(failing, clock, slog.New(slog.DiscardHandler))
	seedRefreshToken(t, failing.Storage, now.Add(-31*time.Second))
	strategy := NewRefreshStrategy(true, 0, 0, 30*time.Second, clock)

	fresh := func(context.Context) (connector.Identity, error) {
		return connector.Identity{UserID: "u1", Username: "bob"}, nil
	}

	stored, err := failing.GetRefresh(ctx, "r1")
	require.NoError(t, err)
	req := &internal.RefreshToken{RefreshId: "r1", Token: "t1"}

	raw, _, err := rt.Rotate(ctx, &stored, req, strategy, fresh)
	require.NoError(t, err)
	var rotated internal.RefreshToken
	require.NoError(t, internal.Unmarshal(raw, &rotated))

	before, err := failing.GetRefresh(ctx, "r1")
	require.NoError(t, err)
	require.Equal(t, rotated.Token, before.Token)
	require.Equal(t, "t1", before.ObsoleteToken)

	// Within the window the obsolete token takes the reuse path, which mints
	// nothing. A failed offline session update there must not roll back the
	// rotated token the client already holds.
	failing.failNow.Store(true)
	failing.updates.Store(0)
	now = now.Add(time.Second)
	_, _, err = rt.Rotate(ctx, &stored, req, strategy, fresh)
	require.Error(t, err, "the injected offline session failure must fail the grant")

	after, err := failing.GetRefresh(ctx, "r1")
	require.NoError(t, err)
	require.Equal(t, before.Token, after.Token, "reuse path must leave the rotated token in place")
	require.Equal(t, before.ObsoleteToken, after.ObsoleteToken, "reuse path must leave the obsolete token in place")
	require.Equal(t, before.LastUsed, after.LastUsed, "reuse path must leave last-used in place")
	require.Equal(t, int32(1), failing.updates.Load(), "reuse path must write the claim only, no undo")
}

func TestRefreshStoreRotateReuseWindowSkipsConnector(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	clock := func() time.Time { return now }

	store := memory.New(slog.New(slog.DiscardHandler))
	rt := NewRefreshStore(store, clock, slog.New(slog.DiscardHandler))
	seedRefreshToken(t, store, now.Add(-31*time.Second))
	strategy := NewRefreshStrategy(true, 0, 0, 30*time.Second, clock)

	var resolves atomic.Int32
	fresh := func(context.Context) (connector.Identity, error) {
		resolves.Add(1)
		return connector.Identity{UserID: "u1", Username: "bob", ConnectorData: []byte("cd")}, nil
	}

	stored, err := store.GetRefresh(ctx, "r1")
	require.NoError(t, err)
	req := &internal.RefreshToken{RefreshId: "r1", Token: "t1"}

	raw, _, err := rt.Rotate(ctx, &stored, req, strategy, fresh)
	require.NoError(t, err)
	var rotated internal.RefreshToken
	require.NoError(t, internal.Unmarshal(raw, &rotated))

	// Within the window the obsolete token redeems the rotated one without
	// consulting the connector, so the retry works even if the user is gone.
	now = now.Add(time.Second)
	retried, _, err := rt.Rotate(ctx, &stored, req, strategy, fresh)
	require.NoError(t, err)
	var redeemed internal.RefreshToken
	require.NoError(t, internal.Unmarshal(retried, &redeemed))
	require.Equal(t, rotated.Token, redeemed.Token, "obsolete token did not redeem the rotated one")
	require.Equal(t, int32(1), resolves.Load())

	// The refreshed identity's connector data lands on the offline session.
	sess, err := store.GetOfflineSessions(ctx, "u1", "mock")
	require.NoError(t, err)
	require.Equal(t, []byte("cd"), sess.ConnectorData)
}

func TestRefreshStoreRotateDisabledRotationRefreshesClaims(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	clock := func() time.Time { return now }

	store := memory.New(slog.New(slog.DiscardHandler))
	rt := NewRefreshStore(store, clock, slog.New(slog.DiscardHandler))
	aged := now.Add(-31 * time.Second)
	seedRefreshToken(t, store, aged)
	strategy := NewRefreshStrategy(false, 0, 0, 30*time.Second, clock)

	resolveErr := errors.New("user deleted")
	var resolves atomic.Int32
	fresh := func(context.Context) (connector.Identity, error) {
		resolves.Add(1)
		if resolveErr != nil {
			return connector.Identity{}, resolveErr
		}
		return connector.Identity{UserID: "u1", Username: "bob"}, nil
	}

	stored, err := store.GetRefresh(ctx, "r1")
	require.NoError(t, err)
	req := &internal.RefreshToken{RefreshId: "r1", Token: "t1"}

	// A failed resolution undoes the in-place claim: last-used is restored and
	// the token row is untouched.
	_, _, err = rt.Rotate(ctx, &stored, req, strategy, fresh)
	require.ErrorIs(t, err, resolveErr)
	after, err := store.GetRefresh(ctx, "r1")
	require.NoError(t, err)
	require.Equal(t, "t1", after.Token)
	require.Equal(t, "t0", after.ObsoleteToken)
	require.Equal(t, aged, after.LastUsed)

	resolveErr = nil
	raw, ident, err := rt.Rotate(ctx, &stored, req, strategy, fresh)
	require.NoError(t, err)
	require.Equal(t, "bob", ident.Username)

	// No rotation: the same token is returned, with refreshed claims.
	var tok internal.RefreshToken
	require.NoError(t, internal.Unmarshal(raw, &tok))
	require.Equal(t, "t1", tok.Token)
	after, err = store.GetRefresh(ctx, "r1")
	require.NoError(t, err)
	require.Equal(t, "t1", after.Token)
	require.Equal(t, "bob", after.Claims.Username)
	require.Equal(t, int32(2), resolves.Load())
}

func TestRefreshStoreUpdateOfflineSessionAfterRevoke(t *testing.T) {
	ctx := context.Background()
	rt, store := newTestStore(t)

	stored := storage.RefreshToken{
		ID: "r1", Token: "t1", ClientID: "client-1", ConnectorID: "mock",
		Claims: storage.Claims{UserID: "u1", Username: "alice"}, CreatedAt: time.Now(), LastUsed: time.Now(),
	}
	require.NoError(t, store.CreateRefresh(ctx, stored))
	require.NoError(t, store.CreateOfflineSessions(ctx, storage.OfflineSessions{
		UserID: "u1", ConnID: "mock",
		Refresh: map[string]*storage.RefreshTokenRef{"client-1": {ID: "r1", ClientID: "client-1"}},
	}))

	// The token is revoked after the caller loaded it, e.g. while a rotation is in flight.
	rt.RevokeClients(ctx, "u1", "mock", []string{"client-1"})

	var err error
	require.NotPanics(t, func() {
		err = rt.updateOfflineSession(ctx, &stored, connector.Identity{UserID: "u1"}, time.Now())
	})
	require.Error(t, err)
}

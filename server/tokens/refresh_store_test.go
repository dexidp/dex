package tokens

import (
	"context"
	"log/slog"
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

	rt.Revoke(ctx, "u1", "mock")

	_, err := store.GetRefresh(ctx, "r1")
	require.ErrorIs(t, err, storage.ErrNotFound)
	_, err = store.GetRefresh(ctx, "r2")
	require.ErrorIs(t, err, storage.ErrNotFound)

	// The offline session is kept; its refs are cleared, connector data preserved.
	sess, err := store.GetOfflineSessions(ctx, "u1", "mock")
	require.NoError(t, err)
	require.Empty(t, sess.Refresh)
	require.Equal(t, []byte("cd"), sess.ConnectorData)
}

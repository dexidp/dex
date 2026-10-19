package server

// Refresh-token grants and the rotation transaction contract.
//
// Rotate claims, resolves and finalizes a rotation with pure storage updaters:
// identity resolution — which may read storage (the local password DB's Refresh
// calls GetPassword) or contact an upstream — runs between the transactions.
// Backends that serialize on a single connection or lock (SQLite, memory)
// deadlocked when resolution ran inside the transaction, wedging /token
// permanently while /healthz kept passing (#5017, #4991).
//
// These tests pin both sides: no storage read is issued from inside a rotation
// updater, and connectors are contacted at most once per rotation.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/connector/mock"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
	dexsql "github.com/dexidp/dex/storage/sql"
)

// reentrancyGuard fails the storage reads the refresh grant can issue from
// inside the UpdateRefreshToken updater. A deadlock on SQLite or memory is a
// hang; the guard turns the same code path into an immediate error, so a
// regression fails the test instead of timing out the package.
//
// It is driven synchronously by a single request and is not safe for
// concurrent use. Reads other than the two the grant reaches from the updater
// pass through unchecked.
type reentrancyGuard struct {
	storage.Storage
	inUpdate bool
}

func (g *reentrancyGuard) UpdateRefreshToken(ctx context.Context, id string, updater func(r storage.RefreshToken) (storage.RefreshToken, error)) error {
	g.inUpdate = true
	defer func() { g.inUpdate = false }()
	return g.Storage.UpdateRefreshToken(ctx, id, updater)
}

func (g *reentrancyGuard) GetPassword(ctx context.Context, email string) (storage.Password, error) {
	if g.inUpdate {
		return storage.Password{}, fmt.Errorf("GetPassword: re-entrant read from inside the UpdateRefreshToken updater")
	}
	return g.Storage.GetPassword(ctx, email)
}

func (g *reentrancyGuard) GetOfflineSessions(ctx context.Context, userID string, connID string) (storage.OfflineSessions, error) {
	if g.inUpdate {
		return storage.OfflineSessions{}, fmt.Errorf("GetOfflineSessions: re-entrant read from inside the UpdateRefreshToken updater")
	}
	return g.Storage.GetOfflineSessions(ctx, userID, connID)
}

// seedLocalClientAndConnector creates a confidential client and opens the
// local password connector.
func seedLocalClientAndConnector(t *testing.T, s *Server) {
	t.Helper()
	ctx := t.Context()

	err := s.storage.CreateClient(ctx, storage.Client{
		ID:           "test",
		Secret:       "barfoo",
		RedirectURIs: []string{"foo://bar.com/", "https://auth.example.com"},
		Name:         "dex client",
	})
	require.NoError(t, err)

	localConn := storage.Connector{
		ID:              "local",
		Type:            connectors.LocalConnector,
		Name:            "Email",
		ResourceVersion: "1",
	}
	require.NoError(t, s.storage.CreateConnector(ctx, localConn))
	_, err = s.connectors.Open(localConn)
	require.NoError(t, err)
}

// seedLocalPasswordDB seeds the client, the local connector and one
// storage-backed password user.
func seedLocalPasswordDB(t *testing.T, s *Server) {
	t.Helper()
	seedLocalClientAndConnector(t, s)
	seedLocalPasswordUser(t, s)
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
}

// doPasswordGrant exchanges the seeded user's credentials for a refresh token.
func doPasswordGrant(t *testing.T, s *Server) string {
	t.Helper()

	u, err := url.Parse(s.issuerURL.String())
	require.NoError(t, err)
	u.Path = path.Join(u.Path, "/token")

	v := url.Values{}
	v.Add("scope", "openid profile email offline_access")
	v.Add("grant_type", "password")
	v.Add("username", "user@example.com")
	v.Add("password", "secret")

	req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(v.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("test", "barfoo")

	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, "password grant failed: %s", rr.Body.String())

	var resp tokenResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.RefreshToken, "no refresh token in password grant response")
	return resp.RefreshToken
}

// refreshGrantRaw redeems a refresh token and reports the response recorder
// without asserting the outcome. A non-zero timeout bounds the request both
// ways: the request context, so a deadlock surfaces as a context error where
// database/sql honors it, and a watchdog around the call itself, because a
// wedged single-connection pool can block past any context. Either way a
// regression fails in bounded time instead of hanging the package.
func refreshGrantRaw(t *testing.T, s *Server, refreshToken string, timeout time.Duration) *httptest.ResponseRecorder {
	t.Helper()

	u, err := url.Parse(s.issuerURL.String())
	require.NoError(t, err)
	u.Path = path.Join(u.Path, "/token")

	v := url.Values{}
	v.Add("grant_type", "refresh_token")
	v.Add("refresh_token", refreshToken)

	req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(v.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("test", "barfoo")
	if timeout > 0 {
		ctx, cancel := context.WithTimeout(req.Context(), timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

	rr := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		s.ServeHTTP(rr, req)
		close(done)
	}()
	if timeout > 0 {
		select {
		case <-done:
		case <-time.After(timeout + 5*time.Second):
			t.Fatalf("refresh grant did not complete within %s: the rotation transaction is deadlocked on its own storage read", timeout)
		}
	} else {
		<-done
	}
	return rr
}

// doRefreshGrant redeems a refresh token and asserts it succeeds.
func doRefreshGrant(t *testing.T, s *Server, refreshToken string, timeout time.Duration) tokenResponse {
	t.Helper()
	rr := refreshGrantRaw(t, s, refreshToken, timeout)
	require.Equal(t, http.StatusOK, rr.Code, "refresh grant failed: %s", rr.Body.String())

	var resp tokenResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.RefreshToken, "no rotated refresh token in response")
	return resp
}

// TestRefreshGrantStorageBackedPasswordDB refreshes a token issued to a user
// created through the password store (gRPC CreatePassword / SCIM provisioning).
// The guard fails any storage read from inside the rotation transaction, so
// this passes only when the password DB's identity is resolved before it.
func TestRefreshGrantStorageBackedPasswordDB(t *testing.T) {
	httpServer, s := newTestServer(t, func(c *Config) {
		c.PasswordConnector = "local"
		c.Storage = &reentrancyGuard{Storage: memory.New(newLogger(t))}
	})
	defer httpServer.Close()
	seedLocalPasswordDB(t, s)

	first := doPasswordGrant(t, s)
	second := doRefreshGrant(t, s, first, 0)
	require.NotEqual(t, first, second.RefreshToken, "refresh token was not rotated")

	// The rotated token refreshes too: not a one-off surviving the transaction.
	third := doRefreshGrant(t, s, second.RefreshToken, 0)
	require.NotEqual(t, second.RefreshToken, third.RefreshToken, "chained refresh did not rotate")
}

// TestRefreshGrantStorageBackedPasswordDBSQLite repeats the flow on a real
// SQLite backend, the deployment shape the deadlock was reported on. The
// request context is bounded: the pre-fix nested GetPassword waits on the
// single-connection pool until the context expires, so a regression fails in
// bounded time instead of hanging the package.
func TestRefreshGrantStorageBackedPasswordDBSQLite(t *testing.T) {
	sqlite, err := (&dexsql.SQLite3{File: t.TempDir() + "/dex.db"}).Open(newLogger(t))
	require.NoError(t, err)

	httpServer, s := newTestServer(t, func(c *Config) {
		c.PasswordConnector = "local"
		c.Storage = sqlite
	})
	defer httpServer.Close()
	seedLocalPasswordDB(t, s)

	first := doPasswordGrant(t, s)
	second := doRefreshGrant(t, s, first, 10*time.Second)
	require.NotEqual(t, first, second.RefreshToken, "refresh token was not rotated")
}

// TestRefreshGrantStaticPasswordUser refreshes a token for a user defined in
// staticPasswords. The static wrapper serves such users without touching the
// backing storage, so this pins the originally reported path (the connector
// data read) rather than the password store one.
func TestRefreshGrantStaticPasswordUser(t *testing.T) {
	httpServer, s := newTestServer(t, func(c *Config) {
		c.PasswordConnector = "local"
		c.Storage = &reentrancyGuard{
			Storage: storage.WithStaticPasswords(memory.New(newLogger(t)), staticTestPassword(t), newLogger(t)),
		}
	})
	defer httpServer.Close()
	seedLocalClientAndConnector(t, s)

	first := doPasswordGrant(t, s)
	second := doRefreshGrant(t, s, first, 0)
	require.NotEqual(t, first, second.RefreshToken, "refresh token was not rotated")
}

func staticTestPassword(t *testing.T) []storage.Password {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)
	return []storage.Password{{
		Email:    "user@example.com",
		Username: "user-login",
		Name:     "User Full Name",
		UserID:   "user-id",
		Groups:   []string{"team-a"},
		Hash:     hash,
	}}
}

// TestRefreshGrantPasswordDBReuseWindow exercises the rotation reuse window
// with a password-store user. A re-presented obsolete token must redeem the
// token it was rotated into, and — exactly as before the fix — must not fail
// on a user deleted between the rotation and the retry: the reuse path never
// consults the connector.
func TestRefreshGrantPasswordDBReuseWindow(t *testing.T) {
	const reuseWindow = 30 * time.Second

	now := time.Now().UTC()
	clock := func() time.Time { return now }

	httpServer, s := newTestServer(t, func(c *Config) {
		c.PasswordConnector = "local"
		c.Storage = &reentrancyGuard{Storage: memory.New(newLogger(t))}
		c.Now = clock
		c.RefreshTokenPolicy = tokens.NewRefreshStrategy(true, 0, 0, reuseWindow, clock)
	})
	defer httpServer.Close()
	seedLocalPasswordDB(t, s)

	first := doPasswordGrant(t, s)

	// Advance past the reuse window so the first refresh rotates.
	now = now.Add(reuseWindow + time.Second)
	second := doRefreshGrant(t, s, first, 0)
	require.NotEqual(t, first, second.RefreshToken, "refresh token was not rotated")

	// The obsolete token, re-presented within the window, redeems the rotated
	// token instead of being rejected.
	now = now.Add(time.Second)
	retried := doRefreshGrant(t, s, first, 0)
	require.Equal(t, second.RefreshToken, retried.RefreshToken, "obsolete token did not redeem the rotated one")

	// A user deleted between the rotation and an in-window retry keeps the
	// retry working: the reuse path must not consult the password store.
	require.NoError(t, s.storage.DeletePassword(t.Context(), "user@example.com"))
	now = now.Add(time.Second)
	retried = doRefreshGrant(t, s, first, 0)
	require.Equal(t, second.RefreshToken, retried.RefreshToken, "in-window retry failed after the user was deleted")
}

// TestRefreshGrantResolveFailureDeniesUntilResolved pins the failure semantics
// of the two-step rotation: a resolution that fails undoes the claim — the
// grant errors, no reuse window opens (an immediate retry fails again), and
// once the user resolves (here: is re-created) the next refresh rotates.
func TestRefreshGrantResolveFailureDeniesUntilResolved(t *testing.T) {
	const reuseWindow = 30 * time.Second

	now := time.Now().UTC()
	clock := func() time.Time { return now }

	httpServer, s := newTestServer(t, func(c *Config) {
		c.PasswordConnector = "local"
		c.Storage = &reentrancyGuard{Storage: memory.New(newLogger(t))}
		c.Now = clock
		c.RefreshTokenPolicy = tokens.NewRefreshStrategy(true, 0, 0, reuseWindow, clock)
	})
	defer httpServer.Close()
	seedLocalPasswordDB(t, s)

	first := doPasswordGrant(t, s)

	// Age the token past its window and delete the user: the rotation is
	// claimed, resolution cannot find the user, and the grant errors — with
	// the claim undone, so an immediate retry (still nominally within the
	// window) fails again instead of minting from stale claims.
	require.NoError(t, s.storage.DeletePassword(t.Context(), "user@example.com"))
	now = now.Add(reuseWindow + time.Second)
	require.Equal(t, http.StatusInternalServerError, refreshGrantStatus(t, s, first, 10*time.Second))

	now = now.Add(time.Second)
	require.Equal(t, http.StatusInternalServerError, refreshGrantStatus(t, s, first, 10*time.Second), "in-window retry after a failed resolution must not succeed from stale claims")

	// With the user back, the next refresh performs the rotation.
	seedLocalPasswordUser(t, s)
	rotated := doRefreshGrant(t, s, first, 0)
	require.NotEqual(t, first, rotated.RefreshToken, "rotation did not retry after the failure")
}

// TestRefreshGrantFinalizeFailureUndoesClaim pins the other failure window: a
// finalize that fails after a successful resolution also undoes the claim, so
// the immediate retry re-rotates instead of reusing the un-rotated token with
// stale claims.
func TestRefreshGrantFinalizeFailureUndoesClaim(t *testing.T) {
	const reuseWindow = 30 * time.Second

	now := time.Now().UTC()
	clock := func() time.Time { return now }

	failing := &finalizingStorage{Storage: memory.New(newLogger(t))}
	httpServer, s := newTestServer(t, func(c *Config) {
		c.PasswordConnector = "local"
		c.Storage = failing
		c.Now = clock
		c.RefreshTokenPolicy = tokens.NewRefreshStrategy(true, 0, 0, reuseWindow, clock)
	})
	defer httpServer.Close()
	seedLocalPasswordDB(t, s)

	first := doPasswordGrant(t, s)

	now = now.Add(reuseWindow + time.Second)
	failing.failFinalize.Store(true)
	require.Equal(t, http.StatusInternalServerError, refreshGrantStatus(t, s, first, 10*time.Second), "refresh must fail when the finalize transaction fails")

	// The retry (within the window) must rotate: with the claim undone it
	// re-enters the mint path rather than reusing the un-rotated token.
	now = now.Add(time.Second)
	retried := doRefreshGrant(t, s, first, 0)
	require.NotEqual(t, first, retried.RefreshToken, "in-window retry reused the un-rotated token: the finalize failure did not undo the claim")
}

// finalizingStorage fails exactly one finalize transaction, one-shot: the
// second UpdateRefreshToken after a reset fails once, later ones (including the
// claim-undo it triggers) succeed.
type finalizingStorage struct {
	storage.Storage
	failFinalize atomic.Bool
	updates      atomic.Int32
}

func (f *finalizingStorage) UpdateRefreshToken(ctx context.Context, id string, updater func(r storage.RefreshToken) (storage.RefreshToken, error)) error {
	if f.updates.Add(1) == 2 && f.failFinalize.CompareAndSwap(true, false) {
		return errors.New("injected finalize failure")
	}
	return f.Storage.UpdateRefreshToken(ctx, id, updater)
}

// refreshGrantStatus performs a refresh grant and reports only its HTTP status,
// bounded by a watchdog like doRefreshGrant.
func refreshGrantStatus(t *testing.T, s *Server, refreshToken string, timeout time.Duration) int {
	t.Helper()
	return refreshGrantRaw(t, s, refreshToken, timeout).Code
}

// seedLocalPasswordUser re-creates the password user removed by a deletion
// step, for flows that restore the user.
func seedLocalPasswordUser(t *testing.T, s *Server) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)
	require.NoError(t, s.storage.CreatePassword(t.Context(), storage.Password{
		Email:         "user@example.com",
		Username:      "user-login",
		Name:          "User Full Name",
		EmailVerified: boolPtr(false),
		UserID:        "user-id",
		Groups:        []string{"team-a"},
		Hash:          hash,
	}))
}

// TestRefreshGrantUpstreamConnectorRefreshedAtMostOnce pins both sides of the
// contract for an upstream-facing connector: its Refresh runs outside the
// rotation transactions (it reads storage itself — the guard fails any such
// read issued from inside an updater), and at most once per rotation — across
// a rotation and an in-window retry of the obsolete token, exactly once,
// because the retry reuses the already-rotated token without contacting it
// again. The offline session must also carry the connector data the refreshed
// identity returned.
func TestRefreshGrantUpstreamConnectorRefreshedAtMostOnce(t *testing.T) {
	const reuseWindow = 30 * time.Second

	now := time.Now().UTC()
	clock := func() time.Time { return now }

	guard := &reentrancyGuard{Storage: memory.New(newLogger(t))}
	httpServer, s := newTestServer(t, func(c *Config) {
		c.Storage = guard
		c.Now = clock
		c.RefreshTokenPolicy = tokens.NewRefreshStrategy(true, 0, 0, reuseWindow, clock)
	})
	defer httpServer.Close()
	mockRefreshTokenTestStorage(t, s.storage, false)

	counting := &countingRefreshConnector{Connector: mock.NewCallbackConnector(newLogger(t)), store: guard}
	s.connectors.Set("test", connectors.Connector{
		Type: "mockCallback",
		// The stored connector row has no resource version; a mismatching one
		// makes Cache.Get re-resolve the connector and drop this injection.
		ResourceVersion: "",
		Connector:       counting,
	})

	tokenData, err := internal.Marshal(&internal.RefreshToken{RefreshId: "test", Token: "bar"})
	require.NoError(t, err)

	now = now.Add(reuseWindow + time.Second)
	rotated := doRefreshGrant(t, s, tokenData, 0)
	require.NotEmpty(t, rotated.RefreshToken)

	now = now.Add(time.Second)
	retried := doRefreshGrant(t, s, tokenData, 0)
	require.Equal(t, rotated.RefreshToken, retried.RefreshToken)

	require.Equal(t, int32(1), counting.refreshCalls.Load(), "upstream connector Refresh must be called exactly once")

	// The refreshed identity's connector data lands on the offline session.
	sess, err := s.storage.GetOfflineSessions(t.Context(), "1", "test")
	require.NoError(t, err)
	require.Equal(t, []byte("foobar"), sess.ConnectorData, "offline session did not carry the refreshed connector data")
}

// TestRefreshGrantConcurrentRequestsResolveOnce pins the concurrency property
// the claim step provides: two truly concurrent refreshes of the same token
// both succeed, both receive the same rotated token, and the connector is
// resolved exactly once. Uses plain memory storage — the reentrancyGuard is
// not safe for concurrent use.
func TestRefreshGrantConcurrentRequestsResolveOnce(t *testing.T) {
	const reuseWindow = 30 * time.Second

	now := time.Now().UTC()
	clock := func() time.Time { return now }

	httpServer, s := newTestServer(t, func(c *Config) {
		c.Storage = memory.New(newLogger(t))
		c.Now = clock
		c.RefreshTokenPolicy = tokens.NewRefreshStrategy(true, 0, 0, reuseWindow, clock)
	})
	defer httpServer.Close()
	mockRefreshTokenTestStorage(t, s.storage, false)

	counting := &countingRefreshConnector{Connector: mock.NewCallbackConnector(newLogger(t))}
	s.connectors.Set("test", connectors.Connector{
		Type:            "mockCallback",
		ResourceVersion: "",
		Connector:       counting,
	})

	tokenData, err := internal.Marshal(&internal.RefreshToken{RefreshId: "test", Token: "bar"})
	require.NoError(t, err)

	now = now.Add(reuseWindow + time.Second)

	results := make([]tokenResponse, 2)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = doRefreshGrant(t, s, tokenData, 0)
		}(i)
	}
	wg.Wait()

	// One request rotates; the concurrent one either reuses the still-current
	// old token (it ran during the resolve gap) or redeems the rotated one (it
	// ran after the finalize). Both outcomes are valid, and redeeming either
	// within the window converges to the current token.
	converged := make([]string, 2)
	for i, tok := range []string{results[0].RefreshToken, results[1].RefreshToken} {
		converged[i] = doRefreshGrant(t, s, tok, 0).RefreshToken
	}
	require.Equal(t, converged[0], converged[1], "redeeming either concurrently issued token must converge to the current one")

	require.Equal(t, int32(1), counting.refreshCalls.Load(), "concurrent refreshes must resolve the connector exactly once")
}

// countingRefreshConnector counts Refresh calls on a wrapped connector. A
// non-nil store makes Refresh issue a storage read itself, so tests can prove
// the call happens outside any rotation updater.
type countingRefreshConnector struct {
	connector.Connector
	refreshCalls atomic.Int32
	store        storage.Storage
}

func (c *countingRefreshConnector) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	c.refreshCalls.Add(1)
	if c.store != nil {
		if _, err := c.store.GetPassword(ctx, "probe@example.com"); err != nil && !errors.Is(err, storage.ErrNotFound) {
			return identity, err
		}
	}
	refreshConn, ok := c.Connector.(connector.RefreshConnector)
	if !ok {
		return identity, nil
	}
	return refreshConn.Refresh(ctx, s, identity)
}

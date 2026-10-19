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
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
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

// seedLocalPasswordDB creates a confidential client, opens the local password
// connector and creates one storage-backed password user.
func seedLocalPasswordDB(t *testing.T, s *Server) {
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

	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)
	err = s.storage.CreatePassword(ctx, storage.Password{
		Email:         "user@example.com",
		Username:      "user-login",
		Name:          "User Full Name",
		EmailVerified: boolPtr(false),
		UserID:        "user-id",
		Groups:        []string{"team-a"},
		Hash:          hash,
	})
	require.NoError(t, err)
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

// doRefreshGrant redeems a refresh token. A non-zero timeout bounds the
// request both ways: the request context, so a deadlock surfaces as a context
// error where database/sql honors it, and a watchdog around the call itself,
// because a wedged single-connection pool can block past any context. Either
// way a regression fails in bounded time instead of hanging the package.
func doRefreshGrant(t *testing.T, s *Server, refreshToken string, timeout time.Duration) tokenResponse {
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

	ctx := t.Context()
	require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
		ID:           "test",
		Secret:       "barfoo",
		RedirectURIs: []string{"foo://bar.com/", "https://auth.example.com"},
		Name:         "dex client",
	}))
	localConn := storage.Connector{
		ID:              "local",
		Type:            connectors.LocalConnector,
		Name:            "Email",
		ResourceVersion: "1",
	}
	require.NoError(t, s.storage.CreateConnector(ctx, localConn))
	_, err := s.connectors.Open(localConn)
	require.NoError(t, err)

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

// TestRefreshGrantResolveFailureKeepsTokenUsable pins the failure semantics of
// the two-step rotation: a resolution that fails after the claim errors the
// grant but leaves the token claimable within its reuse window, and the next
// refresh past the window retries the rotation and succeeds.
func TestRefreshGrantResolveFailureKeepsTokenUsable(t *testing.T) {
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
	// claimed, resolution cannot find the user, and the grant errors.
	require.NoError(t, s.storage.DeletePassword(t.Context(), "user@example.com"))
	now = now.Add(reuseWindow + time.Second)
	require.Equal(t, http.StatusInternalServerError, refreshGrantStatus(t, s, first, 10*time.Second))

	// The claimed token still redeems within its window — same token back, no
	// resolution, no mint.
	now = now.Add(time.Second)
	redeemed := doRefreshGrant(t, s, first, 0)
	require.Equal(t, first, redeemed.RefreshToken, "claimed token did not redeem itself within the reuse window")

	// Past the window, with the user back, the rotation retries and mints.
	seedLocalPasswordUser(t, s)
	now = now.Add(reuseWindow + time.Second)
	rotated := doRefreshGrant(t, s, first, 0)
	require.NotEqual(t, first, rotated.RefreshToken, "rotation did not retry past the reuse window")
}

// refreshGrantStatus performs a refresh grant and reports only its HTTP status,
// bounded by a watchdog like doRefreshGrant.
func refreshGrantStatus(t *testing.T, s *Server, refreshToken string, timeout time.Duration) int {
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

	rr := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		s.ServeHTTP(rr, req)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout + 5*time.Second):
		t.Fatalf("refresh grant did not complete within %s: the rotation transaction is deadlocked on its own storage read", timeout)
	}
	return rr.Code
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
// rotation transactions (the guard fails any storage read issued from inside
// one), and at most once per rotation — across a rotation and an in-window
// retry of the obsolete token, exactly once, because the retry reuses the
// already-rotated token without contacting it again.
func TestRefreshGrantUpstreamConnectorRefreshedAtMostOnce(t *testing.T) {
	const reuseWindow = 30 * time.Second

	now := time.Now().UTC()
	clock := func() time.Time { return now }

	httpServer, s := newTestServer(t, func(c *Config) {
		c.Storage = &reentrancyGuard{Storage: memory.New(newLogger(t))}
		c.Now = clock
		c.RefreshTokenPolicy = tokens.NewRefreshStrategy(true, 0, 0, reuseWindow, clock)
	})
	defer httpServer.Close()
	mockRefreshTokenTestStorage(t, s.storage, false)

	counting := &countingRefreshConnector{Connector: mock.NewCallbackConnector(newLogger(t))}
	s.connectors.Set("test", connectors.Connector{
		Type: "mockCallback",
		// The stored connector row has no resource version; a mismatching one
		// makes Cache.Get re-resolve the connector and drop this injection.
		ResourceVersion: "",
		Connector:       counting,
	})

	u, err := url.Parse(s.issuerURL.String())
	require.NoError(t, err)
	u.Path = path.Join(u.Path, "/token")

	refresh := func(token string) tokenResponse {
		v := url.Values{}
		v.Add("grant_type", "refresh_token")
		v.Add("refresh_token", token)

		req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(v.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("test", "barfoo")

		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code, "refresh grant failed: %s", rr.Body.String())

		var resp tokenResponse
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
		return resp
	}

	tokenData, err := internal.Marshal(&internal.RefreshToken{RefreshId: "test", Token: "bar"})
	require.NoError(t, err)

	now = now.Add(reuseWindow + time.Second)
	rotated := refresh(tokenData)
	require.NotEmpty(t, rotated.RefreshToken)

	now = now.Add(time.Second)
	retried := refresh(tokenData)
	require.Equal(t, rotated.RefreshToken, retried.RefreshToken)

	require.Equal(t, int32(1), counting.refreshCalls.Load(), "upstream connector Refresh must be called exactly once")
}

// countingRefreshConnector counts Refresh calls on a wrapped connector.
type countingRefreshConnector struct {
	connector.Connector
	refreshCalls atomic.Int32
}

func (c *countingRefreshConnector) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	c.refreshCalls.Add(1)
	refreshConn, ok := c.Connector.(connector.RefreshConnector)
	if !ok {
		return identity, nil
	}
	return refreshConn.Refresh(ctx, s, identity)
}

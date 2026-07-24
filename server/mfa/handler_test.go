package mfa

import (
	"crypto/rand"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/connector/mock"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
	dexweb "github.com/dexidp/dex/web"
)

// testMux adapts a gorilla router to router.Mux so the Handler can mount its
// routes.
type testMux struct{ r *mux.Router }

func (m testMux) Handle(p string, h http.Handler)         { m.r.Handle(p, h) }
func (m testMux) HandleFunc(p string, h http.HandlerFunc) { m.r.HandleFunc(p, h) }
func (m testMux) HandleCORS(p string, h http.HandlerFunc) { m.r.HandleFunc(p, h) }
func (m testMux) HandlePrefix(p string, h http.Handler) {
	m.r.PathPrefix(p).Handler(http.StripPrefix(p, h))
}

func resolveTestConnector(storage.Connector) (connector.Connector, error) {
	return mock.NewCallbackConnector(nil), nil
}

// newTestHandler builds an MFA Handler mounted on a router, wired to an in-memory
// store with a single "mock" connector, the way the server wires it. It returns
// the handler (a few tests call its methods directly), the router serving its
// routes, and the store.
func newTestHandler(t *testing.T, providers map[string]Provider, defaultChain []string) (*Handler, http.Handler, storage.Storage) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug}))
	store := memory.New(logger)

	//nolint:dogsled // only the templates are needed here
	_, _, _, tmpls, err := templates.LoadWebConfig(templates.Config{
		WebFS:     dexweb.FS(),
		IssuerURL: "http://127.0.0.1",
	})
	require.NoError(t, err)

	issuerURL, err := url.Parse("http://127.0.0.1")
	require.NoError(t, err)

	conns := connectors.NewCache(store, resolveTestConnector)
	require.NoError(t, store.CreateConnector(t.Context(), storage.Connector{
		ID: "mock", Type: "mockCallback", Name: "Mock", ResourceVersion: "1",
	}))
	conns.Set("mock", connectors.Connector{Type: "mockCallback", ResourceVersion: "1"})

	h := &Handler{
		Storage:         store,
		Templates:       tmpls,
		Logger:          logger,
		IssuerURL:       oauth2.IssuerURL{URL: *issuerURL},
		MFAProviders:    providers,
		DefaultMFAChain: defaultChain,
		Now:             time.Now,
		Connectors:      conns,
	}

	router := mux.NewRouter()
	h.Mount(testMux{router})
	return h, router, store
}

func randomHMACKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return key
}

func TestCompleteStep(t *testing.T) {
	provider, err := NewWebAuthnProvider("Test", "", nil, "", "", "http://127.0.0.1", nil)
	require.NoError(t, err)

	h, _, store := newTestHandler(t,
		map[string]Provider{"webauthn-1": provider, "webauthn-2": provider},
		[]string{"webauthn-1", "webauthn-2"},
	)

	ctx := t.Context()
	authReq := storage.AuthRequest{
		ID:          "test-req-chain",
		ClientID:    "example-app",
		Expiry:      time.Now().Add(time.Hour),
		HMACKey:     randomHMACKey(t),
		LoggedIn:    true,
		Claims:      storage.Claims{UserID: "user-1", Email: "user@example.com"},
		ConnectorID: "mock",
	}
	require.NoError(t, store.CreateAuthRequest(ctx, authReq))
	require.NoError(t, store.CreateClient(ctx, storage.Client{ID: "example-app", Secret: "secret"}))

	// Completing first step should redirect to second.
	redirectURL, err := h.CompleteStep(ctx, authReq, "webauthn-1")
	require.NoError(t, err)
	require.Contains(t, redirectURL, "/mfa/webauthn")
	require.Contains(t, redirectURL, "authenticator=webauthn-2")

	// Completing the last step should redirect to the flow dispatcher.
	redirectURL, err = h.CompleteStep(ctx, authReq, "webauthn-2")
	require.NoError(t, err)
	require.Contains(t, redirectURL, "/auth?")

	// Verify MFAValidated was set.
	updated, err := store.GetAuthRequest(ctx, authReq.ID)
	require.NoError(t, err)
	require.True(t, updated.MFAValidated)
}

// TestMFAEntry covers the /mfa entry the dispatcher hands off to: MFA resolves
// the effective chain itself, sending an applicable factor to its page, and —
// crucially — recording MFA as satisfied when nothing applies so the dispatcher
// (which gates on the unfiltered client chain) does not loop back here.
func TestMFAEntry(t *testing.T) {
	ctx := t.Context()

	setup := func(t *testing.T, connectorTypes []string) (http.Handler, storage.Storage, storage.AuthRequest, []byte) {
		t.Helper()
		_, router, store := newTestHandler(t,
			map[string]Provider{"totp": NewTOTPProvider("test-issuer", connectorTypes)},
			nil,
		)

		hmacKey := randomHMACKey(t)
		authReq := storage.AuthRequest{
			ID: "mfa-entry-req", ClientID: "app", Expiry: time.Now().Add(time.Hour),
			HMACKey: hmacKey, LoggedIn: true,
			Claims:      storage.Claims{UserID: "user-1", Email: "user@example.com"},
			ConnectorID: "mock",
		}
		require.NoError(t, store.CreateAuthRequest(ctx, authReq))
		require.NoError(t, store.CreateClient(ctx, storage.Client{ID: "app", Secret: "s", MFAChain: []string{"totp"}}))
		return router, store, authReq, hmacKey
	}

	get := func(router http.Handler, authReq storage.AuthRequest, hmacKey []byte) *httptest.ResponseRecorder {
		hmacVal := internal.ComputeHMAC(hmacKey, authReq.ID, "mfa")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/mfa?req="+authReq.ID+"&hmac="+hmacVal, nil))
		return w
	}

	t.Run("applicable factor redirects to its page", func(t *testing.T) {
		// nil connector types => the provider applies to every connector.
		router, _, authReq, hmacKey := setup(t, nil)
		w := get(router, authReq, hmacKey)

		require.Equal(t, http.StatusSeeOther, w.Code)
		require.Contains(t, w.Header().Get("Location"), "/mfa/totp")
		require.Contains(t, w.Header().Get("Location"), "authenticator=totp")
	})

	t.Run("no applicable factor marks validated and continues", func(t *testing.T) {
		// The provider is enabled only for oidc; the connector is not, so the
		// resolved chain is empty.
		router, store, authReq, hmacKey := setup(t, []string{"oidc"})
		w := get(router, authReq, hmacKey)

		require.Equal(t, http.StatusSeeOther, w.Code)
		require.Contains(t, w.Header().Get("Location"), "/auth?")

		updated, err := store.GetAuthRequest(ctx, authReq.ID)
		require.NoError(t, err)
		require.True(t, updated.MFAValidated, "MFA must be marked satisfied so the dispatcher does not loop back to /mfa")
	})
}

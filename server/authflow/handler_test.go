package authflow

import (
	"crypto/rand"
	"crypto/rsa"
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
	"github.com/dexidp/dex/server/consent"
	"github.com/dexidp/dex/server/logout"
	"github.com/dexidp/dex/server/mfa"
	"github.com/dexidp/dex/server/session"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
	dexweb "github.com/dexidp/dex/web"
)

func newLogger(t *testing.T) *slog.Logger {
	return slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// testResolveConnector is the connector resolver used by the flow's unit tests.
// They set connectors in the cache directly; the mock callback connector covers
// the few paths that open one.
func testResolveConnector(conn storage.Connector) (connector.Connector, error) {
	return mock.NewCallbackConnector(nil), nil
}

// testMux adapts a gorilla router to router.Mux so a Handler can mount its
// routes (the handlers read path variables with mux.Vars).
type testMux struct{ r *mux.Router }

func (m testMux) Handle(p string, h http.Handler)         { m.r.Handle(p, h) }
func (m testMux) HandleFunc(p string, h http.HandlerFunc) { m.r.HandleFunc(p, h) }
func (m testMux) HandleCORS(p string, h http.HandlerFunc) { m.r.HandleFunc(p, h) }
func (m testMux) HandlePrefix(p string, h http.Handler) {
	m.r.PathPrefix(p).Handler(http.StripPrefix(p, h))
}

// testServer wraps a Handler with the router it is mounted on so tests can both
// call flow methods directly (promoted from the embedded Handler) and drive it
// over HTTP via ServeHTTP.
type testServer struct {
	*Handler
	mux http.Handler
	// mfa is the step handler the server mounts separately; a few tests drive it
	// directly. The dispatcher holds no such reference in production.
	mfa *mfa.Handler
}

func (ts *testServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ts.mux.ServeHTTP(w, r)
}

// testFlowConfig bundles the login Config with the raw inputs the server uses to
// build the shared flow components, so a test can tweak either before assembly.
type testFlowConfig struct {
	Handler
	SessionConfig   *session.Config
	MFAProviders    map[string]mfa.Provider
	DefaultMFAChain []string
	SkipApproval    bool
}

// newTestHandler builds the login flow and its shared components wired to an
// httptest server, assembling them exactly as the server package does.
// updateConfig may tweak the config before the components are built.
func newTestHandler(t *testing.T, updateConfig func(c *testFlowConfig)) (*httptest.Server, *testServer) {
	t.Helper()
	logger := newLogger(t)
	ctx := t.Context()

	sig, err := signer.NewMockSigner(testKey)
	require.NoError(t, err)

	store := memory.New(logger)

	var handler http.Handler
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)

	issuerURL, err := url.Parse(srv.URL)
	require.NoError(t, err)

	//nolint:dogsled // only the templates are needed here
	_, _, _, tmpls, err := templates.LoadWebConfig(templates.Config{
		WebFS:     dexweb.FS(),
		IssuerURL: srv.URL,
	})
	require.NoError(t, err)

	now := func() time.Time { return time.Now() }
	conns := connectors.NewCache(store, testResolveConnector)
	issuer := tokens.NewIssuer(store, sig, *issuerURL, 24*time.Hour, now, logger)

	tc := testFlowConfig{
		Handler: Handler{
			IssuerURL:              *issuerURL,
			Connectors:             conns,
			Storage:                store,
			Templates:              tmpls,
			Signer:                 sig,
			Now:                    now,
			Logger:                 logger,
			SupportedResponseTypes: map[string]bool{"code": true, "token": true, "id_token": true},
			PKCE:                   PKCEConfig{CodeChallengeMethodsSupported: []string{"S256", "plain"}},
			AuthRequestsValidFor:   24 * time.Hour,
		},
		SkipApproval: true,
	}
	if updateConfig != nil {
		updateConfig(&tc)
	}

	// Assemble the flow the same way the server does: shared infrastructure plus
	// independent step handlers that hand off by redirect.
	sessions := &session.Manager{Storage: store, Config: tc.SessionConfig, Now: now, Logger: logger, IssuerURL: *issuerURL}
	mfaManager := &mfa.Handler{IssuerURL: *issuerURL, Storage: store, Templates: tmpls, Logger: logger, MFAProviders: tc.MFAProviders, DefaultMFAChain: tc.DefaultMFAChain, Now: now, Connectors: conns}
	consentManager := &consent.Handler{IssuerURL: *issuerURL, Storage: store, Templates: tmpls, Logger: logger, Sessions: sessions, SkipApproval: tc.SkipApproval}
	logoutManager := &logout.Handler{Storage: store, Templates: tmpls, Logger: logger, Sessions: sessions, Connectors: conns, Issuer: issuer, Signer: sig, IssuerURL: *issuerURL}

	tc.Sessions = sessions
	tc.Issuer = issuer
	tc.Handler.MFAEnabled = len(tc.MFAProviders) > 0
	tc.Handler.DefaultMFAChain = tc.DefaultMFAChain
	tc.Handler.SkipApproval = tc.SkipApproval

	h := &tc.Handler

	router := mux.NewRouter()
	h.Mount(testMux{router})
	mfaManager.Mount(testMux{router})
	consentManager.Mount(testMux{router})
	logoutManager.Mount(testMux{router})
	handler = router

	for _, id := range []string{"mock", "mock2"} {
		require.NoError(t, store.CreateConnector(ctx, storage.Connector{
			ID:              id,
			Type:            "mockCallback",
			Name:            "Mock",
			ResourceVersion: "1",
		}))
	}

	return srv, &testServer{Handler: h, mux: router, mfa: mfaManager}
}

// testKey is a throwaway RSA key for the mock signer; the flow's unit tests
// don't verify signatures, so a freshly generated key is enough.
var testKey = func() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return key
}()

// toResponseTypeSet converts a list of response types to the set form the
// Handler expects.
func toResponseTypeSet(types []string) map[string]bool {
	m := make(map[string]bool, len(types))
	for _, t := range types {
		m[t] = true
	}
	return m
}

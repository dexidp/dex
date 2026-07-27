package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/featureflags"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/session"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func newLogger(t *testing.T) *slog.Logger {
	return slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func boolPtr(v bool) *bool {
	return &v
}

// isFlowPath reports whether a redirect target is an internal step of the login
// chain (the /auth dispatcher, MFA factors, consent) rather than the client
// redirect.
func isFlowPath(p string) bool {
	if strings.HasSuffix(p, "/auth") { // dispatcher re-entry
		return true
	}
	for _, step := range []string{"/mfa/", "/approval"} {
		if strings.Contains(p, step) {
			return true
		}
	}
	return false
}

// followFlow walks the internal redirects of the login chain starting from rr
// (each hop is HMAC-protected and followed with a GET) and returns the path the
// flow comes to rest at: the client redirect_uri when the request is issued, or
// the flow step that renders a page (e.g. /approval). It leaves rr on the final
// response so callers can still inspect status, body or the Location query.
func followFlow(t *testing.T, s *Server, rr *httptest.ResponseRecorder) (*httptest.ResponseRecorder, string) {
	t.Helper()
	for range 10 {
		if rr.Code != http.StatusFound && rr.Code != http.StatusSeeOther {
			return rr, "" // rendered a page with no redirect
		}
		loc := rr.Header().Get("Location")
		u, err := url.Parse(loc)
		require.NoError(t, err)
		if !isFlowPath(u.Path) {
			return rr, u.Path // left the flow — this is the client redirect
		}
		next := httptest.NewRecorder()
		s.ServeHTTP(next, httptest.NewRequest(http.MethodGet, loc, nil))
		if next.Code != http.StatusFound && next.Code != http.StatusSeeOther {
			return next, u.Path // this flow step rendered (e.g. the consent screen)
		}
		rr = next
	}
	t.Fatal("followFlow: redirect loop did not settle")
	return rr, ""
}

type emptyStorage struct {
	storage.Storage
}

func (*emptyStorage) GetAuthRequest(context.Context, string) (storage.AuthRequest, error) {
	return storage.AuthRequest{}, storage.ErrNotFound
}

func mockConnectorDataTestStorage(t *testing.T, s storage.Storage) {
	ctx := t.Context()
	c := storage.Client{
		ID:           "test",
		Secret:       "barfoo",
		RedirectURIs: []string{"foo://bar.com/", "https://auth.example.com"},
		Name:         "dex client",
		LogoURL:      "https://goo.gl/JIyzIC",
	}

	err := s.CreateClient(ctx, c)
	require.NoError(t, err)

	c1 := storage.Connector{
		ID:   "test",
		Type: "mockPassword",
		Name: "mockPassword",
		Config: []byte(`{
"username": "test",
"password": "test"
}`),
	}

	err = s.CreateConnector(ctx, c1)
	require.NoError(t, err)

	c2 := storage.Connector{
		ID:   "http://any.valid.url/",
		Type: "mock",
		Name: "mockURLID",
	}

	err = s.CreateConnector(ctx, c2)
	require.NoError(t, err)
}

func setSessionsEnabled(t *testing.T, enabled bool) {
	t.Helper()
	if enabled {
		t.Setenv("DEX_SESSIONS_ENABLED", "true")
	} else {
		t.Setenv("DEX_SESSIONS_ENABLED", "false")
	}
}

// spnegoShortCircuit implements connector.PasswordConnector and connector.SPNEGOAware
// to simulate successful SPNEGO authentication on GET.
type spnegoShortCircuit struct{ Identity connector.Identity }

func (s spnegoShortCircuit) Close() error { return nil }

func (s spnegoShortCircuit) Prompt() string { return "" }

func (s spnegoShortCircuit) Login(ctx context.Context, sc connector.Scopes, u, p string) (connector.Identity, bool, error) {
	return connector.Identity{}, false, nil
}

func (s spnegoShortCircuit) TrySPNEGO(ctx context.Context, sc connector.Scopes, w http.ResponseWriter, r *http.Request) (*connector.Identity, connector.Handled, error) {
	id := s.Identity
	return &id, true, nil
}

// spnegoError implements connector.PasswordConnector and connector.SPNEGOAware
// to simulate SPNEGO authentication that fails with an error (e.g., LDAP lookup failed).
type spnegoError struct{ Err error }

func (s spnegoError) Close() error { return nil }

func (s spnegoError) Prompt() string { return "" }

func (s spnegoError) Login(ctx context.Context, sc connector.Scopes, u, p string) (connector.Identity, bool, error) {
	return connector.Identity{}, false, nil
}

func (s spnegoError) TrySPNEGO(ctx context.Context, sc connector.Scopes, w http.ResponseWriter, r *http.Request) (*connector.Identity, connector.Handled, error) {
	return nil, true, s.Err
}

func setNonEmpty(vals url.Values, key, value string) {
	if value != "" {
		vals.Set(key, value)
	}
}

// registerTestConnector creates a connector in storage and registers it in the server's connectors map.
func registerTestConnector(t *testing.T, s *Server, connID string, c connector.Connector) {
	t.Helper()
	ctx := t.Context()

	storageConn := storage.Connector{
		ID:              connID,
		Type:            "saml",
		Name:            "Test SAML",
		ResourceVersion: "1",
	}
	if err := s.storage.CreateConnector(ctx, storageConn); err != nil {
		t.Fatalf("failed to create connector in storage: %v", err)
	}

	s.connectors.Set(connID, connectors.Connector{
		ResourceVersion: "1",
		Connector:       c,
	})
}

// mockSAMLRefreshConnector implements SAMLConnector + RefreshConnector for testing.
type mockSAMLRefreshConnector struct {
	refreshIdentity connector.Identity
}

func (m *mockSAMLRefreshConnector) POSTData(s connector.Scopes, requestID string) (ssoURL, samlRequest string, err error) {
	return "", "", nil
}

func (m *mockSAMLRefreshConnector) HandlePOST(s connector.Scopes, samlResponse, inResponseTo string) (connector.Identity, error) {
	return connector.Identity{}, nil
}

func (m *mockSAMLRefreshConnector) Refresh(ctx context.Context, s connector.Scopes, ident connector.Identity) (connector.Identity, error) {
	return m.refreshIdentity, nil
}

// testSessionKey is the AES key the test servers encrypt session cookies with.
// Tests that forge a session cookie sign it with this key rather than reading
// the key back off the server they are exercising.
var testSessionKey = []byte("0123456789abcdef0123456789abcdef")

// mockConnector is the connector every test server serves.
func mockConnector(id string) storage.Connector {
	return storage.Connector{
		ID:              id,
		Type:            "mockCallback",
		Name:            "Mock",
		ResourceVersion: "1",
	}
}

// testSessionConfig is the session config the test servers run with.
func testSessionConfig() *session.Config {
	return &session.Config{
		CookieName:          "dex_session",
		CookieEncryptionKey: testSessionKey,
		AbsoluteLifetime:    24 * time.Hour,
		ValidIfNotUsedFor:   time.Hour,
	}
}

// newTestServerWith builds a server serving conns, behind an httptest.Server
// that dispatches to it. updateConfig adjusts the shared default config before
// the server is built; the caller-facing constructors below are thin wrappers
// that differ only in the connectors and the grant types they enable.
func newTestServerWith(t *testing.T, conns []storage.Connector, updateConfig func(c *Config)) (*httptest.Server, *Server) {
	t.Helper()

	var server *Server
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.ServeHTTP(w, r)
	}))

	logger := newLogger(t)
	ctx := t.Context()

	sig, err := signer.NewMockSigner(testKey)
	require.NoError(t, err, "failed to create mock signer")

	config := Config{
		Issuer:  s.URL,
		Storage: memory.New(logger),
		Web: WebConfig{
			Dir: "../web",
		},
		Logger:             logger,
		PrometheusRegistry: prometheus.NewRegistry(),
		HealthChecker:      gosundheit.New(),
		SkipApprovalScreen: true, // Don't prompt for approval, just immediately redirect with code.
		Signer:             sig,
	}
	if updateConfig != nil {
		updateConfig(&config)
	}
	s.URL = config.Issuer

	// Default rotation policy, set before the server is built so the token
	// endpoint captures it.
	if config.RefreshTokenPolicy == nil {
		config.RefreshTokenPolicy = tokens.NewRefreshStrategy(true, 0, 0, 0, config.Now)
	}

	// Mirror cmd: the session config is present iff the sessions feature flag is on.
	if featureflags.SessionsEnabled.Enabled() && config.SessionConfig == nil {
		config.SessionConfig = testSessionConfig()
	}

	for _, conn := range conns {
		require.NoError(t, config.Storage.CreateConnector(ctx, conn), "create connector")
	}

	server, err = newServer(ctx, config)
	require.NoError(t, err)

	return s, server
}

// newTestServer serves one mock connector with every implemented grant enabled.
func newTestServer(t *testing.T, updateConfig func(c *Config)) (*httptest.Server, *Server) {
	return newTestServerWith(t, []storage.Connector{mockConnector("mock")}, func(c *Config) {
		c.AllowedGrantTypes = []string{ // all implemented types
			oauth2.GrantTypeDeviceCode,
			oauth2.GrantTypeAuthorizationCode,
			oauth2.GrantTypeClientCredentials,
			oauth2.GrantTypeRefreshToken,
			oauth2.GrantTypeTokenExchange,
			oauth2.GrantTypeImplicit,
			oauth2.GrantTypePassword,
		}
		if updateConfig != nil {
			updateConfig(c)
		}
	})
}

// newTestServerMultipleConnectors serves two mock connectors, for the paths that
// depend on the connector selection screen.
func newTestServerMultipleConnectors(t *testing.T, updateConfig func(c *Config)) (*httptest.Server, *Server) {
	return newTestServerWith(t, []storage.Connector{mockConnector("mock"), mockConnector("mock2")}, updateConfig)
}

// newTestServerWithSessions serves one mock connector with sessions always on,
// regardless of the feature flag.
func newTestServerWithSessions(t *testing.T, updateConfig func(c *Config)) (*httptest.Server, *Server) {
	return newTestServerWith(t, []storage.Connector{mockConnector("mock")}, func(c *Config) {
		c.AllowedGrantTypes = []string{
			oauth2.GrantTypeAuthorizationCode,
			oauth2.GrantTypeClientCredentials,
			oauth2.GrantTypeRefreshToken,
			oauth2.GrantTypeTokenExchange,
			oauth2.GrantTypeDeviceCode,
		}
		c.SessionConfig = testSessionConfig()
		if updateConfig != nil {
			updateConfig(c)
		}
	})
}

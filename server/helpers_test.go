package server

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/storage"
)

func boolPtr(v bool) *bool {
	return &v
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

	s.mu.Lock()
	s.connectors[connID] = Connector{
		ResourceVersion: "1",
		Connector:       c,
	}
	s.mu.Unlock()
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

package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

func TestHandleInvalidOAuth2Callbacks(t *testing.T) {
	httpServer, server := newTestServer(t, func(c *Config) {
		c.Storage = &emptyStorage{c.Storage}
	})
	defer httpServer.Close()

	tests := []struct {
		TargetURI    string
		ExpectedCode int
	}{
		{"/callback", http.StatusBadRequest},
		{"/callback?code=&state=", http.StatusBadRequest},
		{"/callback?code=AAAAAAA&state=BBBBBBB", http.StatusBadRequest},
	}

	rr := httptest.NewRecorder()

	for i, r := range tests {
		server.ServeHTTP(rr, httptest.NewRequest("GET", r.TargetURI, nil))
		if rr.Code != r.ExpectedCode {
			t.Fatalf("test %d expected %d, got %d", i, r.ExpectedCode, rr.Code)
		}
	}
}

func TestHandleInvalidSAMLCallbacks(t *testing.T) {
	httpServer, server := newTestServer(t, func(c *Config) {
		c.Storage = &emptyStorage{c.Storage}
	})
	defer httpServer.Close()

	type requestForm struct {
		RelayState string
	}
	tests := []struct {
		RequestForm  requestForm
		ExpectedCode int
	}{
		{requestForm{}, http.StatusBadRequest},
		{requestForm{RelayState: "AAAAAAA"}, http.StatusBadRequest},
	}

	rr := httptest.NewRecorder()

	for i, r := range tests {
		jsonValue, err := json.Marshal(r.RequestForm)
		if err != nil {
			t.Fatal(err.Error())
		}
		server.ServeHTTP(rr, httptest.NewRequest("POST", "/callback", bytes.NewBuffer(jsonValue)))
		if rr.Code != r.ExpectedCode {
			t.Fatalf("test %d expected %d, got %d", i, r.ExpectedCode, rr.Code)
		}
	}
}

func TestFinalizeLoginCreatesUserIdentity(t *testing.T) {
	ctx := t.Context()
	setSessionsEnabled(t, true)

	connID := "mockPw"
	authReqID := "test-create-ui"
	expiry := time.Now().Add(100 * time.Second)

	httpServer, s := newTestServer(t, func(c *Config) {
		c.SkipApprovalScreen = true
		c.Now = time.Now
	})
	defer httpServer.Close()

	sc := storage.Connector{
		ID:              connID,
		Type:            "mockPassword",
		Name:            "MockPassword",
		ResourceVersion: "1",
		Config:          []byte(`{"username": "foo", "password": "password"}`),
	}
	require.NoError(t, s.storage.CreateConnector(ctx, sc))
	_, err := s.connectors.Open(sc)
	require.NoError(t, err)

	authReq := storage.AuthRequest{
		ID:            authReqID,
		ConnectorID:   connID,
		RedirectURI:   "cb",
		Expiry:        expiry,
		ResponseTypes: []string{oauth2.ResponseTypeCode},
	}
	require.NoError(t, s.storage.CreateAuthRequest(ctx, authReq))

	rr := httptest.NewRecorder()
	reqPath := fmt.Sprintf("/auth/%s/login?state=%s&back=&login=foo&password=password", connID, authReqID)
	s.ServeHTTP(rr, httptest.NewRequest("POST", reqPath, nil))

	require.Equal(t, 303, rr.Code)

	ui, err := s.storage.GetUserIdentity(ctx, "0-385-28089-0", connID)
	require.NoError(t, err, "UserIdentity should exist after login")
	require.Equal(t, "0-385-28089-0", ui.UserID)
	require.Equal(t, connID, ui.ConnectorID)
	require.Equal(t, "kilgore@kilgore.trout", ui.Claims.Email)
	require.NotZero(t, ui.CreatedAt, "CreatedAt should be set")
	require.NotZero(t, ui.LastLogin, "LastLogin should be set")
}

func TestFinalizeLoginUpdatesUserIdentity(t *testing.T) {
	ctx := t.Context()
	setSessionsEnabled(t, true)

	connID := "mockPw"
	authReqID := "test-update-ui"
	expiry := time.Now().Add(100 * time.Second)
	oldTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	httpServer, s := newTestServer(t, func(c *Config) {
		c.SkipApprovalScreen = true
		c.Now = time.Now
	})
	defer httpServer.Close()

	sc := storage.Connector{
		ID:              connID,
		Type:            "mockPassword",
		Name:            "MockPassword",
		ResourceVersion: "1",
		Config:          []byte(`{"username": "foo", "password": "password"}`),
	}
	require.NoError(t, s.storage.CreateConnector(ctx, sc))
	_, err := s.connectors.Open(sc)
	require.NoError(t, err)

	// Pre-create UserIdentity with old data
	require.NoError(t, s.storage.CreateUserIdentity(ctx, storage.UserIdentity{
		UserID:      "0-385-28089-0",
		ConnectorID: connID,
		Claims: storage.Claims{
			UserID:   "0-385-28089-0",
			Username: "Old Name",
			Email:    "old@example.com",
		},
		Consents:  map[string][]string{"existing-client": {"openid"}},
		CreatedAt: oldTime,
		LastLogin: oldTime,
	}))

	authReq := storage.AuthRequest{
		ID:            authReqID,
		ConnectorID:   connID,
		RedirectURI:   "cb",
		Expiry:        expiry,
		ResponseTypes: []string{oauth2.ResponseTypeCode},
	}
	require.NoError(t, s.storage.CreateAuthRequest(ctx, authReq))

	rr := httptest.NewRecorder()
	reqPath := fmt.Sprintf("/auth/%s/login?state=%s&back=&login=foo&password=password", connID, authReqID)
	s.ServeHTTP(rr, httptest.NewRequest("POST", reqPath, nil))

	require.Equal(t, 303, rr.Code)

	ui, err := s.storage.GetUserIdentity(ctx, "0-385-28089-0", connID)
	require.NoError(t, err, "UserIdentity should exist after login")
	require.Equal(t, "Kilgore Trout", ui.Claims.Username, "claims should be refreshed from the connector")
	require.Equal(t, "kilgore@kilgore.trout", ui.Claims.Email, "claims should be refreshed from the connector")
	require.True(t, ui.LastLogin.After(oldTime), "LastLogin should be updated")
	require.Equal(t, oldTime, ui.CreatedAt, "CreatedAt should not change on update")
	require.Equal(t, []string{"openid"}, ui.Consents["existing-client"], "existing consents should be preserved")
}

func TestFinalizeLoginSkipsUserIdentityWhenDisabled(t *testing.T) {
	ctx := t.Context()
	setSessionsEnabled(t, false)

	connID := "mockPw"
	authReqID := "test-no-ui"
	expiry := time.Now().Add(100 * time.Second)

	httpServer, s := newTestServer(t, func(c *Config) {
		c.SkipApprovalScreen = true
		c.Now = time.Now
	})
	defer httpServer.Close()

	sc := storage.Connector{
		ID:              connID,
		Type:            "mockPassword",
		Name:            "MockPassword",
		ResourceVersion: "1",
		Config:          []byte(`{"username": "foo", "password": "password"}`),
	}
	require.NoError(t, s.storage.CreateConnector(ctx, sc))
	_, err := s.connectors.Open(sc)
	require.NoError(t, err)

	authReq := storage.AuthRequest{
		ID:            authReqID,
		ConnectorID:   connID,
		RedirectURI:   "cb",
		Expiry:        expiry,
		ResponseTypes: []string{oauth2.ResponseTypeCode},
	}
	require.NoError(t, s.storage.CreateAuthRequest(ctx, authReq))

	rr := httptest.NewRecorder()
	reqPath := fmt.Sprintf("/auth/%s/login?state=%s&back=&login=foo&password=password", connID, authReqID)
	s.ServeHTTP(rr, httptest.NewRequest("POST", reqPath, nil))

	require.Equal(t, 303, rr.Code)

	_, err = s.storage.GetUserIdentity(ctx, "0-385-28089-0", connID)
	require.ErrorIs(t, err, storage.ErrNotFound, "UserIdentity should not be created when sessions disabled")
}

func TestHandlePasswordLoginWithSkipApproval(t *testing.T) {
	ctx := t.Context()

	connID := "mockPw"
	authReqID := "test"
	expiry := time.Now().Add(100 * time.Second)
	resTypes := []string{oauth2.ResponseTypeCode}

	tests := []struct {
		name                  string
		skipApproval          bool
		authReq               storage.AuthRequest
		expectedRes           string
		offlineSessionCreated bool
	}{
		{
			name:         "Force approval",
			skipApproval: false,
			authReq: storage.AuthRequest{
				ID:                  authReqID,
				ConnectorID:         connID,
				RedirectURI:         "cb",
				Expiry:              expiry,
				ResponseTypes:       resTypes,
				ForceApprovalPrompt: true,
			},
			expectedRes:           "/approval",
			offlineSessionCreated: false,
		},
		{
			name:         "Skip approval by server config",
			skipApproval: true,
			authReq: storage.AuthRequest{
				ID:                  authReqID,
				ConnectorID:         connID,
				RedirectURI:         "cb",
				Expiry:              expiry,
				ResponseTypes:       resTypes,
				ForceApprovalPrompt: true,
			},
			expectedRes:           "/approval",
			offlineSessionCreated: false,
		},
		{
			name:         "No skip",
			skipApproval: false,
			authReq: storage.AuthRequest{
				ID:                  authReqID,
				ConnectorID:         connID,
				RedirectURI:         "cb",
				Expiry:              expiry,
				ResponseTypes:       resTypes,
				ForceApprovalPrompt: false,
			},
			expectedRes:           "/approval",
			offlineSessionCreated: false,
		},
		{
			name:         "Skip approval",
			skipApproval: true,
			authReq: storage.AuthRequest{
				ID:                  authReqID,
				ConnectorID:         connID,
				RedirectURI:         "cb",
				Expiry:              expiry,
				ResponseTypes:       resTypes,
				ForceApprovalPrompt: false,
			},
			expectedRes:           "/cb",
			offlineSessionCreated: false,
		},
		{
			name:         "Force approval, request refresh token",
			skipApproval: false,
			authReq: storage.AuthRequest{
				ID:                  authReqID,
				ConnectorID:         connID,
				RedirectURI:         "cb",
				Expiry:              expiry,
				ResponseTypes:       resTypes,
				ForceApprovalPrompt: true,
				Scopes:              []string{"offline_access"},
			},
			expectedRes:           "/approval",
			offlineSessionCreated: true,
		},
		{
			name:         "Skip approval, request refresh token",
			skipApproval: true,
			authReq: storage.AuthRequest{
				ID:                  authReqID,
				ConnectorID:         connID,
				RedirectURI:         "cb",
				Expiry:              expiry,
				ResponseTypes:       resTypes,
				ForceApprovalPrompt: false,
				Scopes:              []string{"offline_access"},
			},
			expectedRes:           "/cb",
			offlineSessionCreated: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpServer, s := newTestServer(t, func(c *Config) {
				c.SkipApprovalScreen = tc.skipApproval
				c.Now = time.Now
			})
			defer httpServer.Close()

			sc := storage.Connector{
				ID:              connID,
				Type:            "mockPassword",
				Name:            "MockPassword",
				ResourceVersion: "1",
				Config:          []byte("{\"username\": \"foo\", \"password\": \"password\"}"),
			}
			if err := s.storage.CreateConnector(ctx, sc); err != nil {
				t.Fatalf("create connector: %v", err)
			}
			if _, err := s.connectors.Open(sc); err != nil {
				t.Fatalf("open connector: %v", err)
			}
			if err := s.storage.CreateAuthRequest(ctx, tc.authReq); err != nil {
				t.Fatalf("failed to create AuthRequest: %v", err)
			}

			rr := httptest.NewRecorder()

			path := fmt.Sprintf("/auth/%s/login?state=%s&back=&login=foo&password=password", connID, authReqID)
			s.ServeHTTP(rr, httptest.NewRequest("POST", path, nil))

			require.Equal(t, 303, rr.Code)

			_, restPath := followFlow(t, s, rr)
			require.Equal(t, tc.expectedRes, restPath)

			offlineSession, err := s.storage.GetOfflineSessions(ctx, "0-385-28089-0", connID)
			if tc.offlineSessionCreated {
				require.NoError(t, err)
				require.NotEmpty(t, offlineSession)
			} else {
				require.Error(t, storage.ErrNotFound, err)
			}
		})
	}
}

func TestHandleConnectorCallbackWithSkipApproval(t *testing.T) {
	ctx := t.Context()

	connID := "mock"
	authReqID := "test"
	expiry := time.Now().Add(100 * time.Second)
	resTypes := []string{oauth2.ResponseTypeCode}

	tests := []struct {
		name                  string
		skipApproval          bool
		authReq               storage.AuthRequest
		expectedRes           string
		offlineSessionCreated bool
	}{
		{
			name:         "Force approval",
			skipApproval: false,
			authReq: storage.AuthRequest{
				ID:                  authReqID,
				ConnectorID:         connID,
				RedirectURI:         "cb",
				Expiry:              expiry,
				ResponseTypes:       resTypes,
				ForceApprovalPrompt: true,
			},
			expectedRes:           "/approval",
			offlineSessionCreated: false,
		},
		{
			name:         "Skip approval by server config",
			skipApproval: true,
			authReq: storage.AuthRequest{
				ID:                  authReqID,
				ConnectorID:         connID,
				RedirectURI:         "cb",
				Expiry:              expiry,
				ResponseTypes:       resTypes,
				ForceApprovalPrompt: true,
			},
			expectedRes:           "/approval",
			offlineSessionCreated: false,
		},
		{
			name:         "Skip approval by auth request",
			skipApproval: false,
			authReq: storage.AuthRequest{
				ID:                  authReqID,
				ConnectorID:         connID,
				RedirectURI:         "cb",
				Expiry:              expiry,
				ResponseTypes:       resTypes,
				ForceApprovalPrompt: false,
			},
			expectedRes:           "/approval",
			offlineSessionCreated: false,
		},
		{
			name:         "Skip approval",
			skipApproval: true,
			authReq: storage.AuthRequest{
				ID:                  authReqID,
				ConnectorID:         connID,
				RedirectURI:         "cb",
				Expiry:              expiry,
				ResponseTypes:       resTypes,
				ForceApprovalPrompt: false,
			},
			expectedRes:           "/cb",
			offlineSessionCreated: false,
		},
		{
			name:         "Force approval, request refresh token",
			skipApproval: false,
			authReq: storage.AuthRequest{
				ID:                  authReqID,
				ConnectorID:         connID,
				RedirectURI:         "cb",
				Expiry:              expiry,
				ResponseTypes:       resTypes,
				ForceApprovalPrompt: true,
				Scopes:              []string{"offline_access"},
			},
			expectedRes:           "/approval",
			offlineSessionCreated: true,
		},
		{
			name:         "Skip approval, request refresh token",
			skipApproval: true,
			authReq: storage.AuthRequest{
				ID:                  authReqID,
				ConnectorID:         connID,
				RedirectURI:         "cb",
				Expiry:              expiry,
				ResponseTypes:       resTypes,
				ForceApprovalPrompt: false,
				Scopes:              []string{"offline_access"},
			},
			expectedRes:           "/cb",
			offlineSessionCreated: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpServer, s := newTestServer(t, func(c *Config) {
				c.SkipApprovalScreen = tc.skipApproval
				c.Now = time.Now
			})
			defer httpServer.Close()

			if err := s.storage.CreateAuthRequest(ctx, tc.authReq); err != nil {
				t.Fatalf("failed to create AuthRequest: %v", err)
			}
			rr := httptest.NewRecorder()

			path := fmt.Sprintf("/callback/%s?state=%s", connID, authReqID)
			s.ServeHTTP(rr, httptest.NewRequest("GET", path, nil))

			require.Equal(t, 303, rr.Code)

			_, restPath := followFlow(t, s, rr)
			require.Equal(t, tc.expectedRes, restPath)

			offlineSession, err := s.storage.GetOfflineSessions(ctx, "0-385-28089-0", connID)
			if tc.offlineSessionCreated {
				require.NoError(t, err)
				require.NotEmpty(t, offlineSession)
			} else {
				require.Error(t, storage.ErrNotFound, err)
			}
		})
	}
}

// SPNEGO integration test (server layer): on GET login, if connector implements SPNEGOAware
// and returns an identity, server should finalize login and redirect without rendering form.
func TestHandlePasswordLogin_SPNEGOShortCircuit(t *testing.T) {
	ctx := t.Context()
	connID := "mockPassword"
	authReqID := "spnego"
	expiry := time.Now().Add(100 * time.Second)
	resTypes := []string{oauth2.ResponseTypeCode}

	httpServer, s := newTestServer(t, func(c *Config) {
		c.SkipApprovalScreen = true
		c.Now = time.Now
	})
	defer httpServer.Close()

	// Create password connector which we will wrap with a SPNEGO-aware adapter in storage config
	sc := storage.Connector{
		ID:              connID,
		Type:            "mockPassword",
		Name:            "MockPassword",
		ResourceVersion: "1",
		Config:          []byte("{\"username\": \"foo\", \"password\": \"password\"}"),
	}
	require.NoError(t, s.storage.CreateConnector(ctx, sc))
	_, err := s.connectors.Open(sc)
	require.NoError(t, err)

	// Prepare auth request
	require.NoError(t, s.storage.CreateAuthRequest(ctx, storage.AuthRequest{
		ID:            authReqID,
		ClientID:      "client_1",
		ConnectorID:   connID,
		RedirectURI:   "cb",
		Expiry:        expiry,
		ResponseTypes: resTypes,
		Scopes:        []string{"openid"},
	}))

	// Replace the server connector with a SPNEGO-aware fake that short-circuits
	orig, _ := s.connectors.Get(ctx, connID)
	s.connectors.Set(connID, connectors.Connector{
		ResourceVersion: orig.ResourceVersion,
		Connector: spnegoShortCircuit{Identity: connector.Identity{
			UserID:        "user-id",
			Username:      "user",
			Email:         "user@example.com",
			EmailVerified: true,
		}},
	})

	// Need a client for finalizeLogin to succeed
	require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
		ID:           "client_1",
		Secret:       "secret",
		RedirectURIs: []string{"http://127.0.0.1/callback"},
		Name:         "test",
	}))

	// GET login should short-circuit and redirect to /approval or code response
	rr := httptest.NewRecorder()
	path := fmt.Sprintf("/auth/%s/login?state=%s&back=", connID, authReqID)
	s.ServeHTTP(rr, httptest.NewRequest("GET", path, nil))

	// In SkipApproval mode server may directly send code response (200) or 303 redirect
	if rr.Code != http.StatusSeeOther && rr.Code != http.StatusOK {
		t.Fatalf("expected 200 or 303, got %d", rr.Code)
	}
}

// TestHandlePasswordLogin_SPNEGOError verifies that when SPNEGO returns an error
// (e.g., Kerberos auth succeeded but LDAP lookup failed), the server renders
// an error page instead of showing an empty 401 or falling back to password form.
func TestHandlePasswordLogin_SPNEGOError(t *testing.T) {
	ctx := t.Context()
	connID := "mockPassword"
	authReqID := "spnego-err"
	expiry := time.Now().Add(100 * time.Second)
	resTypes := []string{oauth2.ResponseTypeCode}

	httpServer, s := newTestServer(t, func(c *Config) {
		c.SkipApprovalScreen = true
		c.Now = time.Now
	})
	defer httpServer.Close()

	// Create password connector
	sc := storage.Connector{
		ID:              connID,
		Type:            "mockPassword",
		Name:            "MockPassword",
		ResourceVersion: "1",
		Config:          []byte("{\"username\": \"foo\", \"password\": \"password\"}"),
	}
	require.NoError(t, s.storage.CreateConnector(ctx, sc))
	_, err := s.connectors.Open(sc)
	require.NoError(t, err)

	// Prepare auth request
	require.NoError(t, s.storage.CreateAuthRequest(ctx, storage.AuthRequest{
		ID:            authReqID,
		ClientID:      "client_1",
		ConnectorID:   connID,
		RedirectURI:   "cb",
		Expiry:        expiry,
		ResponseTypes: resTypes,
		Scopes:        []string{"openid"},
	}))

	// Replace the server connector with a SPNEGO-aware fake that returns an error
	orig, _ := s.connectors.Get(ctx, connID)
	s.connectors.Set(connID, connectors.Connector{
		ResourceVersion: orig.ResourceVersion,
		Connector:       spnegoError{Err: errors.New("ldap: user lookup failed for kerberos principal")},
	})

	// Need a client for the flow
	require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
		ID:           "client_1",
		Secret:       "secret",
		RedirectURIs: []string{"http://127.0.0.1/callback"},
		Name:         "test",
	}))

	// GET login should return 401 with error message rendered
	rr := httptest.NewRecorder()
	path := fmt.Sprintf("/auth/%s/login?state=%s&back=", connID, authReqID)
	s.ServeHTTP(rr, httptest.NewRequest("GET", path, nil))

	// Should return 401 (unauthorized) with an error page, not 200 (form) or empty response
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rr.Code)
	}

	// The response body should contain safe generic error message (not internal details)
	body := rr.Body.String()
	if !strings.Contains(body, "Authentication failed") {
		t.Fatalf("expected error page with 'Authentication failed', got: %s", body)
	}
	// Should NOT contain internal error details (per 008-hide-internal-500-error-details.patch)
	if strings.Contains(body, "ldap: user lookup failed") {
		t.Fatalf("error page should not contain internal error details")
	}
}

func TestHandleConnectorLoginGrantTypeRejection(t *testing.T) {
	ctx := t.Context()
	httpServer, s := newTestServer(t, func(c *Config) {
		c.Storage.CreateClient(ctx, storage.Client{
			ID:           "test-client",
			Secret:       "secret",
			RedirectURIs: []string{"http://example.com/callback"},
		})
	})
	defer httpServer.Close()

	// Restrict mock connector to device_code only
	err := s.storage.UpdateConnector(ctx, "mock", func(c storage.Connector) (storage.Connector, error) {
		c.GrantTypes = []string{oauth2.GrantTypeDeviceCode}
		return c, nil
	})
	require.NoError(t, err)
	s.connectors.Close("mock")

	// Try to use mock connector for auth code flow via the full server router
	rr := httptest.NewRecorder()
	reqURL := httpServer.URL + "/auth/mock?response_type=code&client_id=test-client&redirect_uri=http://example.com/callback&scope=openid"
	req := httptest.NewRequest(http.MethodGet, reqURL, nil)
	s.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), "does not support this grant type")
}

func TestConnectorDataPersistence(t *testing.T) {
	// Test that ConnectorData is correctly stored in refresh token
	// and can be used for subsequent refresh operations.
	httpServer, server := newTestServer(t, func(c *Config) {
		c.RefreshTokenPolicy = tokens.NewRefreshStrategy(true, 0, 0, 0, nil)
	})
	defer httpServer.Close()

	ctx := t.Context()
	connID := "saml-conndata"

	// Create a mock SAML connector that also implements RefreshConnector
	mockConn := &mockSAMLRefreshConnector{
		refreshIdentity: connector.Identity{
			UserID:        "refreshed-user",
			Username:      "refreshed-name",
			Email:         "refreshed@example.com",
			EmailVerified: true,
			Groups:        []string{"refreshed-group"},
		},
	}
	registerTestConnector(t, server, connID, mockConn)

	// Create client
	client := storage.Client{
		ID:           "conndata-client",
		Secret:       "conndata-secret",
		RedirectURIs: []string{"https://example.com/callback"},
		Name:         "ConnData Test Client",
	}
	require.NoError(t, server.storage.CreateClient(ctx, client))

	// Create refresh token with ConnectorData (simulating what HandlePOST would store)
	connectorData := []byte(`{"userID":"user-123","username":"testuser","email":"test@example.com","emailVerified":true,"groups":["admin","dev"]}`)
	refreshToken := storage.RefreshToken{
		ID:          "conndata-refresh",
		Token:       "conndata-token",
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
		ClientID:    client.ID,
		ConnectorID: connID,
		Scopes:      []string{"openid", "email", "offline_access"},
		Claims: storage.Claims{
			UserID:        "user-123",
			Username:      "testuser",
			Email:         "test@example.com",
			EmailVerified: true,
			Groups:        []string{"admin", "dev"},
		},
		ConnectorData: connectorData,
		Nonce:         "conndata-nonce",
	}
	require.NoError(t, server.storage.CreateRefresh(ctx, refreshToken))

	offlineSession := storage.OfflineSessions{
		UserID:        "user-123",
		ConnID:        connID,
		Refresh:       map[string]*storage.RefreshTokenRef{client.ID: {ID: refreshToken.ID, ClientID: client.ID}},
		ConnectorData: connectorData,
	}
	require.NoError(t, server.storage.CreateOfflineSessions(ctx, offlineSession))

	// Verify ConnectorData is stored correctly
	storedToken, err := server.storage.GetRefresh(ctx, refreshToken.ID)
	require.NoError(t, err)
	require.Equal(t, connectorData, storedToken.ConnectorData,
		"ConnectorData should be persisted in refresh token storage")

	// Verify ConnectorData is stored in offline session
	storedSession, err := server.storage.GetOfflineSessions(ctx, "user-123", connID)
	require.NoError(t, err)
	require.Equal(t, connectorData, storedSession.ConnectorData,
		"ConnectorData should be persisted in offline session storage")
}

package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/session"
	"github.com/dexidp/dex/storage"
)

func TestHandleAuthorizationConnectorGrantTypeFiltering(t *testing.T) {
	tests := []struct {
		name string
		// grantTypes per connector ID; nil means unrestricted
		connectorGrantTypes map[string][]string
		responseType        string
		wantCode            int
		// wantRedirectContains is checked when wantCode == 302
		wantRedirectContains string
		// wantBodyContains is checked when wantCode != 302
		wantBodyContains string
	}{
		{
			name: "one connector filtered, redirect to remaining",
			connectorGrantTypes: map[string][]string{
				"mock":  {oauth2.GrantTypeDeviceCode},
				"mock2": nil,
			},
			responseType:         "code",
			wantCode:             http.StatusFound,
			wantRedirectContains: "/auth/mock2",
		},
		{
			name: "all connectors filtered",
			connectorGrantTypes: map[string][]string{
				"mock":  {oauth2.GrantTypeDeviceCode},
				"mock2": {oauth2.GrantTypeDeviceCode},
			},
			responseType:     "code",
			wantCode:         http.StatusBadRequest,
			wantBodyContains: "No connectors available",
		},
		{
			name: "no restrictions, both available",
			connectorGrantTypes: map[string][]string{
				"mock":  nil,
				"mock2": nil,
			},
			responseType: "code",
			wantCode:     http.StatusOK,
		},
		{
			name: "implicit flow filters auth_code-only connector",
			connectorGrantTypes: map[string][]string{
				"mock":  {oauth2.GrantTypeAuthorizationCode},
				"mock2": nil,
			},
			responseType:         "token",
			wantCode:             http.StatusFound,
			wantRedirectContains: "/auth/mock2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			httpServer, s := newTestServerMultipleConnectors(t, func(c *Config) {
				c.Storage.CreateClient(ctx, storage.Client{
					ID:           "test",
					RedirectURIs: []string{"http://example.com/callback"},
				})
			})
			defer httpServer.Close()

			for id, gts := range tc.connectorGrantTypes {
				err := s.storage.UpdateConnector(ctx, id, func(c storage.Connector) (storage.Connector, error) {
					c.GrantTypes = gts
					return c, nil
				})
				require.NoError(t, err)
				s.connectors.Close(id)
			}

			rr := httptest.NewRecorder()
			reqURL := fmt.Sprintf("%s/auth?response_type=%s&client_id=test&redirect_uri=http://example.com/callback&scope=openid", httpServer.URL, tc.responseType)
			req := httptest.NewRequest(http.MethodGet, reqURL, nil)
			s.ServeHTTP(rr, req)

			require.Equal(t, tc.wantCode, rr.Code)
			if tc.wantRedirectContains != "" {
				require.Contains(t, rr.Header().Get("Location"), tc.wantRedirectContains)
			}
			if tc.wantBodyContains != "" {
				require.Contains(t, rr.Body.String(), tc.wantBodyContains)
			}
		})
	}
}

func TestHandleAuthorizationInvalidRequestWithSessions(t *testing.T) {
	ctx := t.Context()
	httpServer, s := newTestServerMultipleConnectors(t, func(c *Config) {
		c.SessionConfig = &session.Config{
			CookieName:        "dex_session",
			AbsoluteLifetime:  24 * time.Hour,
			ValidIfNotUsedFor: 1 * time.Hour,
		}
		c.Storage.CreateClient(ctx, storage.Client{
			ID:           "test",
			RedirectURIs: []string{"http://example.com/callback"},
		})
	})
	defer httpServer.Close()

	// Send a request with an unregistered redirect_uri — should not panic.
	rr := httptest.NewRecorder()
	reqURL := fmt.Sprintf("%s/auth?response_type=code&client_id=test&redirect_uri=http://evil.com/callback&scope=openid", httpServer.URL)
	req := httptest.NewRequest(http.MethodGet, reqURL, nil)
	s.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleAuthorizationWithAllowedConnectors(t *testing.T) {
	ctx := t.Context()

	httpServer, s := newTestServerMultipleConnectors(t, nil)
	defer httpServer.Close()

	// Create a client that only allows "mock" connector (not "mock2")
	client := storage.Client{
		ID:                "filtered-client",
		Secret:            "secret",
		RedirectURIs:      []string{"https://example.com/callback"},
		Name:              "Filtered Client",
		AllowedConnectors: []string{"mock"},
	}
	require.NoError(t, s.storage.CreateClient(ctx, client))

	// Request the auth page with this client - should only show "mock" connector
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", fmt.Sprintf("/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid",
		client.ID, url.QueryEscape("https://example.com/callback")), nil)
	s.ServeHTTP(rr, req)

	// With only one allowed connector and alwaysShowLogin=false (default),
	// the server should redirect directly to the connector
	require.Equal(t, http.StatusFound, rr.Code)
	location := rr.Header().Get("Location")
	require.Contains(t, location, "/auth/mock")
	require.NotContains(t, location, "mock2")
}

func TestHandleAuthorizationWithNoMatchingConnectors(t *testing.T) {
	ctx := t.Context()

	httpServer, s := newTestServerMultipleConnectors(t, nil)
	defer httpServer.Close()

	// Create a client that only allows a non-existent connector
	client := storage.Client{
		ID:                "no-connectors-client",
		Secret:            "secret",
		RedirectURIs:      []string{"https://example.com/callback"},
		Name:              "No Connectors Client",
		AllowedConnectors: []string{"nonexistent"},
	}
	require.NoError(t, s.storage.CreateClient(ctx, client))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", fmt.Sprintf("/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid",
		client.ID, url.QueryEscape("https://example.com/callback")), nil)
	s.ServeHTTP(rr, req)

	// Should return an error, not an empty login page
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleAuthorizationSessionSkipsConnectorSelection(t *testing.T) {
	ctx := t.Context()

	sessionConfig := &session.Config{
		CookieName:        "dex_session",
		AbsoluteLifetime:  24 * time.Hour,
		ValidIfNotUsedFor: 1 * time.Hour,
	}

	client := storage.Client{
		ID:           "test-client",
		Secret:       "secret",
		RedirectURIs: []string{"https://example.com/callback"},
		Name:         "Test Client",
	}

	authURL := fmt.Sprintf("/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid",
		client.ID, url.QueryEscape("https://example.com/callback"))

	createSession := func(t *testing.T, s *Server, connectorID string) *http.Cookie {
		t.Helper()
		now := time.Now()
		nonce := "test-nonce"
		session := storage.AuthSession{
			UserID:         "user1",
			ConnectorID:    connectorID,
			Nonce:          nonce,
			ClientStates:   map[string]*storage.ClientAuthState{},
			CreatedAt:      now.Add(-30 * time.Minute),
			LastActivity:   now.Add(-5 * time.Minute),
			IPAddress:      "127.0.0.1",
			UserAgent:      "test",
			AbsoluteExpiry: now.Add(24 * time.Hour),
			IdleExpiry:     now.Add(1 * time.Hour),
		}
		require.NoError(t, s.storage.CreateAuthSession(ctx, session))
		return &http.Cookie{
			Name:  "dex_session",
			Value: internal.SessionCookieValue("user1", connectorID, nonce, nil),
		}
	}

	t.Run("valid session redirects to session connector", func(t *testing.T) {
		httpServer, s := newTestServerMultipleConnectors(t, func(c *Config) {
			c.SessionConfig = sessionConfig
		})
		defer httpServer.Close()
		require.NoError(t, s.storage.CreateClient(ctx, client))

		cookie := createSession(t, s, "mock")

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", authURL, nil)
		req.AddCookie(cookie)
		s.ServeHTTP(rr, req)

		require.Equal(t, http.StatusFound, rr.Code)
		require.Contains(t, rr.Header().Get("Location"), "/auth/mock")
	})

	t.Run("prompt=select_account shows connector selection despite session", func(t *testing.T) {
		httpServer, s := newTestServerMultipleConnectors(t, func(c *Config) {
			c.SessionConfig = sessionConfig
		})
		defer httpServer.Close()
		require.NoError(t, s.storage.CreateClient(ctx, client))

		cookie := createSession(t, s, "mock")

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", authURL+"&prompt=select_account", nil)
		req.AddCookie(cookie)
		s.ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("no session shows connector selection", func(t *testing.T) {
		httpServer, s := newTestServerMultipleConnectors(t, func(c *Config) {
			c.SessionConfig = sessionConfig
		})
		defer httpServer.Close()
		require.NoError(t, s.storage.CreateClient(ctx, client))

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", authURL, nil)
		s.ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("alwaysShowLogin shows connector selection despite session", func(t *testing.T) {
		httpServer, s := newTestServerMultipleConnectors(t, func(c *Config) {
			c.SessionConfig = sessionConfig
			c.AlwaysShowLoginScreen = true
		})
		defer httpServer.Close()
		require.NoError(t, s.storage.CreateClient(ctx, client))

		cookie := createSession(t, s, "mock")

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", authURL, nil)
		req.AddCookie(cookie)
		s.ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("session connector not in filtered list shows connector selection", func(t *testing.T) {
		httpServer, s := newTestServerMultipleConnectors(t, func(c *Config) {
			c.SessionConfig = sessionConfig
		})
		defer httpServer.Close()

		filteredClient := storage.Client{
			ID:                "filtered-client",
			Secret:            "secret",
			RedirectURIs:      []string{"https://example.com/callback"},
			Name:              "Filtered Client",
			AllowedConnectors: []string{"mock", "mock2"},
		}
		require.NoError(t, s.storage.CreateClient(ctx, filteredClient))

		// Session is for "other-connector" which is not in the allowed list.
		cookie := createSession(t, s, "other-connector")

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", fmt.Sprintf("/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid",
			filteredClient.ID, url.QueryEscape("https://example.com/callback")), nil)
		req.AddCookie(cookie)
		s.ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestHandleAuthorizationWithoutAllowedConnectors(t *testing.T) {
	ctx := t.Context()

	httpServer, s := newTestServerMultipleConnectors(t, nil)
	defer httpServer.Close()

	// Create a client with no connector restrictions
	client := storage.Client{
		ID:           "unfiltered-client",
		Secret:       "secret",
		RedirectURIs: []string{"https://example.com/callback"},
		Name:         "Unfiltered Client",
	}
	require.NoError(t, s.storage.CreateClient(ctx, client))

	// Request the auth page - should show all connectors (rendered as HTML)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", fmt.Sprintf("/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid",
		client.ID, url.QueryEscape("https://example.com/callback")), nil)
	s.ServeHTTP(rr, req)

	// With multiple connectors and no filter, the login page should be rendered (200 OK)
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestBackLinkIncludesPromptSelectAccount(t *testing.T) {
	ctx := t.Context()

	httpServer, s := newTestServerMultipleConnectors(t, func(c *Config) {
		// select_account prompt only works with the sessions feature flag enabled.
		c.SessionConfig = &session.Config{}
	})
	defer httpServer.Close()

	// Add a password connector so handleConnectorLogin passes the backlink via redirect.
	pwConn := storage.Connector{
		ID:              "mockPw",
		Type:            "mockPassword",
		Name:            "MockPassword",
		ResourceVersion: "1",
		Config:          []byte(`{"username": "foo", "password": "bar"}`),
	}
	require.NoError(t, s.storage.CreateConnector(ctx, pwConn))
	_, err := s.connectors.Open(pwConn)
	require.NoError(t, err)

	client := storage.Client{
		ID:           "test-client",
		Secret:       "secret",
		RedirectURIs: []string{"https://example.com/callback"},
		Name:         "Test Client",
	}
	require.NoError(t, s.storage.CreateClient(ctx, client))

	rr := httptest.NewRecorder()
	authURL := fmt.Sprintf("/auth/mockPw?client_id=%s&redirect_uri=%s&response_type=code&scope=openid",
		client.ID, url.QueryEscape("https://example.com/callback"))
	req := httptest.NewRequest("GET", authURL, nil)
	s.ServeHTTP(rr, req)

	require.Equal(t, http.StatusFound, rr.Code)

	loc, err := url.Parse(rr.Header().Get("Location"))
	require.NoError(t, err)

	backLink := loc.Query().Get("back")
	require.NotEmpty(t, backLink, "back link should be set when multiple connectors exist")

	backURL, err := url.Parse(backLink)
	require.NoError(t, err)
	require.Equal(t, "select_account", backURL.Query().Get("prompt"),
		"back link should include prompt=select_account")
}

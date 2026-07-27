package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/dexidp/dex/storage"
)

func TestHandlePassword(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name                  string
		scopes                string
		offlineSessionCreated bool
	}{
		{
			name:                  "Password login, request refresh token",
			scopes:                "openid offline_access email",
			offlineSessionCreated: true,
		},
		{
			name:                  "Password login",
			scopes:                "openid email",
			offlineSessionCreated: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup a dex server.
			httpServer, s := newTestServer(t, func(c *Config) {
				c.PasswordConnector = "test"
				c.Now = time.Now
			})
			defer httpServer.Close()

			mockConnectorDataTestStorage(t, s.storage)

			makeReq := func(username, password string) *httptest.ResponseRecorder {
				u, err := url.Parse(s.issuerURL.String())
				require.NoError(t, err)

				u.Path = path.Join(u.Path, "/token")
				v := url.Values{}
				v.Add("scope", tc.scopes)
				v.Add("grant_type", "password")
				v.Add("username", username)
				v.Add("password", password)

				req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(v.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
				req.SetBasicAuth("test", "barfoo")

				rr := httptest.NewRecorder()
				s.ServeHTTP(rr, req)

				return rr
			}

			// Check unauthorized error
			{
				rr := makeReq("test", "invalid")
				require.Equal(t, 401, rr.Code)
			}

			// Check that we received expected refresh token
			{
				rr := makeReq("test", "test")
				require.Equal(t, 200, rr.Code)

				var ref struct {
					Token string `json:"refresh_token"`
				}
				err := json.Unmarshal(rr.Body.Bytes(), &ref)
				require.NoError(t, err)

				newSess, err := s.storage.GetOfflineSessions(ctx, "0-385-28089-0", "test")
				if tc.offlineSessionCreated {
					require.NoError(t, err)
					require.Equal(t, `{"test": "true"}`, string(newSess.ConnectorData))
				} else {
					require.Error(t, storage.ErrNotFound, err)
				}
			}
		})
	}
}

func TestHandlePassword_LocalPasswordDBClaims(t *testing.T) {
	ctx := t.Context()

	// Setup a dex server.
	httpServer, s := newTestServer(t, func(c *Config) {
		c.PasswordConnector = "local"
	})
	defer httpServer.Close()

	// Client credentials for password grant.
	client := storage.Client{
		ID:           "test",
		Secret:       "barfoo",
		RedirectURIs: []string{"foo://bar.com/", "https://auth.example.com"},
	}
	require.NoError(t, s.storage.CreateClient(ctx, client))

	// Enable local connector.
	localConn := storage.Connector{
		ID:              "local",
		Type:            LocalConnector,
		Name:            "Email",
		ResourceVersion: "1",
	}
	require.NoError(t, s.storage.CreateConnector(ctx, localConn))
	_, err := s.OpenConnector(localConn)
	require.NoError(t, err)

	// Create a user in the password DB with groups and preferred_username.
	pw := "secret"
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	require.NoError(t, err)
	require.NoError(t, s.storage.CreatePassword(ctx, storage.Password{
		Email:             "user@example.com",
		Username:          "user-login",
		Name:              "User Full Name",
		EmailVerified:     boolPtr(false),
		PreferredUsername: "user-public",
		UserID:            "user-id",
		Groups:            []string{"team-a", "team-a/admins"},
		Hash:              hash,
	}))

	u, err := url.Parse(s.issuerURL.String())
	require.NoError(t, err)
	u.Path = path.Join(u.Path, "/token")

	v := url.Values{}
	v.Add("scope", "openid profile email groups")
	v.Add("grant_type", "password")
	v.Add("username", "user@example.com")
	v.Add("password", pw)

	req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(v.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("test", "barfoo")

	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var tokenResponse struct {
		IDToken string `json:"id_token"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &tokenResponse))
	require.NotEmpty(t, tokenResponse.IDToken)

	p, err := oidc.NewProvider(ctx, httpServer.URL)
	require.NoError(t, err)
	idToken, err := p.Verifier(&oidc.Config{SkipClientIDCheck: true}).Verify(ctx, tokenResponse.IDToken)
	require.NoError(t, err)

	var claims struct {
		Name              string   `json:"name"`
		EmailVerified     bool     `json:"email_verified"`
		PreferredUsername string   `json:"preferred_username"`
		Groups            []string `json:"groups"`
	}
	require.NoError(t, idToken.Claims(&claims))
	require.Equal(t, "User Full Name", claims.Name)
	require.False(t, claims.EmailVerified)
	require.Equal(t, "user-public", claims.PreferredUsername)
	require.Equal(t, []string{"team-a", "team-a/admins"}, claims.Groups)
}

// TestHandlePasswordAllowedConnectors verifies the password grant enforces the
// client's AllowedConnectors (mirrors TestHandleTokenExchangeAllowedConnectors
// and TestHandleRefreshTokenAllowedConnectors — every token grant must enforce it).
func TestHandlePasswordAllowedConnectors(t *testing.T) {
	tests := []struct {
		name              string
		allowedConnectors []string
		expectedCode      int
	}{
		{"connector in allowed list", []string{"test"}, http.StatusOK},
		{"connector matches non-first entry", []string{"other", "test"}, http.StatusOK},
		{"connector not in allowed list", []string{"other"}, http.StatusBadRequest},
		{"empty allowed list permits any connector", nil, http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			httpServer, s := newTestServer(t, func(c *Config) {
				c.PasswordConnector = "test"
				c.Now = time.Now
			})
			defer httpServer.Close()

			mockConnectorDataTestStorage(t, s.storage)
			require.NoError(t, s.storage.UpdateClient(ctx, "test", func(c storage.Client) (storage.Client, error) {
				c.AllowedConnectors = tc.allowedConnectors
				return c, nil
			}))

			u, err := url.Parse(s.issuerURL.String())
			require.NoError(t, err)
			u.Path = path.Join(u.Path, "/token")

			v := url.Values{}
			v.Add("scope", "openid email")
			v.Add("grant_type", "password")
			v.Add("username", "test")
			v.Add("password", "test")

			req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(v.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.SetBasicAuth("test", "barfoo")

			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)
			require.Equal(t, tc.expectedCode, rr.Code, rr.Body.String())
			if tc.expectedCode == http.StatusBadRequest {
				require.Contains(t, rr.Body.String(), "Connector not allowed",
					"rejection must be for the connector policy, not an unrelated reason")
			}
		})
	}
}

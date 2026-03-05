package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"testing"
	"time"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	"github.com/AppsFlyer/go-sundheit/checks"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

func boolPtr(v bool) *bool {
	return &v
}

func TestHandleHealth(t *testing.T) {
	httpServer, server := newTestServer(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", "/healthz", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 got %d", rr.Code)
	}
}

func TestHandleDiscovery(t *testing.T) {
	httpServer, server := newTestServer(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", "/.well-known/openid-configuration", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 got %d", rr.Code)
	}

	var res discovery
	err := json.NewDecoder(rr.Result().Body).Decode(&res)
	require.NoError(t, err)
	require.Equal(t, discovery{
		Issuer:         httpServer.URL,
		Auth:           fmt.Sprintf("%s/auth", httpServer.URL),
		Token:          fmt.Sprintf("%s/token", httpServer.URL),
		Keys:           fmt.Sprintf("%s/keys", httpServer.URL),
		UserInfo:       fmt.Sprintf("%s/userinfo", httpServer.URL),
		DeviceEndpoint: fmt.Sprintf("%s/device/code", httpServer.URL),
		Introspect:     fmt.Sprintf("%s/token/introspect", httpServer.URL),
		GrantTypes: []string{
			"authorization_code",
			"client_credentials",
			"refresh_token",
			"urn:ietf:params:oauth:grant-type:device_code",
			"urn:ietf:params:oauth:grant-type:token-exchange",
		},
		ResponseTypes: []string{
			"code",
		},
		Subjects: []string{
			"public",
		},
		IDTokenAlgs: []string{
			"RS256",
		},
		CodeChallengeAlgs: []string{
			"S256",
			"plain",
		},
		Scopes: []string{
			"openid",
			"email",
			"groups",
			"profile",
			"offline_access",
		},
		AuthMethods: []string{
			"client_secret_basic",
			"client_secret_post",
		},
		Claims: []string{
			"iss",
			"sub",
			"aud",
			"iat",
			"exp",
			"email",
			"email_verified",
			"locale",
			"name",
			"preferred_username",
			"at_hash",
		},
	}, res)
}

func TestHandleHealthFailure(t *testing.T) {
	httpServer, server := newTestServer(t, func(c *Config) {
		c.HealthChecker = gosundheit.New()

		c.HealthChecker.RegisterCheck(
			&checks.CustomCheck{
				CheckName: "fail",
				CheckFunc: func(_ context.Context) (details interface{}, err error) {
					return nil, errors.New("error")
				},
			},
			gosundheit.InitiallyPassing(false),
			gosundheit.ExecutionPeriod(1*time.Second),
		)
	})
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", "/healthz", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 got %d", rr.Code)
	}
}

type emptyStorage struct {
	storage.Storage
}

func (*emptyStorage) GetAuthRequest(context.Context, string) (storage.AuthRequest, error) {
	return storage.AuthRequest{}, storage.ErrNotFound
}

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

// TestHandleAuthCode checks that it is forbidden to use same code twice
func TestHandleAuthCode(t *testing.T) {
	tests := []struct {
		name       string
		handleCode func(*testing.T, context.Context, *oauth2.Config, string)
	}{
		{
			name: "Code Reuse should return invalid_grant",
			handleCode: func(t *testing.T, ctx context.Context, oauth2Config *oauth2.Config, code string) {
				_, err := oauth2Config.Exchange(ctx, code)
				require.NoError(t, err)

				_, err = oauth2Config.Exchange(ctx, code)
				require.Error(t, err)

				oauth2Err, ok := err.(*oauth2.RetrieveError)
				require.True(t, ok)

				var errResponse struct{ Error string }
				err = json.Unmarshal(oauth2Err.Body, &errResponse)
				require.NoError(t, err)

				// invalid_grant must be returned for invalid values
				// https://tools.ietf.org/html/rfc6749#section-5.2
				require.Equal(t, errInvalidGrant, errResponse.Error)
			},
		},
		{
			name: "No Code should return invalid_request",
			handleCode: func(t *testing.T, ctx context.Context, oauth2Config *oauth2.Config, _ string) {
				_, err := oauth2Config.Exchange(ctx, "")
				require.Error(t, err)

				oauth2Err, ok := err.(*oauth2.RetrieveError)
				require.True(t, ok)

				var errResponse struct{ Error string }
				err = json.Unmarshal(oauth2Err.Body, &errResponse)
				require.NoError(t, err)

				require.Equal(t, errInvalidRequest, errResponse.Error)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()

			httpServer, s := newTestServer(t, func(c *Config) { c.Issuer += "/non-root-path" })
			defer httpServer.Close()

			p, err := oidc.NewProvider(ctx, httpServer.URL)
			require.NoError(t, err)

			var oauth2Client oauth2Client
			oauth2Client.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/callback" {
					http.Redirect(w, r, oauth2Client.config.AuthCodeURL(""), http.StatusSeeOther)
					return
				}

				q := r.URL.Query()
				require.Equal(t, q.Get("error"), "", q.Get("error_description"))

				code := q.Get("code")
				tc.handleCode(t, ctx, oauth2Client.config, code)

				w.WriteHeader(http.StatusOK)
			}))
			defer oauth2Client.server.Close()

			redirectURL := oauth2Client.server.URL + "/callback"
			client := storage.Client{
				ID:           "testclient",
				Secret:       "testclientsecret",
				RedirectURIs: []string{redirectURL},
			}
			err = s.storage.CreateClient(ctx, client)
			require.NoError(t, err)

			oauth2Client.config = &oauth2.Config{
				ClientID:     client.ID,
				ClientSecret: client.Secret,
				Endpoint:     p.Endpoint(),
				Scopes:       []string{oidc.ScopeOpenID, "email", "offline_access"},
				RedirectURL:  redirectURL,
			}

			resp, err := http.Get(oauth2Client.server.URL + "/login")
			require.NoError(t, err)

			resp.Body.Close()
		})
	}
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

func TestHandlePasswordLoginWithSkipApproval(t *testing.T) {
	ctx := t.Context()

	connID := "mockPw"
	authReqID := "test"
	expiry := time.Now().Add(100 * time.Second)
	resTypes := []string{responseTypeCode}

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
			expectedRes:           "/auth/mockPw/cb",
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
			expectedRes:           "/auth/mockPw/cb",
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
			if _, err := s.OpenConnector(sc); err != nil {
				t.Fatalf("open connector: %v", err)
			}
			if err := s.storage.CreateAuthRequest(ctx, tc.authReq); err != nil {
				t.Fatalf("failed to create AuthRequest: %v", err)
			}

			rr := httptest.NewRecorder()

			path := fmt.Sprintf("/auth/%s/login?state=%s&back=&login=foo&password=password", connID, authReqID)
			s.handlePasswordLogin(rr, httptest.NewRequest("POST", path, nil))

			require.Equal(t, 303, rr.Code)

			resp := rr.Result()

			defer resp.Body.Close()

			cb, _ := url.Parse(resp.Header.Get("Location"))
			require.Equal(t, tc.expectedRes, cb.Path)

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

func TestHandleClientCredentials(t *testing.T) {
	tests := []struct {
		name          string
		clientID      string
		clientSecret  string
		scopes        string
		wantCode      int
		wantAccessTok bool
		wantIDToken   bool
		wantUsername  string
	}{
		{
			name:          "Basic grant, no scopes",
			clientID:      "test",
			clientSecret:  "barfoo",
			scopes:        "",
			wantCode:      200,
			wantAccessTok: true,
			wantIDToken:   false,
		},
		{
			name:          "With openid scope",
			clientID:      "test",
			clientSecret:  "barfoo",
			scopes:        "openid",
			wantCode:      200,
			wantAccessTok: true,
			wantIDToken:   true,
		},
		{
			name:          "With openid and profile scope includes username",
			clientID:      "test",
			clientSecret:  "barfoo",
			scopes:        "openid profile",
			wantCode:      200,
			wantAccessTok: true,
			wantIDToken:   true,
			wantUsername:  "Test Client",
		},
		{
			name:          "With openid email profile groups",
			clientID:      "test",
			clientSecret:  "barfoo",
			scopes:        "openid email profile groups",
			wantCode:      200,
			wantAccessTok: true,
			wantIDToken:   true,
			wantUsername:  "Test Client",
		},
		{
			name:         "Invalid client secret",
			clientID:     "test",
			clientSecret: "wrong",
			scopes:       "",
			wantCode:     401,
		},
		{
			name:         "Unknown client",
			clientID:     "nonexistent",
			clientSecret: "secret",
			scopes:       "",
			wantCode:     401,
		},
		{
			name:         "offline_access scope rejected",
			clientID:     "test",
			clientSecret: "barfoo",
			scopes:       "openid offline_access",
			wantCode:     400,
		},
		{
			name:         "Unrecognized scope",
			clientID:     "test",
			clientSecret: "barfoo",
			scopes:       "openid bogus",
			wantCode:     400,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()

			httpServer, s := newTestServer(t, func(c *Config) {
				c.Now = time.Now
			})
			defer httpServer.Close()

			// Create a confidential client for testing.
			err := s.storage.CreateClient(ctx, storage.Client{
				ID:           "test",
				Secret:       "barfoo",
				RedirectURIs: []string{"https://example.com/callback"},
				Name:         "Test Client",
			})
			require.NoError(t, err)

			u, err := url.Parse(s.issuerURL.String())
			require.NoError(t, err)
			u.Path = path.Join(u.Path, "/token")

			v := url.Values{}
			v.Add("grant_type", "client_credentials")
			if tc.scopes != "" {
				v.Add("scope", tc.scopes)
			}

			req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(v.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.SetBasicAuth(tc.clientID, tc.clientSecret)

			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)

			require.Equal(t, tc.wantCode, rr.Code)

			if tc.wantCode == 200 {
				var resp struct {
					AccessToken  string `json:"access_token"`
					TokenType    string `json:"token_type"`
					ExpiresIn    int    `json:"expires_in"`
					IDToken      string `json:"id_token"`
					RefreshToken string `json:"refresh_token"`
				}
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)

				if tc.wantAccessTok {
					require.NotEmpty(t, resp.AccessToken)
					require.Equal(t, "bearer", resp.TokenType)
					require.Greater(t, resp.ExpiresIn, 0)
				}
				if tc.wantIDToken {
					require.NotEmpty(t, resp.IDToken)

					// Verify the ID token claims.
					provider, err := oidc.NewProvider(ctx, httpServer.URL)
					require.NoError(t, err)
					verifier := provider.Verifier(&oidc.Config{ClientID: tc.clientID})
					idToken, err := verifier.Verify(ctx, resp.IDToken)
					require.NoError(t, err)

					// Decode the subject to verify the connector ID.
					var sub internal.IDTokenSubject
					require.NoError(t, internal.Unmarshal(idToken.Subject, &sub))
					require.Equal(t, "", sub.ConnId)
					require.Equal(t, tc.clientID, sub.UserId)

					var claims struct {
						Name              string `json:"name"`
						PreferredUsername string `json:"preferred_username"`
					}
					require.NoError(t, idToken.Claims(&claims))

					if tc.wantUsername != "" {
						require.Equal(t, tc.wantUsername, claims.Name)
						require.Equal(t, tc.wantUsername, claims.PreferredUsername)
					} else {
						require.Empty(t, claims.Name)
						require.Empty(t, claims.PreferredUsername)
					}
				} else {
					require.Empty(t, resp.IDToken)
				}
				// client_credentials must never return a refresh token.
				require.Empty(t, resp.RefreshToken)
			}
		})
	}
}

func TestHandleConnectorCallbackWithSkipApproval(t *testing.T) {
	ctx := t.Context()

	connID := "mock"
	authReqID := "test"
	expiry := time.Now().Add(100 * time.Second)
	resTypes := []string{responseTypeCode}

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
			expectedRes:           "/callback/cb",
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
			expectedRes:           "/callback/cb",
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
			s.handleConnectorCallback(rr, httptest.NewRequest("GET", path, nil))

			require.Equal(t, 303, rr.Code)

			resp := rr.Result()
			defer resp.Body.Close()

			cb, _ := url.Parse(resp.Header.Get("Location"))
			require.Equal(t, tc.expectedRes, cb.Path)

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

func TestHandleTokenExchange(t *testing.T) {
	tests := []struct {
		name               string
		scope              string
		requestedTokenType string
		subjectTokenType   string
		subjectToken       string

		expectedCode      int
		expectedTokenType string
	}{
		{
			"id-for-acccess",
			"openid",
			tokenTypeAccess,
			tokenTypeID,
			"foobar",
			http.StatusOK,
			tokenTypeAccess,
		},
		{
			"id-for-id",
			"openid",
			tokenTypeID,
			tokenTypeID,
			"foobar",
			http.StatusOK,
			tokenTypeID,
		},
		{
			"id-for-default",
			"openid",
			"",
			tokenTypeID,
			"foobar",
			http.StatusOK,
			tokenTypeAccess,
		},
		{
			"access-for-access",
			"openid",
			tokenTypeAccess,
			tokenTypeAccess,
			"foobar",
			http.StatusOK,
			tokenTypeAccess,
		},
		{
			"missing-subject_token_type",
			"openid",
			tokenTypeAccess,
			"",
			"foobar",
			http.StatusBadRequest,
			"",
		},
		{
			"missing-subject_token",
			"openid",
			tokenTypeAccess,
			tokenTypeAccess,
			"",
			http.StatusBadRequest,
			"",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			httpServer, s := newTestServer(t, func(c *Config) {
				c.Storage.CreateClient(ctx, storage.Client{
					ID:     "client_1",
					Secret: "secret_1",
				})
			})
			defer httpServer.Close()
			vals := make(url.Values)
			vals.Set("grant_type", grantTypeTokenExchange)
			setNonEmpty(vals, "connector_id", "mock")
			setNonEmpty(vals, "scope", tc.scope)
			setNonEmpty(vals, "requested_token_type", tc.requestedTokenType)
			setNonEmpty(vals, "subject_token_type", tc.subjectTokenType)
			setNonEmpty(vals, "subject_token", tc.subjectToken)
			setNonEmpty(vals, "client_id", "client_1")
			setNonEmpty(vals, "client_secret", "secret_1")

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, httpServer.URL+"/token", strings.NewReader(vals.Encode()))
			req.Header.Set("content-type", "application/x-www-form-urlencoded")

			s.handleToken(rr, req)

			require.Equal(t, tc.expectedCode, rr.Code, rr.Body.String())
			require.Equal(t, "application/json", rr.Result().Header.Get("content-type"))
			if tc.expectedCode == http.StatusOK {
				var res accessTokenResponse
				err := json.NewDecoder(rr.Result().Body).Decode(&res)
				require.NoError(t, err)
				require.Equal(t, tc.expectedTokenType, res.IssuedTokenType)
			}
		})
	}
}

func TestHandleTokenExchangeConnectorGrantTypeRestriction(t *testing.T) {
	ctx := t.Context()
	httpServer, s := newTestServer(t, func(c *Config) {
		c.Storage.CreateClient(ctx, storage.Client{
			ID:     "client_1",
			Secret: "secret_1",
		})
	})
	defer httpServer.Close()

	// Restrict mock connector to authorization_code only
	err := s.storage.UpdateConnector(ctx, "mock", func(c storage.Connector) (storage.Connector, error) {
		c.GrantTypes = []string{grantTypeAuthorizationCode}
		return c, nil
	})
	require.NoError(t, err)
	// Clear cached connector to pick up new grant types
	s.mu.Lock()
	delete(s.connectors, "mock")
	s.mu.Unlock()

	vals := make(url.Values)
	vals.Set("grant_type", grantTypeTokenExchange)
	vals.Set("connector_id", "mock")
	vals.Set("scope", "openid")
	vals.Set("requested_token_type", tokenTypeAccess)
	vals.Set("subject_token_type", tokenTypeID)
	vals.Set("subject_token", "foobar")
	vals.Set("client_id", "client_1")
	vals.Set("client_secret", "secret_1")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, httpServer.URL+"/token", strings.NewReader(vals.Encode()))
	req.Header.Set("content-type", "application/x-www-form-urlencoded")

	s.handleToken(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
}

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
				"mock":  {grantTypeDeviceCode},
				"mock2": nil,
			},
			responseType:         "code",
			wantCode:             http.StatusFound,
			wantRedirectContains: "/auth/mock2",
		},
		{
			name: "all connectors filtered",
			connectorGrantTypes: map[string][]string{
				"mock":  {grantTypeDeviceCode},
				"mock2": {grantTypeDeviceCode},
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
				"mock":  {grantTypeAuthorizationCode},
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
				s.mu.Lock()
				delete(s.connectors, id)
				s.mu.Unlock()
			}

			rr := httptest.NewRecorder()
			reqURL := fmt.Sprintf("%s/auth?response_type=%s&client_id=test&redirect_uri=http://example.com/callback&scope=openid", httpServer.URL, tc.responseType)
			req := httptest.NewRequest(http.MethodGet, reqURL, nil)
			s.handleAuthorization(rr, req)

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
		c.GrantTypes = []string{grantTypeDeviceCode}
		return c, nil
	})
	require.NoError(t, err)
	s.mu.Lock()
	delete(s.connectors, "mock")
	s.mu.Unlock()

	// Try to use mock connector for auth code flow via the full server router
	rr := httptest.NewRecorder()
	reqURL := httpServer.URL + "/auth/mock?response_type=code&client_id=test-client&redirect_uri=http://example.com/callback&scope=openid"
	req := httptest.NewRequest(http.MethodGet, reqURL, nil)
	s.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), "does not support this grant type")
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

func TestConnectorDataPersistence(t *testing.T) {
	// Test that ConnectorData is correctly stored in refresh token
	// and can be used for subsequent refresh operations.
	httpServer, server := newTestServer(t, func(c *Config) {
		c.RefreshTokenPolicy = &RefreshTokenPolicy{rotateRefreshTokens: true}
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

// makeTestJWT builds a minimal JWT with the given sub for testing.
func makeTestJWT(sub string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"` + sub + `","iss":"https://issuer.example","aud":"client_1","exp":9999999999}`))
	return header + "." + payload + ".fakesig"
}

// TestExtractJWTSubAndAud tests extractJWTSubAndAud.
func TestExtractJWTSubAndAud(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantSub string
		wantAud []string
		wantErr bool
	}{
		{
			name:    "valid JWT returns sub and aud",
			token:   makeTestJWT("user-abc-123"),
			wantSub: "user-abc-123",
			wantAud: []string{"client_1"},
		},
		{
			name:    "not a JWT (no dots)",
			token:   "notajwt",
			wantErr: true,
		},
		{
			name:    "invalid base64 payload",
			token:   "aGVhZGVy.!!!.c2ln",
			wantErr: true,
		},
		{
			name: "valid JWT without sub returns empty string",
			token: func() string {
				h := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
				p := base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"https://issuer.example"}`))
				return h + "." + p + ".sig"
			}(),
			wantSub: "",
			wantAud: nil,
			wantErr: false,
		},
		{
			name: "aud as array",
			token: func() string {
				h := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
				p := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"u1","aud":["a","b"]}`))
				return h + "." + p + ".sig"
			}(),
			wantSub: "u1",
			wantAud: []string{"a", "b"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sub, aud, err := extractJWTSubAndAud(tc.token)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantSub, sub)
			require.Equal(t, tc.wantAud, aud)
		})
	}
}

// TestHandleIDJAGExchange tests the ID-JAG token exchange handler.
func TestHandleIDJAGExchange(t *testing.T) {
	subjectToken := makeTestJWT("user-123")

	tests := []struct {
		name             string
		audience         string
		subjectTokenType string
		subjectToken     string
		policies         []TokenExchangePolicy
		wantCode         int
		wantTokenTypeNA  bool
	}{
		{
			name:             "happy path: valid ID-JAG issued",
			audience:         "https://resource-as.example.com",
			subjectTokenType: tokenTypeID,
			subjectToken:     subjectToken,
			wantCode:         http.StatusOK,
			wantTokenTypeNA:  true,
		},
		{
			name:             "missing audience returns 400",
			audience:         "",
			subjectTokenType: tokenTypeID,
			subjectToken:     subjectToken,
			wantCode:         http.StatusBadRequest,
		},
		{
			name:             "wrong subject_token_type returns 400",
			audience:         "https://resource-as.example.com",
			subjectTokenType: tokenTypeAccess, // must be id_token for ID-JAG
			subjectToken:     subjectToken,
			wantCode:         http.StatusBadRequest,
		},
		{
			name:             "policy denies the audience: 403",
			audience:         "https://resource-as.example.com",
			subjectTokenType: tokenTypeID,
			subjectToken:     subjectToken,
			policies: []TokenExchangePolicy{
				// client_1 may only reach other.example.com, not resource-as.example.com
				{ClientID: "client_1", AllowedAudiences: []string{"https://other.example.com"}},
			},
			wantCode: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			httpServer, s := newTestServer(t, func(c *Config) {
				require.NoError(t, c.Storage.CreateClient(ctx, storage.Client{
					ID:     "client_1",
					Secret: "secret_1",
				}))
				c.TokenExchange = TokenExchangeConfig{
					TokenTypes: []string{tokenTypeIDJAG},
				}
				c.IDJAGPolicies = tc.policies
			})
			defer httpServer.Close()

			vals := url.Values{}
			vals.Set("grant_type", grantTypeTokenExchange)
			vals.Set("requested_token_type", tokenTypeIDJAG)
			vals.Set("subject_token_type", tc.subjectTokenType)
			vals.Set("subject_token", tc.subjectToken)
			vals.Set("connector_id", "mock")
			if tc.audience != "" {
				vals.Set("audience", tc.audience)
			}
			vals.Set("client_id", "client_1")
			vals.Set("client_secret", "secret_1")

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, httpServer.URL+"/token", strings.NewReader(vals.Encode()))
			req.Header.Set("content-type", "application/x-www-form-urlencoded")
			s.handleToken(rr, req)

			require.Equal(t, tc.wantCode, rr.Code, "body: %s", rr.Body.String())

			if tc.wantTokenTypeNA {
				var res accessTokenResponse
				require.NoError(t, json.NewDecoder(rr.Result().Body).Decode(&res))
				require.Equal(t, "N_A", res.TokenType)
				require.Equal(t, tokenTypeIDJAG, res.IssuedTokenType)
				require.NotEmpty(t, res.AccessToken)
				require.Equal(t, 3, len(strings.Split(res.AccessToken, ".")), "expected compact JWT")
				require.Greater(t, res.ExpiresIn, 0)
			}
		})
	}
}

func TestFilterConnectors(t *testing.T) {
	connectors := []storage.Connector{
		{ID: "github", Type: "github", Name: "GitHub"},
		{ID: "google", Type: "oidc", Name: "Google"},
		{ID: "ldap", Type: "ldap", Name: "LDAP"},
	}

	tests := []struct {
		name              string
		allowedConnectors []string
		wantIDs           []string
	}{
		{
			name:              "No filter - all connectors returned",
			allowedConnectors: nil,
			wantIDs:           []string{"github", "google", "ldap"},
		},
		{
			name:              "Empty filter - all connectors returned",
			allowedConnectors: []string{},
			wantIDs:           []string{"github", "google", "ldap"},
		},
		{
			name:              "Filter to one connector",
			allowedConnectors: []string{"github"},
			wantIDs:           []string{"github"},
		},
		{
			name:              "Filter to two connectors",
			allowedConnectors: []string{"github", "ldap"},
			wantIDs:           []string{"github", "ldap"},
		},
		{
			name:              "Filter with non-existent connector ID",
			allowedConnectors: []string{"nonexistent"},
			wantIDs:           []string{},
		},
		{
			name:              "Filter with mix of valid and invalid IDs",
			allowedConnectors: []string{"google", "nonexistent"},
			wantIDs:           []string{"google"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := filterConnectors(connectors, tc.allowedConnectors)
			gotIDs := make([]string, len(result))
			for i, c := range result {
				gotIDs[i] = c.ID
			}
			require.Equal(t, tc.wantIDs, gotIDs)
		})
	}
}

func TestIsConnectorAllowed(t *testing.T) {
	tests := []struct {
		name              string
		allowedConnectors []string
		connectorID       string
		want              bool
	}{
		{
			name:              "No restrictions - all allowed",
			allowedConnectors: nil,
			connectorID:       "any",
			want:              true,
		},
		{
			name:              "Empty list - all allowed",
			allowedConnectors: []string{},
			connectorID:       "any",
			want:              true,
		},
		{
			name:              "Connector in allowed list",
			allowedConnectors: []string{"github", "google"},
			connectorID:       "github",
			want:              true,
		},
		{
			name:              "Connector not in allowed list",
			allowedConnectors: []string{"github", "google"},
			connectorID:       "ldap",
			want:              false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isConnectorAllowed(tc.allowedConnectors, tc.connectorID)
			require.Equal(t, tc.want, got)
		})
	}
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

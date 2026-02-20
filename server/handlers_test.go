package server

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

func setNonEmpty(vals url.Values, key, value string) {
	if value != "" {
		vals.Set(key, value)
	}
}

// mockSAMLSLOConnector implements connector.SAMLConnector and connector.SAMLSLOConnector for testing.
type mockSAMLSLOConnector struct {
	sloNameID string
	sloErr    error
}

func (m *mockSAMLSLOConnector) POSTData(s connector.Scopes, requestID string) (ssoURL, samlRequest string, err error) {
	return "", "", nil
}

func (m *mockSAMLSLOConnector) HandlePOST(s connector.Scopes, samlResponse, inResponseTo string) (connector.Identity, error) {
	return connector.Identity{}, nil
}

func (m *mockSAMLSLOConnector) HandleSLO(w http.ResponseWriter, r *http.Request) (string, error) {
	return m.sloNameID, m.sloErr
}

// mockNonSLOConnector implements connector.CallbackConnector but NOT SAMLSLOConnector.
type mockNonSLOConnector struct{}

func (m *mockNonSLOConnector) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
	return "", nil
}

func (m *mockNonSLOConnector) HandleCallback(s connector.Scopes, r *http.Request) (connector.Identity, error) {
	return connector.Identity{}, nil
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

func TestHandleSAMLSLO(t *testing.T) {
	t.Run("SuccessfulSLO", func(t *testing.T) {
		httpServer, server := newTestServer(t, nil)
		defer httpServer.Close()

		ctx := t.Context()
		connID := "saml-slo-test"
		nameID := "user@example.com"

		mockConn := &mockSAMLSLOConnector{
			sloNameID: nameID,
		}
		registerTestConnector(t, server, connID, mockConn)

		// Create refresh tokens and offline session
		refreshToken1 := storage.RefreshToken{
			ID:          "refresh-token-1",
			Token:       "token-1",
			CreatedAt:   time.Now(),
			LastUsed:    time.Now(),
			ClientID:    "test-client",
			ConnectorID: connID,
			Claims: storage.Claims{
				UserID:   nameID,
				Username: "testuser",
				Email:    nameID,
			},
			Nonce: "nonce-1",
		}
		refreshToken2 := storage.RefreshToken{
			ID:          "refresh-token-2",
			Token:       "token-2",
			CreatedAt:   time.Now(),
			LastUsed:    time.Now(),
			ClientID:    "test-client-2",
			ConnectorID: connID,
			Claims: storage.Claims{
				UserID:   nameID,
				Username: "testuser",
				Email:    nameID,
			},
			Nonce: "nonce-2",
		}
		require.NoError(t, server.storage.CreateRefresh(ctx, refreshToken1))
		require.NoError(t, server.storage.CreateRefresh(ctx, refreshToken2))

		offlineSession := storage.OfflineSessions{
			UserID: nameID,
			ConnID: connID,
			Refresh: map[string]*storage.RefreshTokenRef{
				refreshToken1.ClientID: {
					ID:        refreshToken1.ID,
					ClientID:  refreshToken1.ClientID,
					CreatedAt: refreshToken1.CreatedAt,
					LastUsed:  refreshToken1.LastUsed,
				},
				refreshToken2.ClientID: {
					ID:        refreshToken2.ID,
					ClientID:  refreshToken2.ClientID,
					CreatedAt: refreshToken2.CreatedAt,
					LastUsed:  refreshToken2.LastUsed,
				},
			},
		}
		require.NoError(t, server.storage.CreateOfflineSessions(ctx, offlineSession))

		// Send POST to /saml/slo/{connector}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/saml/slo/"+connID, nil)
		server.ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code, "expected HTTP 200, got %d: %s", rr.Code, rr.Body.String())

		// Verify refresh tokens are deleted
		_, err := server.storage.GetRefresh(ctx, refreshToken1.ID)
		require.ErrorIs(t, err, storage.ErrNotFound, "refresh token 1 should be deleted")

		_, err = server.storage.GetRefresh(ctx, refreshToken2.ID)
		require.ErrorIs(t, err, storage.ErrNotFound, "refresh token 2 should be deleted")

		// Verify offline session is deleted
		_, err = server.storage.GetOfflineSessions(ctx, nameID, connID)
		require.ErrorIs(t, err, storage.ErrNotFound, "offline session should be deleted")
	})

	t.Run("NoExistingSessions", func(t *testing.T) {
		httpServer, server := newTestServer(t, nil)
		defer httpServer.Close()

		connID := "saml-slo-nosession"
		nameID := "nouser@example.com"

		mockConn := &mockSAMLSLOConnector{
			sloNameID: nameID,
		}
		registerTestConnector(t, server, connID, mockConn)

		// No refresh tokens or offline sessions created — should handle gracefully
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/saml/slo/"+connID, nil)
		server.ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code, "expected HTTP 200 for no sessions, got %d: %s", rr.Code, rr.Body.String())
	})

	t.Run("ConnectorNotFound", func(t *testing.T) {
		httpServer, server := newTestServer(t, nil)
		defer httpServer.Close()

		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/saml/slo/nonexistent-connector", nil)
		server.ServeHTTP(rr, req)

		require.Equal(t, http.StatusNotFound, rr.Code, "expected HTTP 404 for missing connector")
	})

	t.Run("ConnectorDoesNotSupportSLO", func(t *testing.T) {
		httpServer, server := newTestServer(t, nil)
		defer httpServer.Close()

		connID := "saml-no-slo"
		mockConn := &mockNonSLOConnector{}
		registerTestConnector(t, server, connID, mockConn)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/saml/slo/"+connID, nil)
		server.ServeHTTP(rr, req)

		require.Equal(t, http.StatusBadRequest, rr.Code, "expected HTTP 400 for connector without SLO support")
	})

	t.Run("HandleSLOError", func(t *testing.T) {
		httpServer, server := newTestServer(t, nil)
		defer httpServer.Close()

		connID := "saml-slo-error"
		mockConn := &mockSAMLSLOConnector{
			sloNameID: "",
			sloErr:    errors.New("invalid SAMLRequest"),
		}
		registerTestConnector(t, server, connID, mockConn)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/saml/slo/"+connID, nil)
		server.ServeHTTP(rr, req)

		require.Equal(t, http.StatusBadRequest, rr.Code, "expected HTTP 400 for invalid SAMLRequest")
	})

	t.Run("MultipleTokensPartialDeletion", func(t *testing.T) {
		httpServer, server := newTestServer(t, nil)
		defer httpServer.Close()

		ctx := t.Context()
		connID := "saml-slo-partial"
		nameID := "partial@example.com"

		mockConn := &mockSAMLSLOConnector{
			sloNameID: nameID,
		}
		registerTestConnector(t, server, connID, mockConn)

		// Create only one refresh token but reference two in offline session
		// (simulating a token that was already deleted)
		refreshToken := storage.RefreshToken{
			ID:          "existing-token",
			Token:       "token-existing",
			CreatedAt:   time.Now(),
			LastUsed:    time.Now(),
			ClientID:    "client-existing",
			ConnectorID: connID,
			Claims: storage.Claims{
				UserID:   nameID,
				Username: "partialuser",
				Email:    nameID,
			},
			Nonce: "nonce-existing",
		}
		require.NoError(t, server.storage.CreateRefresh(ctx, refreshToken))

		offlineSession := storage.OfflineSessions{
			UserID: nameID,
			ConnID: connID,
			Refresh: map[string]*storage.RefreshTokenRef{
				"client-existing": {
					ID:        "existing-token",
					ClientID:  "client-existing",
					CreatedAt: time.Now(),
					LastUsed:  time.Now(),
				},
				"client-missing": {
					ID:        "already-deleted-token",
					ClientID:  "client-missing",
					CreatedAt: time.Now(),
					LastUsed:  time.Now(),
				},
			},
		}
		require.NoError(t, server.storage.CreateOfflineSessions(ctx, offlineSession))

		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/saml/slo/"+connID, nil)
		server.ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code, "expected HTTP 200 even with partial deletion")

		// Verify existing token is deleted
		_, err := server.storage.GetRefresh(ctx, "existing-token")
		require.ErrorIs(t, err, storage.ErrNotFound, "existing refresh token should be deleted")

		// Verify offline session is deleted
		_, err = server.storage.GetOfflineSessions(ctx, nameID, connID)
		require.ErrorIs(t, err, storage.ErrNotFound, "offline session should be deleted")
	})

	t.Run("RefreshAfterSLO", func(t *testing.T) {
		// Test that refresh token is rejected after SLO invalidation.
		// This is the key integration test: SLO → refresh → error.
		httpServer, server := newTestServer(t, func(c *Config) {
			c.RefreshTokenPolicy = &RefreshTokenPolicy{rotateRefreshTokens: true}
		})
		defer httpServer.Close()

		ctx := t.Context()
		connID := "saml-slo-refresh"
		nameID := "slo-refresh@example.com"

		mockConn := &mockSAMLSLOConnector{
			sloNameID: nameID,
		}
		registerTestConnector(t, server, connID, mockConn)

		// Create client for refresh token request
		client := storage.Client{
			ID:           "slo-test-client",
			Secret:       "slo-test-secret",
			RedirectURIs: []string{"https://example.com/callback"},
			Name:         "SLO Test Client",
		}
		require.NoError(t, server.storage.CreateClient(ctx, client))

		// Create refresh token
		refreshToken := storage.RefreshToken{
			ID:          "slo-refresh-token",
			Token:       "slo-token-value",
			CreatedAt:   time.Now(),
			LastUsed:    time.Now(),
			ClientID:    client.ID,
			ConnectorID: connID,
			Scopes:      []string{"openid", "email", "offline_access"},
			Claims: storage.Claims{
				UserID:        nameID,
				Username:      "slouser",
				Email:         nameID,
				EmailVerified: true,
			},
			ConnectorData: []byte(`{"userID":"` + nameID + `","username":"slouser","email":"` + nameID + `","emailVerified":true}`),
			Nonce:         "slo-nonce",
		}
		require.NoError(t, server.storage.CreateRefresh(ctx, refreshToken))

		offlineSession := storage.OfflineSessions{
			UserID: nameID,
			ConnID: connID,
			Refresh: map[string]*storage.RefreshTokenRef{
				client.ID: {
					ID:        refreshToken.ID,
					ClientID:  client.ID,
					CreatedAt: refreshToken.CreatedAt,
					LastUsed:  refreshToken.LastUsed,
				},
			},
		}
		require.NoError(t, server.storage.CreateOfflineSessions(ctx, offlineSession))

		// Step 1: Verify refresh token exists before SLO
		_, err := server.storage.GetRefresh(ctx, refreshToken.ID)
		require.NoError(t, err, "refresh token should exist before SLO")

		// Step 2: Send SLO request
		sloRR := httptest.NewRecorder()
		sloReq := httptest.NewRequest(http.MethodPost, "/saml/slo/"+connID, nil)
		server.ServeHTTP(sloRR, sloReq)
		require.Equal(t, http.StatusOK, sloRR.Code, "SLO should succeed")

		// Step 3: Verify refresh token is deleted
		_, err = server.storage.GetRefresh(ctx, refreshToken.ID)
		require.ErrorIs(t, err, storage.ErrNotFound, "refresh token should be deleted after SLO")

		// Step 4: Attempt to use the refresh token — should fail
		tokenData, err := internal.Marshal(&internal.RefreshToken{RefreshId: refreshToken.ID, Token: refreshToken.Token})
		require.NoError(t, err)

		u, err := url.Parse(server.issuerURL.String())
		require.NoError(t, err)
		u.Path = path.Join(u.Path, "/token")

		v := url.Values{}
		v.Add("grant_type", "refresh_token")
		v.Add("refresh_token", tokenData)

		refreshReq, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(v.Encode()))
		refreshReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		refreshReq.SetBasicAuth(client.ID, client.Secret)

		refreshRR := httptest.NewRecorder()
		server.ServeHTTP(refreshRR, refreshReq)

		// Refresh should fail because the token was deleted by SLO
		require.NotEqual(t, http.StatusOK, refreshRR.Code,
			"refresh should fail after SLO, got status %d: %s", refreshRR.Code, refreshRR.Body.String())
	})

	t.Run("ConnectorDataPersistence", func(t *testing.T) {
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

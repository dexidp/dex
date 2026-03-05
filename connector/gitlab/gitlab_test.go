package gitlab

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dexidp/dex/connector"
)

func readValidRootCAData(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/rootCA.pem")
	if err != nil {
		t.Fatalf("failed to read rootCA.pem testdata: %v", err)
	}
	return b
}

func newLocalHTTPSTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	ts := httptest.NewUnstartedServer(handler)
	cert, err := tls.LoadX509KeyPair("testdata/server.crt", "testdata/server.key")
	if err != nil {
		t.Fatalf("failed to load TLS test cert/key: %v", err)
	}
	ts.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	ts.StartTLS()
	return ts
}

func TestOpenWithRootCADataCreatesHTTPClient(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := &Config{
		RootCAData: readValidRootCAData(t),
	}

	conn, err := cfg.Open("test", logger)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	gc, ok := conn.(*gitlabConnector)
	if !ok {
		t.Fatalf("expected *gitlabConnector, got %T", conn)
	}
	if gc.httpClient == nil {
		t.Fatalf("expected httpClient to be non-nil")
	}
	if gc.httpClient.Timeout != 30*time.Second {
		t.Fatalf("expected httpClient timeout %v, got %v", 30*time.Second, gc.httpClient.Timeout)
	}
	tr, ok := gc.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected transport to be *http.Transport, got %T", gc.httpClient.Transport)
	}
	// ProxyFromEnvironment is expected to be enabled (non-nil proxy func).
	if tr.Proxy == nil {
		t.Fatalf("expected transport.Proxy to be set (ProxyFromEnvironment)")
	}
	if tr.TLSClientConfig == nil || tr.TLSClientConfig.RootCAs == nil {
		t.Fatalf("expected transport TLS root CAs to be configured")
	}
}

func TestOpenWithInvalidRootCADataReturnsError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := &Config{
		RootCAData: []byte("not a pem"),
	}

	_, err := cfg.Open("test", logger)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid rootCAData") {
		t.Fatalf("expected error to contain %q, got %q", "invalid rootCAData", err.Error())
	}
}

func TestHandleCallbackCustomRootCADataEnablesTLSRequests(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	ts := newLocalHTTPSTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		switch r.URL.Path {
		case "/oauth/token":
			// oauth2.Exchange expects an access token in response.
			fmt.Fprint(w, `{"access_token":"abc","token_type":"bearer","expires_in":30}`)
		case "/api/v4/user":
			json.NewEncoder(w).Encode(gitlabUser{Email: "some@email.com", ID: 12345678})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	cfg := &Config{
		BaseURL:      ts.URL,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "https://example.invalid/callback",
		RootCAData:   readValidRootCAData(t),
	}

	conn, err := cfg.Open("test", logger)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}

	hostURL, err := url.Parse(ts.URL)
	expectNil(t, err)
	req, err := http.NewRequest("GET", hostURL.String()+"?code=testcode", nil)
	expectNil(t, err)

	identity, err := conn.(connector.CallbackConnector).HandleCallback(connector.Scopes{Groups: false}, nil, req)
	if err != nil {
		t.Fatalf("HandleCallback() error: %v", err)
	}
	if identity.Email != "some@email.com" || identity.UserID != "12345678" {
		t.Fatalf("unexpected identity: %#v", identity)
	}
}

func TestHandleCallbackWithoutRootCADataFailsTLS(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	ts := newLocalHTTPSTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		switch r.URL.Path {
		case "/oauth/token":
			fmt.Fprint(w, `{"access_token":"abc","token_type":"bearer","expires_in":30}`)
		case "/api/v4/user":
			json.NewEncoder(w).Encode(gitlabUser{Email: "some@email.com", ID: 12345678})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	cfg := &Config{
		BaseURL:      ts.URL,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "https://example.invalid/callback",
		// RootCAData intentionally omitted: should fail TLS verification against our custom server cert.
	}

	conn, err := cfg.Open("test", logger)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}

	hostURL, err := url.Parse(ts.URL)
	expectNil(t, err)
	req, err := http.NewRequest("GET", hostURL.String()+"?code=testcode", nil)
	expectNil(t, err)

	_, err = conn.(connector.CallbackConnector).HandleCallback(connector.Scopes{Groups: false}, nil, req)
	if err == nil {
		t.Fatalf("expected TLS error, got nil")
	}
}

func TestUserGroups(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/oauth/userinfo": userInfo{
			Groups: []string{"team-1", "team-2"},
		},
	})
	defer s.Close()

	c := gitlabConnector{baseURL: s.URL}
	groups, err := c.getGroups(context.Background(), newClient(), true, "joebloggs")

	expectNil(t, err)
	expectEquals(t, groups, []string{
		"team-1",
		"team-2",
	})
}

func TestUserGroupsWithFiltering(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/oauth/userinfo": userInfo{
			Groups: []string{"team-1", "team-2"},
		},
	})
	defer s.Close()

	c := gitlabConnector{baseURL: s.URL, groups: []string{"team-1"}}
	groups, err := c.getGroups(context.Background(), newClient(), true, "joebloggs")

	expectNil(t, err)
	expectEquals(t, groups, []string{
		"team-1",
	})
}

func TestUserGroupsWithoutOrgs(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/oauth/userinfo": userInfo{
			Groups: []string{},
		},
	})
	defer s.Close()

	c := gitlabConnector{baseURL: s.URL}
	groups, err := c.getGroups(context.Background(), newClient(), true, "joebloggs")

	expectNil(t, err)
	expectEquals(t, len(groups), 0)
}

// tests that the email is used as their username when they have no username set
func TestUsernameIncludedInFederatedIdentity(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/api/v4/user": gitlabUser{Email: "some@email.com", ID: 12345678},
		"/oauth/token": map[string]interface{}{
			"access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			"expires_in":   "30",
		},
		"/oauth/userinfo": userInfo{
			Groups: []string{"team-1"},
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	c := gitlabConnector{baseURL: s.URL, httpClient: newClient()}
	identity, err := c.HandleCallback(connector.Scopes{Groups: false}, nil, req)

	expectNil(t, err)
	expectEquals(t, identity.Username, "some@email.com")
	expectEquals(t, identity.UserID, "12345678")
	expectEquals(t, 0, len(identity.Groups))

	c = gitlabConnector{baseURL: s.URL, httpClient: newClient()}
	identity, err = c.HandleCallback(connector.Scopes{Groups: true}, nil, req)

	expectNil(t, err)
	expectEquals(t, identity.Username, "some@email.com")
	expectEquals(t, identity.UserID, "12345678")
	expectEquals(t, identity.Groups, []string{"team-1"})
}

func TestLoginUsedAsIDWhenConfigured(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/api/v4/user": gitlabUser{Email: "some@email.com", ID: 12345678, Name: "Joe Bloggs", Username: "joebloggs"},
		"/oauth/token": map[string]interface{}{
			"access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			"expires_in":   "30",
		},
		"/oauth/userinfo": userInfo{
			Groups: []string{"team-1"},
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	c := gitlabConnector{baseURL: s.URL, httpClient: newClient(), useLoginAsID: true}
	identity, err := c.HandleCallback(connector.Scopes{Groups: true}, nil, req)

	expectNil(t, err)
	expectEquals(t, identity.UserID, "joebloggs")
	expectEquals(t, identity.Username, "Joe Bloggs")
}

func TestLoginWithTeamWhitelisted(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/api/v4/user": gitlabUser{Email: "some@email.com", ID: 12345678, Name: "Joe Bloggs"},
		"/oauth/token": map[string]interface{}{
			"access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			"expires_in":   "30",
		},
		"/oauth/userinfo": userInfo{
			Groups: []string{"team-1"},
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	c := gitlabConnector{baseURL: s.URL, httpClient: newClient(), groups: []string{"team-1"}}
	identity, err := c.HandleCallback(connector.Scopes{Groups: true}, nil, req)

	expectNil(t, err)
	expectEquals(t, identity.UserID, "12345678")
	expectEquals(t, identity.Username, "Joe Bloggs")
}

func TestLoginWithTeamNonWhitelisted(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/api/v4/user": gitlabUser{Email: "some@email.com", ID: 12345678, Name: "Joe Bloggs", Username: "joebloggs"},
		"/oauth/token": map[string]interface{}{
			"access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			"expires_in":   "30",
		},
		"/oauth/userinfo": userInfo{
			Groups: []string{"team-1"},
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	c := gitlabConnector{baseURL: s.URL, httpClient: newClient(), groups: []string{"team-2"}}
	_, err = c.HandleCallback(connector.Scopes{Groups: true}, nil, req)

	expectNotNil(t, err, "HandleCallback error")
	expectEquals(t, err.Error(), "gitlab: get groups: gitlab: user \"joebloggs\" is not in any of the required groups")
}

func TestRefresh(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/api/v4/user": gitlabUser{Email: "some@email.com", ID: 12345678},
		"/oauth/token": map[string]interface{}{
			"access_token":  "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			"refresh_token": "oRzxVjCnohYRHEYEhZshkmakKmoyVoTjfUGC",
			"expires_in":    "30",
		},
		"/oauth/userinfo": userInfo{
			Groups: []string{"team-1"},
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	c := gitlabConnector{baseURL: s.URL, httpClient: newClient()}

	expectedConnectorData, err := json.Marshal(connectorData{
		RefreshToken: "oRzxVjCnohYRHEYEhZshkmakKmoyVoTjfUGC",
		AccessToken:  "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
	})
	expectNil(t, err)

	identity, err := c.HandleCallback(connector.Scopes{OfflineAccess: true}, nil, req)
	expectNil(t, err)
	expectEquals(t, identity.Username, "some@email.com")
	expectEquals(t, identity.UserID, "12345678")
	expectEquals(t, identity.ConnectorData, expectedConnectorData)

	identity, err = c.Refresh(context.Background(), connector.Scopes{OfflineAccess: true}, identity)
	expectNil(t, err)
	expectEquals(t, identity.Username, "some@email.com")
	expectEquals(t, identity.UserID, "12345678")
	expectEquals(t, identity.ConnectorData, expectedConnectorData)
}

func TestRefreshWithEmptyConnectorData(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/api/v4/user": gitlabUser{Email: "some@email.com", ID: 12345678},
		"/oauth/token": map[string]interface{}{
			"access_token":  "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			"refresh_token": "oRzxVjCnohYRHEYEhZshkmakKmoyVoTjfUGC",
			"expires_in":    "30",
		},
		"/oauth/userinfo": userInfo{
			Groups: []string{"team-1"},
		},
	})
	defer s.Close()

	emptyConnectorData, err := json.Marshal(connectorData{
		RefreshToken: "",
		AccessToken:  "",
	})
	expectNil(t, err)

	c := gitlabConnector{baseURL: s.URL, httpClient: newClient()}
	emptyIdentity := connector.Identity{ConnectorData: emptyConnectorData}

	identity, err := c.Refresh(context.Background(), connector.Scopes{OfflineAccess: true}, emptyIdentity)
	expectNotNil(t, err, "Refresh error")
	expectEquals(t, emptyIdentity, identity)
}

func TestGroupsWithPermission(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/api/v4/user": gitlabUser{Email: "some@email.com", ID: 12345678, Name: "Joe Bloggs", Username: "joebloggs"},
		"/oauth/token": map[string]interface{}{
			"access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			"expires_in":   "30",
		},
		"/oauth/userinfo": userInfo{
			Groups:               []string{"ops", "dev", "ops-test", "ops/project", "dev/project1", "dev/project2"},
			OwnerPermission:      []string{"ops"},
			DeveloperPermission:  []string{"dev"},
			MaintainerPermission: []string{"dev/project1"},
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	c := gitlabConnector{baseURL: s.URL, httpClient: newClient(), getGroupsPermission: true}
	identity, err := c.HandleCallback(connector.Scopes{Groups: true}, nil, req)
	expectNil(t, err)

	expectEquals(t, identity.Groups, []string{
		"ops",
		"dev",
		"ops-test",
		"ops/project",
		"dev/project1",
		"dev/project2",
		"ops:owner",
		"dev:developer",
		"ops/project:owner",
		"dev/project1:maintainer",
		"dev/project2:developer",
	})
}

func newTestServer(responses map[string]interface{}) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := responses[r.RequestURI]
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
}

func newClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &http.Client{Transport: tr}
}

func expectNil(t *testing.T, a interface{}) {
	if a != nil {
		t.Errorf("Expected %+v to equal nil", a)
	}
}

func expectNotNil(t *testing.T, a interface{}, msg string) {
	if a == nil {
		t.Errorf("Expected %+v to not to be nil", msg)
	}
}

func expectEquals(t *testing.T, a interface{}, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Expected %+v to equal %+v", a, b)
	}
}

func TestTokenIdentity(t *testing.T) {
	// Note: These tests verify that the connector returns groups based on its configuration.
	// The actual inclusion of groups in the final Dex token depends on the 'groups' scope
	// in the token exchange request, which is handled by the Dex server, not the connector.
	tests := []struct {
		name                string
		userInfo            userInfo
		groups              []string
		getGroupsPermission bool
		useLoginAsID        bool
		expectUserID        string
		expectGroups        []string
	}{
		{
			name:         "without groups config",
			expectUserID: "12345678",
			expectGroups: nil,
		},
		{
			name: "with groups filter",
			userInfo: userInfo{
				Groups: []string{"team-1", "team-2"},
			},
			groups:       []string{"team-1"},
			expectUserID: "12345678",
			expectGroups: []string{"team-1"},
		},
		{
			name: "with groups permission",
			userInfo: userInfo{
				Groups:               []string{"ops", "dev"},
				OwnerPermission:      []string{"ops"},
				DeveloperPermission:  []string{"dev"},
				MaintainerPermission: []string{},
			},
			getGroupsPermission: true,
			expectUserID:        "12345678",
			expectGroups:        []string{"ops", "dev", "ops:owner", "dev:developer"},
		},
		{
			name:         "with useLoginAsID",
			useLoginAsID: true,
			expectUserID: "joebloggs",
			expectGroups: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			responses := map[string]interface{}{
				"/api/v4/user": gitlabUser{
					Email:    "some@email.com",
					ID:       12345678,
					Name:     "Joe Bloggs",
					Username: "joebloggs",
				},
				"/oauth/userinfo": tc.userInfo,
			}

			s := newTestServer(responses)
			defer s.Close()

			c := gitlabConnector{
				baseURL:             s.URL,
				httpClient:          newClient(),
				groups:              tc.groups,
				getGroupsPermission: tc.getGroupsPermission,
				useLoginAsID:        tc.useLoginAsID,
			}

			accessToken := "test-access-token"
			ctx := context.Background()
			identity, err := c.TokenIdentity(ctx, "urn:ietf:params:oauth:token-type:access_token", accessToken)

			expectNil(t, err)
			expectEquals(t, identity.UserID, tc.expectUserID)
			expectEquals(t, identity.Username, "Joe Bloggs")
			expectEquals(t, identity.PreferredUsername, "joebloggs")
			expectEquals(t, identity.Email, "some@email.com")
			expectEquals(t, identity.EmailVerified, true)
			expectEquals(t, identity.Groups, tc.expectGroups)
		})
	}
}

package github

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dexidp/dex/connector"
)

type testResponse struct {
	data     interface{}
	nextLink string
	lastLink string
}

func TestUserGroups(t *testing.T) {
	s := newTestServer(map[string]testResponse{
		"/user/orgs": {
			data:     []org{{Login: "org-1"}, {Login: "org-2"}},
			nextLink: "/user/orgs?since=2",
			lastLink: "/user/orgs?since=2",
		},
		"/user/orgs?since=2": {data: []org{{Login: "org-3"}}},
		"/user/teams": {
			data: []team{
				{Name: "team-1", Org: org{Login: "org-1"}},
				{Name: "team-2", Org: org{Login: "org-1"}},
			},
			nextLink: "/user/teams?since=2",
			lastLink: "/user/teams?since=2",
		},
		"/user/teams?since=2": {
			data: []team{
				{Name: "team-3", Org: org{Login: "org-1"}},
				{Name: "team-4", Org: org{Login: "org-2"}},
			},
			nextLink: "/user/teams?since=2",
			lastLink: "/user/teams?since=2",
		},
	})
	defer s.Close()

	c := githubConnector{apiURL: s.URL}
	groups, err := c.userGroups(context.Background(), newClient())

	expectNil(t, err)
	expectEquals(t, groups, []string{
		"org-1",
		"org-1:team-1",
		"org-1:team-2",
		"org-1:team-3",
		"org-2",
		"org-2:team-4",
		"org-3",
	})
}

func TestUserGroupsWithoutOrgs(t *testing.T) {
	s := newTestServer(map[string]testResponse{
		"/user/orgs":  {data: []org{}},
		"/user/teams": {data: []team{}},
	})
	defer s.Close()

	c := githubConnector{apiURL: s.URL}
	groups, err := c.userGroups(context.Background(), newClient())

	expectNil(t, err)
	expectEquals(t, len(groups), 0)
}

func TestUserGroupsWithTeamNameFieldConfig(t *testing.T) {
	s := newTestServer(map[string]testResponse{
		"/user/orgs": {
			data: []org{{Login: "org-1"}},
		},
		"/user/teams": {
			data: []team{
				{Name: "Team 1", Slug: "team-1", Org: org{Login: "org-1"}},
			},
		},
	})
	defer s.Close()

	c := githubConnector{apiURL: s.URL, teamNameField: "slug"}
	groups, err := c.userGroups(context.Background(), newClient())

	expectNil(t, err)
	expectEquals(t, groups, []string{
		"org-1",
		"org-1:team-1",
	})
}

func TestUserGroupsWithTeamNameAndSlugFieldConfig(t *testing.T) {
	s := newTestServer(map[string]testResponse{
		"/user/orgs": {
			data: []org{{Login: "org-1"}},
		},
		"/user/teams": {
			data: []team{
				{Name: "Team 1", Slug: "team-1", Org: org{Login: "org-1"}},
			},
		},
	})
	defer s.Close()

	c := githubConnector{apiURL: s.URL, teamNameField: "both"}
	groups, err := c.userGroups(context.Background(), newClient())

	expectNil(t, err)
	expectEquals(t, groups, []string{
		"org-1",
		"org-1:Team 1",
		"org-1:team-1",
	})
}

// tests that the users login is used as their username when they have no username set
func TestUsernameIncludedInFederatedIdentity(t *testing.T) {
	s := newTestServer(map[string]testResponse{
		"/user": {data: user{Login: "some-login", ID: 12345678}},
		"/user/emails": {data: []userEmail{{
			Email:    "some@email.com",
			Verified: true,
			Primary:  true,
		}}},
		"/login/oauth/access_token": {data: map[string]interface{}{
			"access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			"expires_in":   "30",
		}},
		"/user/orgs": {
			data: []org{{Login: "org-1"}},
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	c := githubConnector{apiURL: s.URL, hostName: hostURL.Host, httpClient: newClient()}
	identity, err := c.HandleCallback(connector.Scopes{Groups: true}, nil, req)

	expectNil(t, err)
	expectEquals(t, identity.Username, "some-login")
	expectEquals(t, identity.UserID, "12345678")
	expectEquals(t, 0, len(identity.Groups))

	c = githubConnector{apiURL: s.URL, hostName: hostURL.Host, httpClient: newClient(), loadAllGroups: true}
	identity, err = c.HandleCallback(connector.Scopes{Groups: true}, nil, req)

	expectNil(t, err)
	expectEquals(t, identity.Username, "some-login")
	expectEquals(t, identity.UserID, "12345678")
	expectEquals(t, identity.Groups, []string{"org-1"})
}

func TestLoginUsedAsIDWhenConfigured(t *testing.T) {
	s := newTestServer(map[string]testResponse{
		"/user": {data: user{Login: "some-login", ID: 12345678, Name: "Joe Bloggs"}},
		"/user/emails": {data: []userEmail{{
			Email:    "some@email.com",
			Verified: true,
			Primary:  true,
		}}},
		"/login/oauth/access_token": {data: map[string]interface{}{
			"access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			"expires_in":   "30",
		}},
		"/user/orgs": {
			data: []org{{Login: "org-1"}},
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	c := githubConnector{apiURL: s.URL, hostName: hostURL.Host, httpClient: newClient(), useLoginAsID: true}
	identity, err := c.HandleCallback(connector.Scopes{Groups: true}, nil, req)

	expectNil(t, err)
	expectEquals(t, identity.UserID, "some-login")
	expectEquals(t, identity.Username, "Joe Bloggs")
}

func TestHandleCallbackStoresRefreshToken(t *testing.T) {
	s := newTestServer(map[string]testResponse{
		"/user": {data: user{Login: "some-login", ID: 12345678, Name: "Joe Bloggs", Email: "some@email.com"}},
		"/login/oauth/access_token": {data: map[string]interface{}{
			"access_token":  "access-token",
			"refresh_token": "refresh-token",
			"expires_in":    28800,
		}},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	c := githubConnector{apiURL: s.URL, hostName: hostURL.Host, httpClient: newClient()}
	identity, err := c.HandleCallback(connector.Scopes{OfflineAccess: true}, nil, req)
	expectNil(t, err)

	var data connectorData
	if err := json.Unmarshal(identity.ConnectorData, &data); err != nil {
		t.Fatalf("failed to unmarshal connector data: %v", err)
	}
	expectEquals(t, data.AccessToken, "access-token")
	expectEquals(t, data.RefreshToken, "refresh-token")
	if data.Expiry.IsZero() || !data.Expiry.After(time.Now()) {
		t.Fatalf("expected future token expiry, got %v", data.Expiry)
	}
}

func TestRefreshUsesRefreshToken(t *testing.T) {
	var tokenRefreshCalled atomic.Bool
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/login/oauth/access_token":
			tokenRefreshCalled.Store(true)
			if err := r.ParseForm(); err != nil {
				t.Fatalf("failed to parse token refresh form: %v", err)
			}
			if got := r.Form.Get("grant_type"); got != "refresh_token" {
				t.Fatalf("expected refresh_token grant type, got %q", got)
			}
			if got := r.Form.Get("refresh_token"); got != "old-refresh-token" {
				t.Fatalf("expected old refresh token, got %q", got)
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token":  "new-access-token",
				"refresh_token": "new-refresh-token",
				"expires_in":    28800,
			})
		case "/user":
			if got := r.Header.Get("Authorization"); got != "Bearer new-access-token" {
				t.Fatalf("expected refreshed access token, got %q", got)
			}
			json.NewEncoder(w).Encode(user{Login: "new-login", ID: 12345678, Name: "New User", Email: "new@email.com"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	connData, err := json.Marshal(connectorData{
		AccessToken:  "old-access-token",
		RefreshToken: "old-refresh-token",
		Expiry:       time.Now().Add(-time.Hour),
	})
	expectNil(t, err)

	c := githubConnector{
		apiURL:       s.URL,
		hostName:     hostURL.Host,
		httpClient:   newClient(),
		clientID:     "client-id",
		clientSecret: "client-secret",
	}
	identity, err := c.Refresh(context.Background(), connector.Scopes{OfflineAccess: true}, connector.Identity{ConnectorData: connData})
	expectNil(t, err)

	if !tokenRefreshCalled.Load() {
		t.Fatal("expected refresh token request")
	}
	expectEquals(t, identity.Username, "New User")
	expectEquals(t, identity.PreferredUsername, "new-login")
	expectEquals(t, identity.Email, "new@email.com")

	var data connectorData
	if err := json.Unmarshal(identity.ConnectorData, &data); err != nil {
		t.Fatalf("failed to unmarshal connector data: %v", err)
	}
	expectEquals(t, data.AccessToken, "new-access-token")
	expectEquals(t, data.RefreshToken, "new-refresh-token")
	if data.Expiry.IsZero() || !data.Expiry.After(time.Now()) {
		t.Fatalf("expected future token expiry, got %v", data.Expiry)
	}
}

func TestRefreshWithAccessTokenOnlyConnectorData(t *testing.T) {
	var tokenEndpointCalled atomic.Bool
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/login/oauth/access_token":
			tokenEndpointCalled.Store(true)
			http.Error(w, "unexpected token refresh", http.StatusInternalServerError)
		case "/user":
			if got := r.Header.Get("Authorization"); got != "Bearer old-access-token" {
				t.Fatalf("expected old access token, got %q", got)
			}
			json.NewEncoder(w).Encode(user{Login: "some-login", ID: 12345678, Name: "Some User", Email: "some@email.com"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	connData, err := json.Marshal(connectorData{AccessToken: "old-access-token"})
	expectNil(t, err)

	c := githubConnector{apiURL: s.URL, hostName: hostURL.Host, httpClient: newClient()}
	identity, err := c.Refresh(context.Background(), connector.Scopes{}, connector.Identity{ConnectorData: connData})
	expectNil(t, err)

	if tokenEndpointCalled.Load() {
		t.Fatal("did not expect token refresh request")
	}
	expectEquals(t, identity.Username, "Some User")
	expectEquals(t, identity.PreferredUsername, "some-login")
	expectEquals(t, identity.Email, "some@email.com")
	expectEquals(t, identity.ConnectorData, connData)
}

func TestPreferredEmailDomainConfigured(t *testing.T) {
	ctx := context.Background()
	s := newTestServer(map[string]testResponse{
		"/user": {data: user{Login: "some-login", ID: 12345678, Name: "Joe Bloggs"}},
		"/user/emails": {
			data: []userEmail{
				{
					Email:    "some@email.com",
					Verified: true,
					Primary:  true,
				},
				{
					Email:    "another@email.com",
					Verified: true,
					Primary:  false,
				},
				{
					Email:    "some@preferred-domain.com",
					Verified: true,
					Primary:  false,
				},
				{
					Email:    "another@preferred-domain.com",
					Verified: true,
					Primary:  false,
				},
			},
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	client := newClient()
	c := githubConnector{apiURL: s.URL, hostName: hostURL.Host, httpClient: client, preferredEmailDomain: "preferred-domain.com"}

	u, err := c.user(ctx, client)
	expectNil(t, err)
	expectEquals(t, u.Email, "some@preferred-domain.com")
}

func TestPreferredEmailDomainConfiguredWithGlob(t *testing.T) {
	ctx := context.Background()
	s := newTestServer(map[string]testResponse{
		"/user": {data: user{Login: "some-login", ID: 12345678, Name: "Joe Bloggs"}},
		"/user/emails": {
			data: []userEmail{
				{
					Email:    "some@email.com",
					Verified: true,
					Primary:  true,
				},
				{
					Email:    "another@email.com",
					Verified: true,
					Primary:  false,
				},
				{
					Email:    "some@another.preferred-domain.com",
					Verified: true,
					Primary:  false,
				},
				{
					Email:    "some@sub-domain.preferred-domain.co",
					Verified: true,
					Primary:  false,
				},
			},
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	client := newClient()
	c := githubConnector{apiURL: s.URL, hostName: hostURL.Host, httpClient: client, preferredEmailDomain: "*.preferred-domain.co"}

	u, err := c.user(ctx, client)
	expectNil(t, err)
	expectEquals(t, u.Email, "some@sub-domain.preferred-domain.co")
}

func TestPreferredEmailDomainConfigured_UserHasNoPreferredDomainEmail(t *testing.T) {
	ctx := context.Background()
	s := newTestServer(map[string]testResponse{
		"/user": {data: user{Login: "some-login", ID: 12345678, Name: "Joe Bloggs"}},
		"/user/emails": {
			data: []userEmail{
				{
					Email:    "some@email.com",
					Verified: true,
					Primary:  true,
				},
				{
					Email:    "another@email.com",
					Verified: true,
					Primary:  false,
				},
			},
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	client := newClient()
	c := githubConnector{apiURL: s.URL, hostName: hostURL.Host, httpClient: client, preferredEmailDomain: "preferred-domain.com"}

	u, err := c.user(ctx, client)
	expectNil(t, err)
	expectEquals(t, u.Email, "some@email.com")
}

func TestPreferredEmailDomainNotConfigured(t *testing.T) {
	ctx := context.Background()
	s := newTestServer(map[string]testResponse{
		"/user": {data: user{Login: "some-login", ID: 12345678, Name: "Joe Bloggs"}},
		"/user/emails": {
			data: []userEmail{
				{
					Email:    "some@email.com",
					Verified: true,
					Primary:  true,
				},
				{
					Email:    "another@email.com",
					Verified: true,
					Primary:  false,
				},
				{
					Email:    "some@preferred-domain.com",
					Verified: true,
					Primary:  false,
				},
			},
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	client := newClient()
	c := githubConnector{apiURL: s.URL, hostName: hostURL.Host, httpClient: client}

	u, err := c.user(ctx, client)
	expectNil(t, err)
	expectEquals(t, u.Email, "some@email.com")
}

func TestPreferredEmailDomainConfigured_Error_BothPrimaryAndPreferredDomainEmailNotFound(t *testing.T) {
	ctx := context.Background()
	s := newTestServer(map[string]testResponse{
		"/user": {data: user{Login: "some-login", ID: 12345678, Name: "Joe Bloggs"}},
		"/user/emails": {
			data: []userEmail{
				{
					Email:    "some@email.com",
					Verified: true,
					Primary:  false,
				},
				{
					Email:    "another@email.com",
					Verified: true,
					Primary:  false,
				},
				{
					Email:    "some@preferred-domain.com",
					Verified: true,
					Primary:  false,
				},
			},
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	client := newClient()
	c := githubConnector{apiURL: s.URL, hostName: hostURL.Host, httpClient: client, preferredEmailDomain: "foo.bar"}

	_, err = c.user(ctx, client)
	expectNotNil(t, err, "Email not found error")
	expectEquals(t, err.Error(), "github: user has no verified, primary email or preferred-domain email")
}

func Test_isPreferredEmailDomain(t *testing.T) {
	client := newClient()
	tests := []struct {
		preferredEmailDomain string
		email                string
		expected             bool
	}{
		{
			preferredEmailDomain: "example.com",
			email:                "test@example.com",
			expected:             true,
		},
		{
			preferredEmailDomain: "example.com",
			email:                "test@another.com",
			expected:             false,
		},
		{
			preferredEmailDomain: "*.example.com",
			email:                "test@my.example.com",
			expected:             true,
		},
		{
			preferredEmailDomain: "*.example.com",
			email:                "test@my.another.com",
			expected:             false,
		},
		{
			preferredEmailDomain: "*.example.com",
			email:                "test@my.domain.example.com",
			expected:             false,
		},
		{
			preferredEmailDomain: "*.example.com",
			email:                "test@sub.domain.com",
			expected:             false,
		},
		{
			preferredEmailDomain: "*.*.example.com",
			email:                "test@sub.my.example.com",
			expected:             true,
		},
		{
			preferredEmailDomain: "*.*.example.com",
			email:                "test@a.my.google.com",
			expected:             false,
		},
	}
	for _, test := range tests {
		t.Run(test.preferredEmailDomain, func(t *testing.T) {
			c := githubConnector{apiURL: "apiURL", hostName: "github.com", httpClient: client, preferredEmailDomain: test.preferredEmailDomain}
			_, domainPart, _ := strings.Cut(test.email, "@")
			res := c.isPreferredEmailDomain(domainPart)

			expectEquals(t, res, test.expected)
		})
	}
}

func Test_Open_PreferredDomainConfig(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	tests := []struct {
		preferredEmailDomain string
		email                string
		expected             error
	}{
		{
			preferredEmailDomain: "example.com",
			expected:             nil,
		},
		{
			preferredEmailDomain: "*.example.com",
			expected:             nil,
		},
		{
			preferredEmailDomain: "*.*.example.com",
			expected:             nil,
		},
		{
			preferredEmailDomain: "example.*",
			expected:             errors.New("invalid PreferredEmailDomain: glob pattern cannot end with \"*\""),
		},
	}
	for _, test := range tests {
		t.Run(test.preferredEmailDomain, func(t *testing.T) {
			c := Config{
				PreferredEmailDomain: test.preferredEmailDomain,
			}
			_, err := c.Open("id", log)

			expectEquals(t, err, test.expected)
		})
	}
}

func TestGetSendsAPIVersionHeader(t *testing.T) {
	var gotHeader string
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-GitHub-Api-Version")
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]org{})
	}))
	defer s.Close()

	var result []org
	_, err := get(context.Background(), newClient(), s.URL+"/user/orgs", &result)
	expectNil(t, err)
	expectEquals(t, gotHeader, githubAPIVersion)
}

func newTestServer(responses map[string]testResponse) *httptest.Server {
	var s *httptest.Server
	s = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := responses[r.RequestURI]
		linkParts := make([]string, 0)
		if response.nextLink != "" {
			linkParts = append(linkParts, fmt.Sprintf("<%s%s>; rel=\"next\"", s.URL, response.nextLink))
		}
		if response.lastLink != "" {
			linkParts = append(linkParts, fmt.Sprintf("<%s%s>; rel=\"last\"", s.URL, response.lastLink))
		}
		if len(linkParts) > 0 {
			w.Header().Add("Link", strings.Join(linkParts, ", "))
		}
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response.data)
	}))
	return s
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

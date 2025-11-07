package gitlab

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/dexidp/dex/connector"
)

func TestUserGroups(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/api/v4/groups": []group{
			{
				ID:       1,
				Name:     "team-1",
				FullName: "team-1",
				Path:     "team-1",
				FullPath: "team-1",
			},
			{
				ID:       2,
				Name:     "team-2",
				FullName: "team-2",
				Path:     "team-2",
				FullPath: "team-2",
			},
		},
		"/api/v4/groups/1/members/all/1": map[string]interface{}{
			"access_level": 50,
		},
		"/api/v4/groups/2/members/all/1": map[string]interface{}{
			"access_level": 50,
		},
	})
	defer s.Close()

	c := gitlabConnector{baseURL: s.URL}
	groups, err := c.getGroups(context.Background(), newClient(), true, "joebloggs", 1)

	expectNil(t, err)
	expectEquals(t, groups, []string{
		"team-1",
		"team-2",
	})
}

func TestUserGroupsWithFiltering(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/api/v4/groups": []group{
			{
				ID:       1,
				Name:     "team-1",
				FullName: "team-1",
				Path:     "team-1",
				FullPath: "team-1",
			},
			{
				ID:       2,
				Name:     "team-2",
				FullName: "team-2",
				Path:     "team-2",
				FullPath: "team-2",
			},
		},
		"/api/v4/groups/1/members/all/1": map[string]interface{}{
			"access_level": 50,
		},
		"/api/v4/groups/2/members/all/1": map[string]interface{}{
			"access_level": 50,
		},
	})
	defer s.Close()

	c := gitlabConnector{baseURL: s.URL, groups: []string{"team-1"}}
	groups, err := c.getGroups(context.Background(), newClient(), true, "joebloggs", 1)

	expectNil(t, err)
	expectEquals(t, groups, []string{
		"team-1",
	})
}

func TestUserGroupsWithoutOrgs(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/api/v4/groups": []group{},
	})
	defer s.Close()

	c := gitlabConnector{baseURL: s.URL}
	groups, err := c.getGroups(context.Background(), newClient(), true, "joebloggs", 1)

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
		"/api/v4/groups": []group{
			{
				ID:       1,
				Name:     "team-1",
				FullName: "team-1",
				Path:     "team-1",
				FullPath: "team-1",
			},
		},
		"/api/v4/groups/1/members/all/12345678": map[string]interface{}{
			"access_level": 50,
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	c := gitlabConnector{baseURL: s.URL, httpClient: newClient()}
	identity, err := c.HandleCallback(connector.Scopes{Groups: false}, req)

	expectNil(t, err)
	expectEquals(t, identity.Username, "some@email.com")
	expectEquals(t, identity.UserID, "12345678")
	expectEquals(t, 0, len(identity.Groups))

	c = gitlabConnector{baseURL: s.URL, httpClient: newClient()}
	identity, err = c.HandleCallback(connector.Scopes{Groups: true}, req)

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
		"/api/v4/groups": []group{
			{
				ID:       1,
				Name:     "team-1",
				FullName: "team-1",
				Path:     "team-1",
				FullPath: "team-1",
			},
		},
		"/api/v4/groups/1/members/all/1": map[string]interface{}{
			"access_level": 50,
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	c := gitlabConnector{baseURL: s.URL, httpClient: newClient(), useLoginAsID: true}
	identity, err := c.HandleCallback(connector.Scopes{Groups: true}, req)

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
		"/api/v4/groups": []group{
			{
				ID:       1,
				Name:     "team-1",
				FullName: "team-1",
				Path:     "team-1",
				FullPath: "team-1",
			},
		},
		"/api/v4/groups/1/members/all/12345678": map[string]interface{}{
			"access_level": 50,
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	c := gitlabConnector{baseURL: s.URL, httpClient: newClient(), groups: []string{"team-1"}}
	identity, err := c.HandleCallback(connector.Scopes{Groups: true}, req)

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
		"/api/v4/groups": []group{
			{
				ID:       1,
				Name:     "team-1",
				FullName: "team-1",
				Path:     "team-1",
				FullPath: "team-1",
			},
		},
		"/api/v4/groups/1/members/all/12345678": map[string]interface{}{
			"access_level": 50,
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	c := gitlabConnector{baseURL: s.URL, httpClient: newClient(), groups: []string{"team-2"}}
	_, err = c.HandleCallback(connector.Scopes{Groups: true}, req)

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
		"/api/v4/groups": []group{
			{
				ID:       1,
				Name:     "team-1",
				FullName: "team-1",
				Path:     "team-1",
				FullPath: "team-1",
			},
		},
		"/api/v4/groups/1/members/all/12345678": map[string]interface{}{
			"access_level": 50,
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

	identity, err := c.HandleCallback(connector.Scopes{OfflineAccess: true}, req)
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
		"/api/v4/groups": []group{
			{
				ID:       1,
				Name:     "team-1",
				FullName: "team-1",
				Path:     "team-1",
				FullPath: "team-1",
			},
		},
		"/api/v4/groups/1/members/all/12345678": map[string]interface{}{
			"access_level": 50,
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
		"/api/v4/groups": []group{
			{
				ID:       1,
				Name:     "ops",
				FullName: "ops",
				Path:     "ops",
				FullPath: "ops",
			},
			{
				ID:       2,
				Name:     "dev",
				FullName: "dev",
				Path:     "dev",
				FullPath: "dev",
			},
			{
				ID:       3,
				Name:     "ops/project",
				FullName: "ops/project",
				Path:     "ops/project",
				FullPath: "ops/project",
			},
			{
				ID:       4,
				Name:     "dev/project1",
				FullName: "dev/project1",
				Path:     "dev/project1",
				FullPath: "dev/project1",
			},
			{
				ID:       5,
				Name:     "dev/project2",
				FullName: "dev/project2",
				Path:     "dev/project2",
				FullPath: "dev/project2",
			},
		},
		"/api/v4/groups/1/members/all/12345678": map[string]interface{}{
			"access_level": 50,
		},
		"/api/v4/groups/2/members/all/12345678": map[string]interface{}{
			"access_level": 30,
		},
		"/api/v4/groups/3/members/all/12345678": map[string]interface{}{
			"access_level": 50,
		},
		"/api/v4/groups/4/members/all/12345678": map[string]interface{}{
			"access_level": 40,
		},
		"/api/v4/groups/5/members/all/12345678": map[string]interface{}{
			"access_level": 30,
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	c := gitlabConnector{baseURL: s.URL, httpClient: newClient(), getGroupsPermission: true}
	identity, err := c.HandleCallback(connector.Scopes{Groups: true}, req)
	expectNil(t, err)

	expectEquals(t, identity.Groups, []string{
		"ops",
		"ops:owner",
		"dev",
		"dev:developer",
		"ops/project",
		"ops/project:owner",
		"dev/project1",
		"dev/project1:maintainer",
		"dev/project2",
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

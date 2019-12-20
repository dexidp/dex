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
	_, err = c.HandleCallback(connector.Scopes{Groups: true}, req)

	expectNotNil(t, err, "HandleCallback error")
	expectEquals(t, err.Error(), "gitlab: get groups: gitlab: user \"joebloggs\" is not in any of the required groups")
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

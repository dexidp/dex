package github

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/concourse/dex/connector"
)

func TestUserGroups(t *testing.T) {

	orgs := []org{
		{Login: "org-1"},
		{Login: "org-2"},
		{Login: "org-3"},
	}

	teams := []team{
		{Name: "Team 1", Slug: "team-1", Org: org{Login: "org-1"}},
		{Name: "Team 2", Slug: "team-2", Org: org{Login: "org-1"}},
		{Name: "Team 3", Slug: "team-3", Org: org{Login: "org-1"}},
		{Name: "Team 4", Slug: "team-4", Org: org{Login: "org-2"}},
	}

	s := newTestServer(map[string]interface{}{
		"/user/orgs":  orgs,
		"/user/teams": teams,
	})

	connector := githubConnector{apiURL: s.URL}
	groups, err := connector.userGroups(context.Background(), newClient())

	expectNil(t, err)
	expectEquals(t, groups, []string{
		"org-1:Team 1",
		"org-1:team-1",
		"org-1:Team 2",
		"org-1:team-2",
		"org-1:Team 3",
		"org-1:team-3",
		"org-2:Team 4",
		"org-2:team-4",
		"org-3",
	})

	s.Close()
}

func TestUserGroupsWithoutOrgs(t *testing.T) {

	s := newTestServer(map[string]interface{}{
		"/user/orgs":  []org{},
		"/user/teams": []team{},
	})

	connector := githubConnector{apiURL: s.URL}
	groups, err := connector.userGroups(context.Background(), newClient())

	expectNil(t, err)
	expectEquals(t, len(groups), 0)

	s.Close()
}

func TestUsernameIncludedInFederatedIdentity(t *testing.T) {

	s := newTestServer(map[string]interface{}{
		"/user": user{Login: "some-login"},
		"/user/emails": []userEmail{{
			Email:    "some@email.com",
			Verified: true,
			Primary:  true,
		}},
		"/login/oauth/access_token": map[string]interface{}{
			"access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			"expires_in":   "30",
		},
	})

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	githubConnector := githubConnector{apiURL: s.URL, hostName: hostURL.Host, httpClient: newClient()}
	identity, err := githubConnector.HandleCallback(connector.Scopes{}, req)

	expectNil(t, err)
	expectEquals(t, identity.Username, "some-login")

	s.Close()
}

func newTestServer(responses map[string]interface{}) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responses[r.URL.Path])
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

func expectEquals(t *testing.T, a interface{}, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Expected %+v to equal %+v", a, b)
	}
}

package github

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

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
		"org-1:team-1",
		"org-1:team-2",
		"org-1:team-3",
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

func TestUsernameIncludedInFederatedIdentity(t *testing.T) {

	s := newTestServer(map[string]testResponse{
		"/user": {data: user{Login: "some-login"}},
		"/user/emails": {data: []userEmail{{
			Email:    "some@email.com",
			Verified: true,
			Primary:  true,
		}}},
		"/login/oauth/access_token": {data: map[string]interface{}{
			"access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			"expires_in":   "30",
		}},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	c := githubConnector{apiURL: s.URL, hostName: hostURL.Host, httpClient: newClient()}
	identity, err := c.HandleCallback(connector.Scopes{}, req)

	expectNil(t, err)
	expectEquals(t, identity.Username, "some-login")

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

func expectEquals(t *testing.T, a interface{}, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Expected %+v to equal %+v", a, b)
	}
}

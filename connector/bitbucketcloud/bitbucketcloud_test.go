package bitbucketcloud

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
	teamsResponse := userTeamsResponse{
		pagedResponse: pagedResponse{
			Size:    3,
			Page:    1,
			PageLen: 10,
		},
		Values: []team{
			{Name: "team-1"},
			{Name: "team-2"},
			{Name: "team-3"},
		},
	}

	s := newTestServer(map[string]interface{}{
		"/teams?role=member": teamsResponse,
	})

	connector := bitbucketConnector{apiURL: s.URL}
	groups, err := connector.userTeams(context.Background(), newClient())

	expectNil(t, err)
	expectEquals(t, groups, []string{
		"team-1",
		"team-2",
		"team-3",
	})

	s.Close()
}

func TestUserWithoutTeams(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/teams?role=member": userTeamsResponse{},
	})

	connector := bitbucketConnector{apiURL: s.URL}
	groups, err := connector.userTeams(context.Background(), newClient())

	expectNil(t, err)
	expectEquals(t, len(groups), 0)

	s.Close()
}

func TestUsernameIncludedInFederatedIdentity(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/user": user{Username: "some-login"},
		"/user/emails": userEmailResponse{
			pagedResponse: pagedResponse{
				Size:    1,
				Page:    1,
				PageLen: 10,
			},
			Values: []userEmail{{
				Email:       "some@email.com",
				IsConfirmed: true,
				IsPrimary:   true,
			}},
		},
		"/site/oauth2/access_token": map[string]interface{}{
			"access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			"expires_in":   "30",
		},
	})

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	bitbucketConnector := bitbucketConnector{apiURL: s.URL, hostName: hostURL.Host, httpClient: newClient()}
	identity, err := bitbucketConnector.HandleCallback(connector.Scopes{}, req)

	expectNil(t, err)
	expectEquals(t, identity.Username, "some-login")

	s.Close()
}

func newTestServer(responses map[string]interface{}) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responses[r.URL.String()])
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
		t.Fatalf("Expected %+v to equal nil", a)
	}
}

func expectEquals(t *testing.T, a interface{}, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("Expected %+v to equal %+v", a, b)
	}
}

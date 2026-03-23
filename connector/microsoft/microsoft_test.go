package microsoft

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/dexidp/dex/connector"
)

type testResponse struct {
	data interface{}
}

const (
	tenant   = "9b1c3439-a67e-4e92-bb0d-0571d44ca965"
	clientID = "a115ebf3-6020-4384-8eb1-c0c42e667b6f"
)

var dummyToken = testResponse{data: map[string]interface{}{
	"access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
	"expires_in":   "30",
}}

func TestLoginURL(t *testing.T) {
	testURL := "https://test.com"
	testState := "some-state"

	conn := microsoftConnector{
		apiURL:      testURL,
		graphURL:    testURL,
		redirectURI: testURL,
		clientID:    clientID,
		tenant:      tenant,
	}

	loginURL, _, _ := conn.LoginURL(connector.Scopes{}, conn.redirectURI, testState)

	parsedLoginURL, _ := url.Parse(loginURL)
	queryParams := parsedLoginURL.Query()

	expectEquals(t, parsedLoginURL.Path, "/"+tenant+"/oauth2/v2.0/authorize")
	expectEquals(t, queryParams.Get("client_id"), clientID)
	expectEquals(t, queryParams.Get("redirect_uri"), testURL)
	expectEquals(t, queryParams.Get("response_type"), "code")
	expectEquals(t, queryParams.Get("scope"), "user.read")
	expectEquals(t, queryParams.Get("state"), testState)
	expectEquals(t, queryParams.Get("prompt"), "")
	expectEquals(t, queryParams.Get("domain_hint"), "")
}

func TestLoginURLWithOptions(t *testing.T) {
	testURL := "https://test.com"
	promptType := "consent"
	domainHint := "domain.hint"

	conn := microsoftConnector{
		apiURL:      testURL,
		graphURL:    testURL,
		redirectURI: testURL,
		clientID:    clientID,
		tenant:      tenant,

		promptType: promptType,
		domainHint: domainHint,
	}

	loginURL, _, _ := conn.LoginURL(connector.Scopes{}, conn.redirectURI, "some-state")

	parsedLoginURL, _ := url.Parse(loginURL)
	queryParams := parsedLoginURL.Query()

	expectEquals(t, queryParams.Get("prompt"), promptType)
	expectEquals(t, queryParams.Get("domain_hint"), domainHint)
}

func TestUserIdentityFromGraphAPI(t *testing.T) {
	s := newTestServer(map[string]testResponse{
		"/v1.0/me?$select=id,displayName,userPrincipalName": {
			data: user{ID: "S56767889", Name: "Jane Doe", Email: "jane.doe@example.com"},
		},
		"/" + tenant + "/oauth2/v2.0/token": dummyToken,
	})
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL, nil)

	c := microsoftConnector{apiURL: s.URL, graphURL: s.URL, tenant: tenant}
	identity, err := c.HandleCallback(connector.Scopes{Groups: false}, nil, req)
	expectNil(t, err)
	expectEquals(t, identity.Username, "Jane Doe")
	expectEquals(t, identity.UserID, "S56767889")
	expectEquals(t, identity.PreferredUsername, "")
	expectEquals(t, identity.Email, "jane.doe@example.com")
	expectEquals(t, identity.EmailVerified, true)
	expectEquals(t, len(identity.Groups), 0)
}

func TestUserGroupsFromGraphAPI(t *testing.T) {
	s := newTestServer(map[string]testResponse{
		"/v1.0/me?$select=id,displayName,userPrincipalName": {data: user{}},
		"/v1.0/me/getMemberGroups": {data: map[string]interface{}{
			"value": []string{"a", "b"},
		}},
		"/" + tenant + "/oauth2/v2.0/token": dummyToken,
	})
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL, nil)

	c := microsoftConnector{apiURL: s.URL, graphURL: s.URL, tenant: tenant}
	identity, err := c.HandleCallback(connector.Scopes{Groups: true}, nil, req)
	expectNil(t, err)
	expectEquals(t, identity.Groups, []string{"a", "b"})
}

func TestUserNotInRequiredGroupFromGraphAPI(t *testing.T) {
	s := newTestServer(map[string]testResponse{
		"/v1.0/me?$select=id,displayName,userPrincipalName": {
			data: user{ID: "user-id-123", Name: "Jane Doe", Email: "jane.doe@example.com"},
		},
		// The user is a member of groups "c" and "d", but the connector only
		// allows group "a" — so the user should be denied.
		"/v1.0/me/getMemberGroups": {data: map[string]interface{}{
			"value": []string{"c", "d"},
		}},
		"/" + tenant + "/oauth2/v2.0/token": dummyToken,
	})
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL, nil)

	c := microsoftConnector{
		apiURL:   s.URL,
		graphURL: s.URL,
		tenant:   tenant,
		groups:   []string{"a"},
	}
	_, err := c.HandleCallback(connector.Scopes{Groups: true}, nil, req)
	if err == nil {
		t.Fatal("expected error when user is not in any required group, got nil")
	}

	var groupsErr *connector.UserNotInRequiredGroupsError
	if !errors.As(err, &groupsErr) {
		t.Errorf("expected *connector.UserNotInRequiredGroupsError, got %T: %v", err, err)
	}
}

func newTestServer(responses map[string]testResponse) *httptest.Server {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response, found := responses[r.RequestURI]
		if !found {
			fmt.Fprintf(os.Stderr, "Mock response for %q not found\n", r.RequestURI)
			http.NotFound(w, r)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response.data)
	}))
	return s
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

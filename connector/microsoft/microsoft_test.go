package microsoft

import (
	"encoding/json"
	"fmt"
	"log/slog"
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

	loginURL, _ := conn.LoginURL(connector.Scopes{}, conn.redirectURI, testState)

	parsedLoginURL, _ := url.Parse(loginURL)
	queryParams := parsedLoginURL.Query()

	expectEquals(t, parsedLoginURL.Path, "/"+tenant+"/oauth2/v2.0/authorize")
	expectEquals(t, queryParams.Get("client_id"), clientID)
	expectEquals(t, queryParams.Get("redirect_uri"), testURL)
	expectEquals(t, queryParams.Get("response_type"), "code")
	expectEquals(t, queryParams.Get("scope"), "openid https://graph.microsoft.com/.default")
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

	loginURL, _ := conn.LoginURL(connector.Scopes{}, conn.redirectURI, "some-state")

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
	identity, err := c.HandleCallback(connector.Scopes{Groups: false}, req)
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
		"/v1.0/me/memberOf/microsoft.graph.group?$select=displayName,id": {data: map[string]interface{}{
			"value": []group{{Name: "a", Id: "1"}, {Name: "b", Id: "2"}},
		}},
		"/" + tenant + "/oauth2/v2.0/token": dummyToken,
	})
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL, nil)

	c := microsoftConnector{apiURL: s.URL, graphURL: s.URL, tenant: tenant, logger: slog.Default(), groupNameFormat: GroupName}
	identity, err := c.HandleCallback(connector.Scopes{Groups: true}, req)
	expectNil(t, err)
	expectEquals(t, identity.Groups, []string{"a", "b"})
}

func TestUserGroupsWithGroupIDFormat(t *testing.T) {
	s := newTestServer(map[string]testResponse{
		"/v1.0/me?$select=id,displayName,userPrincipalName": {data: user{}},
		"/v1.0/me/memberOf/microsoft.graph.group?$select=displayName,id": {data: map[string]interface{}{
			"value": []group{{Name: "GroupA", Id: "id-1"}, {Name: "GroupB", Id: "id-2"}},
		}},
		"/" + tenant + "/oauth2/v2.0/token": dummyToken,
	})
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL, nil)

	c := microsoftConnector{apiURL: s.URL, graphURL: s.URL, tenant: tenant, logger: slog.Default(), groupNameFormat: GroupID}
	identity, err := c.HandleCallback(connector.Scopes{Groups: true}, req)
	expectNil(t, err)
	expectEquals(t, identity.Groups, []string{"id-1", "id-2"})
}

func TestLoginURLWithCustomScopes(t *testing.T) {
	testURL := "https://test.com"
	testState := "some-state"
	customScopes := []string{"custom.scope1", "custom.scope2"}

	conn := microsoftConnector{
		apiURL:      testURL,
		graphURL:    testURL,
		redirectURI: testURL,
		clientID:    clientID,
		tenant:      tenant,
		scopes:      customScopes,
	}

	loginURL, _ := conn.LoginURL(connector.Scopes{}, conn.redirectURI, testState)

	parsedLoginURL, _ := url.Parse(loginURL)
	queryParams := parsedLoginURL.Query()

	// Custom scopes should be used, plus the default scope is always appended
	expectEquals(t, queryParams.Get("scope"), "custom.scope1 custom.scope2 https://graph.microsoft.com/.default")
}

func TestLoginURLWithOfflineAccess(t *testing.T) {
	testURL := "https://test.com"
	testState := "some-state"

	conn := microsoftConnector{
		apiURL:      testURL,
		graphURL:    testURL,
		redirectURI: testURL,
		clientID:    clientID,
		tenant:      tenant,
	}

	loginURL, _ := conn.LoginURL(connector.Scopes{OfflineAccess: true}, conn.redirectURI, testState)

	parsedLoginURL, _ := url.Parse(loginURL)
	queryParams := parsedLoginURL.Query()

	expectEquals(t, queryParams.Get("scope"), "openid https://graph.microsoft.com/.default offline_access")
}

func TestUserGroupsWithWhitelist(t *testing.T) {
	s := newTestServer(map[string]testResponse{
		"/v1.0/me?$select=id,displayName,userPrincipalName": {data: user{ID: "user123"}},
		"/v1.0/me/memberOf/microsoft.graph.group?$select=displayName,id": {data: map[string]interface{}{
			"value": []group{{Name: "allowed-group", Id: "1"}, {Name: "other-group", Id: "2"}},
		}},
		"/" + tenant + "/oauth2/v2.0/token": dummyToken,
	})
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL, nil)

	c := microsoftConnector{
		apiURL:               s.URL,
		graphURL:             s.URL,
		tenant:               tenant,
		logger:               slog.Default(),
		groupNameFormat:      GroupName,
		groups:               []string{"allowed-group"},
		useGroupsAsWhitelist: true,
	}
	identity, err := c.HandleCallback(connector.Scopes{Groups: true}, req)
	expectNil(t, err)
	// Only the whitelisted group should be returned
	expectEquals(t, identity.Groups, []string{"allowed-group"})
}

func TestUserGroupsNotInRequiredGroups(t *testing.T) {
	s := newTestServer(map[string]testResponse{
		"/v1.0/me?$select=id,displayName,userPrincipalName": {data: user{ID: "user123"}},
		"/v1.0/me/memberOf/microsoft.graph.group?$select=displayName,id": {data: map[string]interface{}{
			"value": []group{{Name: "some-group", Id: "1"}},
		}},
		"/" + tenant + "/oauth2/v2.0/token": dummyToken,
	})
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL, nil)

	c := microsoftConnector{
		apiURL:          s.URL,
		graphURL:        s.URL,
		tenant:          tenant,
		logger:          slog.Default(),
		groupNameFormat: GroupName,
		groups:          []string{"required-group"}, // User is not in this group
	}
	_, err := c.HandleCallback(connector.Scopes{Groups: true}, req)
	// Should fail because user is not in required group
	if err == nil {
		t.Error("Expected error when user is not in required groups")
	}
}

func TestUserGroupsInRequiredGroups(t *testing.T) {
	s := newTestServer(map[string]testResponse{
		"/v1.0/me?$select=id,displayName,userPrincipalName": {data: user{ID: "user123"}},
		"/v1.0/me/memberOf/microsoft.graph.group?$select=displayName,id": {data: map[string]interface{}{
			"value": []group{{Name: "required-group", Id: "1"}, {Name: "other-group", Id: "2"}},
		}},
		"/" + tenant + "/oauth2/v2.0/token": dummyToken,
	})
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL, nil)

	c := microsoftConnector{
		apiURL:          s.URL,
		graphURL:        s.URL,
		tenant:          tenant,
		logger:          slog.Default(),
		groupNameFormat: GroupName,
		groups:          []string{"required-group"},
	}
	identity, err := c.HandleCallback(connector.Scopes{Groups: true}, req)
	expectNil(t, err)
	// All groups should be returned (not filtered) when useGroupsAsWhitelist is false
	expectEquals(t, identity.Groups, []string{"required-group", "other-group"})
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

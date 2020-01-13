package microsoft

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/dexidp/dex/connector"
)

type testResponse struct {
	data interface{}
}

const tenant = "9b1c3439-a67e-4e92-bb0d-0571d44ca965"

var dummyToken = testResponse{data: map[string]interface{}{
	"access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
	"expires_in":   "30",
}}

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
		"/v1.0/me/getMemberGroups": {data: map[string]interface{}{
			"value": []string{"a", "b"},
		}},
		"/" + tenant + "/oauth2/v2.0/token": dummyToken,
	})
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL, nil)

	c := microsoftConnector{apiURL: s.URL, graphURL: s.URL, tenant: tenant}
	identity, err := c.HandleCallback(connector.Scopes{Groups: true}, req)
	expectNil(t, err)
	expectEquals(t, identity.Groups, []string{"a", "b"})
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

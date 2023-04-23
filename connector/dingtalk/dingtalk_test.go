package dingtalk

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/dexidp/dex/connector"
)

// tests that the Mobile is used as their email when they have no email address
func TestUsernameIdentity(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/v1.0/contact/users/me": dingtalkUser{UnionID: "1234", Email: "", Mobile: "138123xxxxx"},
		"/v1.0/oauth2/userAccessToken": map[string]interface{}{
			"accessToken":  "xxxxxx",
			"refreshToken": "xxxxxx",
			"expiresIn":    "30",
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	c := dingtalkConnector{baseURL: s.URL, httpClient: newClient()}
	identity, err := c.HandleCallback(connector.Scopes{}, req)

	expectNil(t, err)
	expectEquals(t, identity.Email, "138123xxxxx")
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

func expectEquals(t *testing.T, a interface{}, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Expected %+v to equal %+v", a, b)
	}
}

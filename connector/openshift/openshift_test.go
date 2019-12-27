package openshift

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/dexidp/dex/connector"

	"github.com/dexidp/dex/storage/kubernetes/k8sapi"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

func TestOpen(t *testing.T) {
	s := newTestServer(map[string]interface{}{})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	_, err = http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	c := Config{
		Issuer:       s.URL,
		ClientID:     "testClientId",
		ClientSecret: "testClientSecret",
		RedirectURI:  "https://localhost/callback",
		InsecureCA:   true,
	}

	logger := logrus.New()

	oconfig, err := c.Open("id", logger)

	oc, ok := oconfig.(*openshiftConnector)

	expectNil(t, err)
	expectEquals(t, ok, true)
	expectEquals(t, oc.apiURL, s.URL)
	expectEquals(t, oc.clientID, "testClientId")
	expectEquals(t, oc.clientSecret, "testClientSecret")
	expectEquals(t, oc.redirectURI, "https://localhost/callback")
	expectEquals(t, oc.oauth2Config.Endpoint.AuthURL, fmt.Sprintf("%s/oauth/authorize", s.URL))
	expectEquals(t, oc.oauth2Config.Endpoint.TokenURL, fmt.Sprintf("%s/oauth/token", s.URL))
}

func TestGetUser(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/apis/user.openshift.io/v1/users/~": user{
			ObjectMeta: k8sapi.ObjectMeta{
				Name: "jdoe",
			},
			FullName: "John Doe",
			Groups:   []string{"users"},
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	_, err = http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	h, err := newHTTPClient(true, "")

	expectNil(t, err)

	oc := openshiftConnector{apiURL: s.URL, httpClient: h}
	u, err := oc.user(context.Background(), h)

	expectNil(t, err)
	expectEquals(t, u.Name, "jdoe")
	expectEquals(t, u.FullName, "John Doe")
	expectEquals(t, len(u.Groups), 1)
}

func TestVerifySingleGroupFn(t *testing.T) {
	allowedGroups := []string{"users"}
	groupMembership := []string{"users", "org1"}

	validGroupMembership := validateAllowedGroups(groupMembership, allowedGroups)

	expectEquals(t, validGroupMembership, true)
}

func TestVerifySingleGroupFailureFn(t *testing.T) {
	allowedGroups := []string{"admins"}
	groupMembership := []string{"users"}

	validGroupMembership := validateAllowedGroups(groupMembership, allowedGroups)

	expectEquals(t, validGroupMembership, false)
}

func TestVerifyMultipleGroupFn(t *testing.T) {
	allowedGroups := []string{"users", "admins"}
	groupMembership := []string{"users", "org1"}

	validGroupMembership := validateAllowedGroups(groupMembership, allowedGroups)

	expectEquals(t, validGroupMembership, true)
}

func TestVerifyGroup(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/apis/user.openshift.io/v1/users/~": user{
			ObjectMeta: k8sapi.ObjectMeta{
				Name: "jdoe",
			},
			FullName: "John Doe",
			Groups:   []string{"users"},
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	_, err = http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	h, err := newHTTPClient(true, "")

	expectNil(t, err)

	oc := openshiftConnector{apiURL: s.URL, httpClient: h}
	u, err := oc.user(context.Background(), h)

	expectNil(t, err)
	expectEquals(t, u.Name, "jdoe")
	expectEquals(t, u.FullName, "John Doe")
	expectEquals(t, len(u.Groups), 1)
}

func TestCallbackIdentity(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/apis/user.openshift.io/v1/users/~": user{
			ObjectMeta: k8sapi.ObjectMeta{
				Name: "jdoe",
				UID:  "12345",
			},
			FullName: "John Doe",
			Groups:   []string{"users"},
		},
		"/oauth/token": map[string]interface{}{
			"access_token": "oRzxVjCnohYRHEYEhZshkmakKmoyVoTjfUGC",
			"expires_in":   "30",
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	h, err := newHTTPClient(true, "")

	expectNil(t, err)

	oc := openshiftConnector{apiURL: s.URL, httpClient: h, oauth2Config: &oauth2.Config{
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/oauth/authorize", s.URL),
			TokenURL: fmt.Sprintf("%s/oauth/token", s.URL),
		},
	}}
	identity, err := oc.HandleCallback(connector.Scopes{Groups: true}, req)

	expectNil(t, err)
	expectEquals(t, identity.UserID, "12345")
	expectEquals(t, identity.Username, "jdoe")
	expectEquals(t, identity.PreferredUsername, "jdoe")
	expectEquals(t, len(identity.Groups), 1)
	expectEquals(t, identity.Groups[0], "users")
}

func newTestServer(responses map[string]interface{}) *httptest.Server {
	var s *httptest.Server
	s = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responses["/.well-known/oauth-authorization-server"] = map[string]interface{}{
			"issuer":                           s.URL,
			"authorization_endpoint":           fmt.Sprintf("%s/oauth/authorize", s.URL),
			"token_endpoint":                   fmt.Sprintf("%s/oauth/token", s.URL),
			"scopes_supported":                 []string{"user:full", "user:info", "user:check-access", "user:list-scoped-projects", "user:list-projects"},
			"response_types_supported":         []string{"token", "code"},
			"grant_types_supported":            []string{"authorization_code", "implicit"},
			"code_challenge_methods_supported": []string{"plain", "S256"},
		}

		response := responses[r.RequestURI]
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
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

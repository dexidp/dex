package gitea

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/dexidp/dex/connector"
)

// tests that the email is used as their username when they have no username set
func TestUsernameIncludedInFederatedIdentity(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/api/v1/user": giteaUser{Email: "some@email.com", ID: 12345678},
		"/login/oauth/access_token": map[string]interface{}{
			"access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			"expires_in":   "30",
		},
	})
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	expectNil(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	expectNil(t, err)

	c := giteaConnector{baseURL: s.URL, httpClient: newClient()}
	identity, err := c.HandleCallback(connector.Scopes{}, nil, req)

	expectNil(t, err)
	expectEquals(t, identity.Username, "some@email.com")
	expectEquals(t, identity.UserID, "12345678")

	c = giteaConnector{baseURL: s.URL, httpClient: newClient()}
	identity, err = c.HandleCallback(connector.Scopes{}, nil, req)

	expectNil(t, err)
	expectEquals(t, identity.Username, "some@email.com")
	expectEquals(t, identity.UserID, "12345678")
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

// TestRefreshUpstreamRejection verifies that a refresh Gitea refuses keeps
// the upstream's OAuth2 error, so the server can tell a rejected refresh
// token apart from other failures.
func TestRefreshUpstreamRejection(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
	}))
	defer s.Close()

	c := giteaConnector{baseURL: s.URL, httpClient: newClient()}
	connData, err := json.Marshal(connectorData{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		Expiry:       time.Now().Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.Refresh(t.Context(), connector.Scopes{OfflineAccess: true}, connector.Identity{ConnectorData: connData})

	var retrieveErr *oauth2.RetrieveError
	if !errors.As(err, &retrieveErr) || retrieveErr.ErrorCode != "invalid_grant" {
		t.Fatalf("expected the upstream invalid_grant in the error chain, got %v", err)
	}
}

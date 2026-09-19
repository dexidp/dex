package openshift

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/httpclient"
	"github.com/dexidp/dex/storage/kubernetes/k8sapi"
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

	logger := slog.New(slog.DiscardHandler)

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

	h, err := httpclient.NewHTTPClient(nil, true)

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

	h, err := httpclient.NewHTTPClient(nil, true)

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

	h, err := httpclient.NewHTTPClient(nil, true)

	expectNil(t, err)

	oc := openshiftConnector{apiURL: s.URL, httpClient: h, oauth2Config: &oauth2.Config{
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/oauth/authorize", s.URL),
			TokenURL: fmt.Sprintf("%s/oauth/token", s.URL),
		},
	}}
	identity, err := oc.HandleCallback(connector.Scopes{Groups: true}, nil, req)

	expectNil(t, err)
	expectEquals(t, identity.UserID, "12345")
	expectEquals(t, identity.Username, "jdoe")
	expectEquals(t, identity.PreferredUsername, "jdoe")
	expectEquals(t, identity.Email, "jdoe")
	expectEquals(t, len(identity.Groups), 1)
	expectEquals(t, identity.Groups[0], "users")
}

func TestRefreshIdentity(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		usersURLPath: user{
			ObjectMeta: k8sapi.ObjectMeta{
				Name: "jdoe",
				UID:  "12345",
			},
			FullName: "John Doe",
			Groups:   []string{"users"},
		},
	})
	defer s.Close()

	h, err := httpclient.NewHTTPClient(nil, true)
	expectNil(t, err)

	oc := openshiftConnector{apiURL: s.URL, httpClient: h, oauth2Config: &oauth2.Config{
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/oauth/authorize", s.URL),
			TokenURL: fmt.Sprintf("%s/oauth/token", s.URL),
		},
	}}

	data, err := json.Marshal(oauth2.Token{AccessToken: "fFAGRNJru1FTz70BzhT3Zg"})
	expectNil(t, err)

	oldID := connector.Identity{ConnectorData: data}

	identity, err := oc.Refresh(context.Background(), connector.Scopes{Groups: true}, oldID)

	expectNil(t, err)
	expectEquals(t, identity.UserID, "12345")
	expectEquals(t, identity.Username, "jdoe")
	expectEquals(t, identity.PreferredUsername, "jdoe")
	expectEquals(t, identity.Email, "jdoe")
	expectEquals(t, len(identity.Groups), 1)
	expectEquals(t, identity.Groups[0], "users")
}

func TestRefreshIdentityFailure(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		usersURLPath: user{
			ObjectMeta: k8sapi.ObjectMeta{
				Name: "jdoe",
				UID:  "12345",
			},
			FullName: "John Doe",
			Groups:   []string{"users"},
		},
	})
	defer s.Close()

	h, err := httpclient.NewHTTPClient(nil, true)
	expectNil(t, err)

	oc := openshiftConnector{apiURL: s.URL, httpClient: h, oauth2Config: &oauth2.Config{
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/oauth/authorize", s.URL),
			TokenURL: fmt.Sprintf("%s/oauth/token", s.URL),
		},
	}}

	data, err := json.Marshal(oauth2.Token{AccessToken: "oRzxVjCnohYRHEYEhZshkmakKmoyVoTjfUGC", Expiry: time.Now().Add(-time.Hour)})
	expectNil(t, err)

	oldID := connector.Identity{ConnectorData: data}

	identity, err := oc.Refresh(context.Background(), connector.Scopes{Groups: true}, oldID)
	expectNotNil(t, err)
	expectEquals(t, connector.Identity{}, identity)
}

func TestOpenWithClientSecretFile(t *testing.T) {
	s := newTestServer(map[string]interface{}{})
	defer s.Close()

	clientSecretFile := filepath.Join(t.TempDir(), "client-secret")
	err := os.WriteFile(clientSecretFile, []byte("client-secret-from-file\n"), 0o600)
	expectNil(t, err)

	c := Config{
		Issuer:           s.URL,
		ClientID:         "testClientId",
		ClientSecretFile: clientSecretFile,
		RedirectURI:      "https://localhost/callback",
		InsecureCA:       true,
	}

	logger := slog.New(slog.DiscardHandler)
	oconfig, err := c.Open("id", logger)

	expectNil(t, err)
	oc, ok := oconfig.(*openshiftConnector)
	expectEquals(t, ok, true)
	expectEquals(t, oc.clientSecret, "")
	expectEquals(t, oc.clientSecretFile, clientSecretFile)
	expectEquals(t, oc.oauth2Config.ClientSecret, "client-secret-from-file")
}

func TestOpenFailsBothClientSecretAndClientSecretFile(t *testing.T) {
	s := newTestServer(map[string]interface{}{})
	defer s.Close()

	clientSecretFile := filepath.Join(t.TempDir(), "client-secret")
	err := os.WriteFile(clientSecretFile, []byte("client-secret"), 0o600)
	expectNil(t, err)

	c := Config{
		Issuer:           s.URL,
		ClientID:         "testClientId",
		ClientSecret:     "testClientSecret",
		ClientSecretFile: clientSecretFile,
		RedirectURI:      "https://localhost/callback",
		InsecureCA:       true,
	}

	logger := slog.New(slog.DiscardHandler)
	_, err = c.Open("id", logger)
	expectNotNil(t, err)
}

func TestOpenFailsNeitherClientSecretNorClientSecretFile(t *testing.T) {
	s := newTestServer(map[string]interface{}{})
	defer s.Close()

	c := Config{
		Issuer:      s.URL,
		ClientID:    "testClientId",
		RedirectURI: "https://localhost/callback",
		InsecureCA:  true,
	}

	logger := slog.New(slog.DiscardHandler)
	_, err := c.Open("id", logger)
	expectNotNil(t, err)
}

func TestClientSecretFileRotation(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		usersURLPath: user{
			ObjectMeta: k8sapi.ObjectMeta{
				Name: "jdoe",
				UID:  "12345",
			},
			FullName: "John Doe",
			Groups:   []string{"users"},
		},
	})
	defer s.Close()

	clientSecretFile := filepath.Join(t.TempDir(), "client-secret")
	err := os.WriteFile(clientSecretFile, []byte("initial-client-secret"), 0o600)
	expectNil(t, err)

	h, err := httpclient.NewHTTPClient(nil, true)
	expectNil(t, err)

	oc := openshiftConnector{
		apiURL:           s.URL,
		httpClient:       h,
		clientSecretFile: clientSecretFile,
		oauth2Config: &oauth2.Config{
			ClientID:     "testClientId",
			ClientSecret: "initial-client-secret",
			Endpoint: oauth2.Endpoint{
				AuthURL:  fmt.Sprintf("%s/oauth/authorize", s.URL),
				TokenURL: fmt.Sprintf("%s/oauth/token", s.URL),
			},
		},
	}

	cfg1, err := oc.currentOAuth2Config()
	expectNil(t, err)
	expectEquals(t, cfg1.ClientSecret, "initial-client-secret")

	err = os.WriteFile(clientSecretFile, []byte("rotated-client-secret"), 0o600)
	expectNil(t, err)

	cfg2, err := oc.currentOAuth2Config()
	expectNil(t, err)
	expectEquals(t, cfg2.ClientSecret, "rotated-client-secret")

	// Original config should not be mutated
	expectEquals(t, oc.oauth2Config.ClientSecret, "initial-client-secret")
}

func TestCurrentOAuth2ConfigWithoutClientSecretFile(t *testing.T) {
	oauth2Cfg := &oauth2.Config{
		ClientID:     "testClientId",
		ClientSecret: "static-secret",
	}
	oc := openshiftConnector{
		oauth2Config: oauth2Cfg,
	}

	cfg, err := oc.currentOAuth2Config()
	expectNil(t, err)
	// Should return the same pointer when no client secret file is configured
	if cfg != oauth2Cfg {
		t.Error("expected same oauth2Config pointer when no client secret file is configured")
	}
}

func TestOpenFailsWithEmptyClientSecretFile(t *testing.T) {
	s := newTestServer(map[string]interface{}{})
	defer s.Close()

	clientSecretFile := filepath.Join(t.TempDir(), "client-secret")
	err := os.WriteFile(clientSecretFile, []byte(""), 0o600)
	expectNil(t, err)

	c := Config{
		Issuer:           s.URL,
		ClientID:         "testClientId",
		ClientSecretFile: clientSecretFile,
		RedirectURI:      "https://localhost/callback",
		InsecureCA:       true,
	}

	// secret file is empty
	logger := slog.New(slog.DiscardHandler)
	connConfig, err := c.Open("id", logger)
	expectNil(t, connConfig)
	expectEquals(t, err.Error(), fmt.Sprintf("client secret file %q contains no valid secret", clientSecretFile))

	// file contains only whitespace and new line char
	err = os.WriteFile(clientSecretFile, []byte("\t\n"), 0o600)
	expectNil(t, err)
	connConfig, err = c.Open("id", logger)
	expectNil(t, connConfig)
	expectEquals(t, err.Error(), fmt.Sprintf("client secret file %q contains no valid secret", clientSecretFile))
}

func TestOpenFailsWithNonExistentClientSecretFile(t *testing.T) {
	s := newTestServer(map[string]interface{}{})
	defer s.Close()

	clientSecretFile := filepath.Join(t.TempDir(), "client-secret")

	c := Config{
		Issuer:           s.URL,
		ClientID:         "testClientId",
		ClientSecretFile: clientSecretFile,
		RedirectURI:      "https://localhost/callback",
		InsecureCA:       true,
	}

	// file does not exist
	logger := slog.New(slog.DiscardHandler)
	connConfig, err := c.Open("id", logger)
	expectNil(t, connConfig)
	expectEquals(t, strings.HasPrefix(err.Error(), fmt.Sprintf("failed to read client secret file %q:", clientSecretFile)), true)
}

func TestCurrentOAuth2ConfigFailsWithEmptyClientSecretFile(t *testing.T) {
	s := newTestServer(map[string]interface{}{})
	defer s.Close()

	clientSecretFile := filepath.Join(t.TempDir(), "client-secret")
	err := os.WriteFile(clientSecretFile, []byte("client-secret-from-file"), 0o600)
	expectNil(t, err)

	c := Config{
		Issuer:           s.URL,
		ClientID:         "testClientId",
		ClientSecretFile: clientSecretFile,
		RedirectURI:      "https://localhost/callback",
		InsecureCA:       true,
	}

	logger := slog.New(slog.DiscardHandler)
	connConfig, err := c.Open("id", logger)
	expectNil(t, err)
	expectNotNil(t, connConfig)
	ocConnConfig := connConfig.(*openshiftConnector)
	oAuthConfig, err := ocConnConfig.currentOAuth2Config()
	expectNil(t, err)
	expectNotNil(t, oAuthConfig)
	expectEquals(t, "client-secret-from-file", oAuthConfig.ClientSecret)

	// secret file is empty
	err = os.WriteFile(clientSecretFile, []byte(""), 0o600)
	expectNil(t, err)

	oAuthConfig, err = ocConnConfig.currentOAuth2Config()
	expectNotNil(t, err)
	expectEquals(t, fmt.Sprintf("client secret file %q contains no valid secret", clientSecretFile), err.Error())

}

func TestCurrentOAuth2ConfigFailsWithNonExistentClientSecretFile(t *testing.T) {
	s := newTestServer(map[string]interface{}{})
	defer s.Close()

	clientSecretFile := filepath.Join(t.TempDir(), "client-secret")
	err := os.WriteFile(clientSecretFile, []byte("client-secret-from-file"), 0o600)
	expectNil(t, err)

	c := Config{
		Issuer:           s.URL,
		ClientID:         "testClientId",
		ClientSecretFile: clientSecretFile,
		RedirectURI:      "https://localhost/callback",
		InsecureCA:       true,
	}

	logger := slog.New(slog.DiscardHandler)
	connConfig, err := c.Open("id", logger)
	expectNil(t, err)
	expectNotNil(t, connConfig)
	ocConnConfig := connConfig.(*openshiftConnector)
	oAuthConfig, err := ocConnConfig.currentOAuth2Config()
	expectNil(t, err)
	expectNotNil(t, oAuthConfig)
	expectEquals(t, oAuthConfig.ClientSecret, "client-secret-from-file")

	// delete the client secret file
	err = os.Remove(clientSecretFile)
	expectNil(t, err)

	oAuthConfig, err = ocConnConfig.currentOAuth2Config()
	expectNotNil(t, err)
	expectEquals(t, strings.HasPrefix(err.Error(), fmt.Sprintf("failed to read client secret file %q:", clientSecretFile)), true)
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
		_ = json.NewEncoder(w).Encode(response)
	}))

	return s
}

func expectNil(t *testing.T, a interface{}) {
	t.Helper()
	if a != nil {
		t.Errorf("Expected %+v to equal nil", a)
	}
}

func expectEquals(t *testing.T, a interface{}, b interface{}) {
	t.Helper()
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Expected %+v to equal %+v", a, b)
	}
}

func expectNotNil(t *testing.T, a interface{}) {
	t.Helper()
	if a == nil {
		t.Errorf("Expected %+v to not equal nil", a)
	}
}

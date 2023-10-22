// Package atlassiancrowd provides authentication strategies using Atlassian Crowd.
package atlassiancrowd

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestUserGroups(t *testing.T) {
	s := newTestServer(map[string]TestServerResponse{
		"/rest/usermanagement/1/user/group/nested?username=testuser": {
			Body: crowdGroups{Groups: []struct{ Name string }{{Name: "group1"}, {Name: "group2"}}},
			Code: 200,
		},
	})
	defer s.Close()

	c := newTestCrowdConnector(s.URL)
	groups, err := c.getGroups(context.Background(), newClient(), true, "testuser")

	expectNil(t, err)
	expectEquals(t, groups, []string{"group1", "group2"})
}

func TestUserGroupsWithFiltering(t *testing.T) {
	s := newTestServer(map[string]TestServerResponse{
		"/rest/usermanagement/1/user/group/nested?username=testuser": {
			Body: crowdGroups{Groups: []struct{ Name string }{{Name: "group1"}, {Name: "group2"}}},
			Code: 200,
		},
	})
	defer s.Close()

	c := newTestCrowdConnector(s.URL)
	c.Groups = []string{"group1"}
	groups, err := c.getGroups(context.Background(), newClient(), true, "testuser")

	expectNil(t, err)
	expectEquals(t, groups, []string{"group1"})
}

func TestUserLoginFlow(t *testing.T) {
	s := newTestServer(map[string]TestServerResponse{
		"/rest/usermanagement/1/session?validate-password=false": {
			Body: crowdAuthentication{},
			Code: 201,
		},
		"/rest/usermanagement/1/user?username=testuser": {
			Body: crowdUser{Active: true, Name: "testuser", Email: "testuser@example.com"},
			Code: 200,
		},
		"/rest/usermanagement/1/user?username=testuser2": {
			Body: `<html>The server understood the request but refuses to authorize it.</html>`,
			Code: 403,
		},
	})
	defer s.Close()

	c := newTestCrowdConnector(s.URL)
	user, err := c.user(context.Background(), newClient(), "testuser")
	expectNil(t, err)
	expectEquals(t, user.Name, "testuser")
	expectEquals(t, user.Email, "testuser@example.com")

	err = c.authenticateUser(context.Background(), newClient(), "testuser")
	expectNil(t, err)

	_, err = c.user(context.Background(), newClient(), "testuser2")
	expectEquals(t, err, fmt.Errorf("dex is forbidden from making requests to the Atlassian Crowd application by URL %q", s.URL))
}

func TestUserPassword(t *testing.T) {
	s := newTestServer(map[string]TestServerResponse{
		"/rest/usermanagement/1/session": {
			Body: crowdAuthenticationError{Reason: "INVALID_USER_AUTHENTICATION", Message: "test"},
			Code: 401,
		},
		"/rest/usermanagement/1/session?validate-password=false": {
			Body: crowdAuthentication{},
			Code: 201,
		},
	})
	defer s.Close()

	c := newTestCrowdConnector(s.URL)
	invalidPassword, err := c.authenticateWithPassword(context.Background(), newClient(), "testuser", "testpassword")

	expectNil(t, err)
	expectEquals(t, invalidPassword, true)

	err = c.authenticateUser(context.Background(), newClient(), "testuser")
	expectNil(t, err)
}

func TestIdentityFromCrowdUser(t *testing.T) {
	user := crowdUser{
		Key:    "12345",
		Name:   "testuser",
		Active: true,
		Email:  "testuser@example.com",
	}

	c := newTestCrowdConnector("/")

	// Sanity checks
	expectEquals(t, user.Name, "testuser")
	expectEquals(t, user.Email, "testuser@example.com")

	// Test unconfigured behaviour
	i := c.identityFromCrowdUser(user)
	expectEquals(t, i.UserID, "12345")
	expectEquals(t, i.Username, "testuser")
	expectEquals(t, i.Email, "testuser@example.com")
	expectEquals(t, i.EmailVerified, true)

	// Test for various PreferredUsernameField settings
	// unset
	expectEquals(t, i.PreferredUsername, "")

	c.Config.PreferredUsernameField = "key"
	i = c.identityFromCrowdUser(user)
	expectEquals(t, i.PreferredUsername, "12345")

	c.Config.PreferredUsernameField = "name"
	i = c.identityFromCrowdUser(user)
	expectEquals(t, i.PreferredUsername, "testuser")

	c.Config.PreferredUsernameField = "email"
	i = c.identityFromCrowdUser(user)
	expectEquals(t, i.PreferredUsername, "testuser@example.com")

	c.Config.PreferredUsernameField = "invalidstring"
	i = c.identityFromCrowdUser(user)
	expectEquals(t, i.PreferredUsername, "")
}

type TestServerResponse struct {
	Body interface{}
	Code int
}

func newTestCrowdConnector(baseURL string) crowdConnector {
	connector := crowdConnector{}
	connector.BaseURL = baseURL
	connector.logger = &logrus.Logger{
		Out:       io.Discard,
		Level:     logrus.DebugLevel,
		Formatter: &logrus.TextFormatter{DisableColors: true},
	}
	return connector
}

func newTestServer(responses map[string]TestServerResponse) *httptest.Server {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := responses[r.RequestURI]
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(response.Code)
		json.NewEncoder(w).Encode(response.Body)
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

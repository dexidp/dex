package bitbucketcloud

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/dexidp/dex/connector"
)

func TestUserGroups(t *testing.T) {
	workspacesResponse := workspacesResponse{
		pagedResponse: pagedResponse{
			Size:    3,
			Page:    1,
			PageLen: 10,
		},
		Values: []workspaceAccess{
			{Workspace: workspaceRef{Slug: "team-1"}},
			{Workspace: workspaceRef{Slug: "team-2"}},
			{Workspace: workspaceRef{Slug: "team-3"}},
		},
	}

	s := newTestServer(map[string]interface{}{
		"/user/workspaces": workspacesResponse,
	})

	connector := bitbucketConnector{apiURL: s.URL}
	groups, err := connector.userWorkspaces(context.Background(), newClient())

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
		"/user/workspaces": workspacesResponse{},
	})

	connector := bitbucketConnector{apiURL: s.URL}
	groups, err := connector.userWorkspaces(context.Background(), newClient())

	expectNil(t, err)
	expectEquals(t, len(groups), 0)

	s.Close()
}

func TestUserGroupsWithPermissions(t *testing.T) {
	workspacesResp := workspacesResponse{
		pagedResponse: pagedResponse{
			Size:    2,
			Page:    1,
			PageLen: 10,
		},
		Values: []workspaceAccess{
			{Workspace: workspaceRef{Slug: "team-1"}},
			{Workspace: workspaceRef{Slug: "team-2"}},
		},
	}

	s := newTestServer(map[string]interface{}{
		"/user/workspaces":                   workspacesResp,
		"/user/workspaces/team-1/permission": workspacePermission{Permission: "owner"},
		"/user/workspaces/team-2/permission": workspacePermission{Permission: "member"},
	})
	defer s.Close()

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	c := bitbucketConnector{apiURL: s.URL, getWorkspacePermissions: true, logger: logger}
	groups, err := c.userWorkspaces(context.Background(), newClient())

	expectNil(t, err)
	expectEquals(t, groups, []string{
		"team-1",
		"team-2",
		"team-1:owner",
		"team-2:member",
	})
}

func TestDeprecatedIncludeTeamGroups(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	cfg := Config{
		ClientID:          "id",
		ClientSecret:      "secret",
		RedirectURI:       "http://localhost",
		IncludeTeamGroups: true,
	}

	_, err := cfg.Open("test", logger)
	expectNil(t, err)

	if !strings.Contains(buf.String(), "includeTeamGroups is deprecated") {
		t.Fatal("expected deprecation warning for includeTeamGroups")
	}
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
	identity, err := bitbucketConnector.HandleCallback(connector.Scopes{}, nil, req)

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

// TestRefreshUpstreamRejection verifies that a refresh Bitbucket refuses keeps
// the upstream's OAuth2 error, so the server can tell a rejected refresh
// token apart from other failures.
func TestRefreshUpstreamRejection(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
	}))
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	c := bitbucketConnector{apiURL: s.URL, hostName: hostURL.Host}
	connData, err := json.Marshal(connectorData{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		Expiry:       time.Now().Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.WithValue(t.Context(), oauth2.HTTPClient, newClient())
	_, err = c.Refresh(ctx, connector.Scopes{OfflineAccess: true}, connector.Identity{ConnectorData: connData})

	var retrieveErr *oauth2.RetrieveError
	if !errors.As(err, &retrieveErr) || retrieveErr.ErrorCode != "invalid_grant" {
		t.Fatalf("expected the upstream invalid_grant in the error chain, got %v", err)
	}
}

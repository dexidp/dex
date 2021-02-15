package bitbucketcloud

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/connector"
)

func TestUserGroups(t *testing.T) {
	workspaceResponse := userWorkspacesResponse{
		pagedResponse: pagedResponse{
			Size:    3,
			Page:    1,
			PageLen: 10,
		},
		Values: []workspace{
			{Name: "workspace-1"},
			{Name: "workspace-2"},
			{Name: "workspace-3"},
		},
	}

	s := newTestServer(map[string]interface{}{
		"/workspaces?role=member": workspaceResponse,
		"/groups/workspace-1":     []group{{Slug: "administrators"}, {Slug: "members"}},
		"/groups/workspace-2":     []group{{Slug: "everyone"}},
		"/groups/workspace-3":     []group{},
	})
	defer s.Close()

	c := bitbucketConnector{apiURL: s.URL, legacyAPIURL: s.URL}
	groups, err := c.userWorkspaces(context.Background(), newClient())

	require.NoError(t, err)
	require.Equal(t, []string{
		"workspace-1",
		"workspace-2",
		"workspace-3",
	}, groups)

	c.includeTeamGroups = true
	groups, err = c.userWorkspaces(context.Background(), newClient())

	require.NoError(t, err)
	require.Equal(t, []string{
		"workspace-1",
		"workspace-2",
		"workspace-3",
		"workspace-1/administrators",
		"workspace-1/members",
		"workspace-2/everyone",
	}, groups)
}

func TestUserWithoutWorkspaces(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/workspaces?role=member": userWorkspacesResponse{},
	})
	defer s.Close()

	c := bitbucketConnector{apiURL: s.URL}
	groups, err := c.userWorkspaces(context.Background(), newClient())

	require.NoError(t, err)
	require.Len(t, groups, 0)
}

func TestUserWithWorkspaces(t *testing.T) {
	s := newTestServer(map[string]interface{}{
		"/workspaces?role=member": userWorkspacesResponse{
			pagedResponse: pagedResponse{
				Size:    3,
				Page:    1,
				PageLen: 10,
			},
			Values: []workspace{
				{Name: "workspace-1"},
				{Name: "workspace-2"},
				{Name: "workspace-3"},
			},
		},
	})
	defer s.Close()

	c := bitbucketConnector{apiURL: s.URL, workspaces: []string{"workspace-1"}}
	groups, err := c.getGroups(context.Background(), newClient(), true, "test")

	require.NoError(t, err)
	require.Equal(t, []string{
		"workspace-1",
	}, groups)
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
	defer s.Close()

	hostURL, err := url.Parse(s.URL)
	require.NoError(t, err)

	req, err := http.NewRequest("GET", hostURL.String(), nil)
	require.NoError(t, err)

	c := bitbucketConnector{apiURL: s.URL, hostName: hostURL.Host, httpClient: newClient()}
	identity, err := c.HandleCallback(connector.Scopes{}, req)

	require.NoError(t, err)
	require.Equal(t, "some-login", identity.Username)
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

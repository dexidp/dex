package connector

import (
	"net/http"
	"testing"

	"github.com/coreos/go-oidc/oidc"
)

var (
	githubExampleUser  = `{"login":"octocat","id":1,"name": "monalisa octocat","email": "octocat@github.com"}`
	githubExampleError = `{"message":"Bad credentials","documentation_url":"https://developer.github.com/v3"}`
)

func TestGitHubIdentity(t *testing.T) {
	tests := []oauth2IdentityTest{
		{
			urlResps: map[string]response{
				githubAPIUserURL: {http.StatusOK, githubExampleUser},
			},
			want: oidc.Identity{
				Name:  "monalisa octocat",
				ID:    "1",
				Email: "octocat@github.com",
			},
		},
		{
			urlResps: map[string]response{
				githubAPIUserURL: {http.StatusUnauthorized, githubExampleError},
			},
			wantErr: githubError{
				Message: "Bad credentials",
			},
		},
	}
	conn, err := newGitHubConnector("fakeclientid", "fakeclientsecret", "http://examle.com/auth/github/callback")
	if err != nil {
		t.Fatal(err)
	}
	runOAuth2IdentityTests(t, conn, tests)
}

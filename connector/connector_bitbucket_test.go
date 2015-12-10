package connector

import (
	"net/http"
	"testing"

	"github.com/coreos/go-oidc/oidc"
)

var bitbucketExampleUser1 = `{
    "display_name": "tutorials account",
    "username": "tutorials",
    "uuid": "{c788b2da-b7a2-404c-9e26-d3f077557007}"
}`

var bitbucketExampleUser2 = `{
    "username": "tutorials",
    "uuid": "{c788b2da-b7a2-404c-9e26-d3f077557007}"
}`

var bitbucketExampleEmail = `{
  "values": [
    {"email": "tutorials1@bitbucket.org","is_confirmed": false,"is_primary": false},
    {"email": "tutorials2@bitbucket.org","is_confirmed": true,"is_primary": false},
    {"email": "tutorials3@bitbucket.org","is_confirmed": true,"is_primary": true}
  ]
}`

func TestBitBucketIdentity(t *testing.T) {
	tests := []oauth2IdentityTest{
		{
			urlResps: map[string]response{
				bitbucketAPIUserURL:  {http.StatusOK, bitbucketExampleUser1},
				bitbucketAPIEmailURL: {http.StatusOK, bitbucketExampleEmail},
			},
			want: oidc.Identity{
				Name:  "tutorials account",
				ID:    "{c788b2da-b7a2-404c-9e26-d3f077557007}",
				Email: "tutorials3@bitbucket.org",
			},
		},
		{
			urlResps: map[string]response{
				bitbucketAPIUserURL:  {http.StatusOK, bitbucketExampleUser2},
				bitbucketAPIEmailURL: {http.StatusOK, bitbucketExampleEmail},
			},
			want: oidc.Identity{
				Name:  "tutorials",
				ID:    "{c788b2da-b7a2-404c-9e26-d3f077557007}",
				Email: "tutorials3@bitbucket.org",
			},
		},
	}
	conn, err := newBitbucketConnector("fakeclientid", "fakeclientsecret", "http://example.com/auth/bitbucket/callback")
	if err != nil {
		t.Fatal(err)
	}
	runOAuth2IdentityTests(t, conn, tests)
}

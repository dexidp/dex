package connector

import (
	"github.com/coreos/go-oidc/oidc"
	"net/http"
	"testing"
)

var facebookUser1 = `{
	"id":"testUser1",
	"name":"testUser1Fname testUser1Lname",
	"email":  "testUser1@facebook.com"
	}`

var facebookUser2 = `{
	"id":"testUser2",
	"name":"testUser2Fname testUser2Lname",
	"email":  "testUser2@facebook.com"
	}`

var facebookExampleError = `{
   "error": {
      "message": "Invalid OAuth access token signature.",
      "type": "OAuthException",
      "code": 190,
      "fbtrace_id": "Ee/6W0EfrWP"
   }
}`

func TestFacebookIdentity(t *testing.T) {
	tests := []oauth2IdentityTest{
		{
			urlResps: map[string]response{
				facebookGraphAPIURL: {http.StatusOK, facebookUser1},
			},
			want: oidc.Identity{
				Name:  "testUser1Fname testUser1Lname",
				ID:    "testUser1",
				Email: "testUser1@facebook.com",
			},
		},
		{
			urlResps: map[string]response{
				facebookGraphAPIURL: {http.StatusOK, facebookUser2},
			},
			want: oidc.Identity{
				Name:  "testUser2Fname testUser2Lname",
				ID:    "testUser2",
				Email: "testUser2@facebook.com",
			},
		},
		{
			urlResps: map[string]response{
				facebookGraphAPIURL: {http.StatusUnauthorized, facebookExampleError},
			},
			wantErr: facebookErr{
				ErrorMessage: ErrorMessage{
					Code:      190,
					Type:      "OAuthException",
					Message:   "Invalid OAuth access token signature.",
					FbTraceId: "Ee/6W0EfrWP",
				},
			},
		},
	}
	conn, err := newFacebookConnector("fakeFacebookAppID", "fakeFacebookAppSecret", "http://example.com/auth/facebook/callback")
	if err != nil {
		t.Fatal(err)
	}
	runOAuth2IdentityTests(t, conn, tests)
}

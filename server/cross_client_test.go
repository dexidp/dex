package server

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/coreos/go-oidc/oidc"
	"github.com/kylelemons/godebug/pretty"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/scope"
)

func makeCrossClientTestFixtures() (*testFixtures, error) {
	xClients := []client.LoadableClient{}
	for _, cliData := range []struct {
		id           string
		trustedPeers []string
	}{
		{
			id: "client_a",
		}, {
			id:           "client_b",
			trustedPeers: []string{"client_a"},
		}, {
			id:           "client_c",
			trustedPeers: []string{"client_a", "client_b"},
		},
	} {
		u := url.URL{
			Scheme: "https://",
			Path:   cliData.id,
			Host:   cliData.id,
		}
		xClients = append(xClients, client.LoadableClient{
			Client: client.Client{
				Credentials: oidc.ClientCredentials{
					ID: cliData.id,
					Secret: base64.URLEncoding.EncodeToString(
						[]byte(cliData.id + "_secret")),
				},
				Metadata: oidc.ClientMetadata{
					RedirectURIs: []url.URL{u},
				},
			},
			TrustedPeers: cliData.trustedPeers,
		})
	}

	xClients = append(xClients, testClients...)
	f, err := makeTestFixturesWithOptions(testFixtureOptions{
		clients: xClients,
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't make test fixtures: %v", err)
	}
	return f, nil
}

func TestServerCrossClientAuthAllowed(t *testing.T) {
	f, err := makeCrossClientTestFixtures()
	if err != nil {
		t.Fatalf("couldn't make test fixtures: %v", err)
	}

	tests := []struct {
		reqClient      string
		authClient     string
		wantAuthorized bool
		wantErr        bool
	}{
		{
			reqClient:      "client_b",
			authClient:     "client_a",
			wantAuthorized: false,
			wantErr:        false,
		},
		{
			reqClient:      "client_a",
			authClient:     "client_b",
			wantAuthorized: true,
			wantErr:        false,
		},
		{
			reqClient:      "client_a",
			authClient:     "client_c",
			wantAuthorized: true,
			wantErr:        false,
		},
		{
			reqClient:      "client_c",
			authClient:     "client_b",
			wantAuthorized: false,
			wantErr:        false,
		},
		{
			reqClient:  "client_c",
			authClient: "nope",
			wantErr:    false,
		},
	}
	for i, tt := range tests {
		got, err := f.srv.CrossClientAuthAllowed(tt.reqClient, tt.authClient)
		if tt.wantErr {
			if err == nil {
				t.Errorf("case %d: want non-nil err", i)
			}
			continue
		}
		if err != nil {
			t.Errorf("case %d: unexpected err %v: ", i, err)
		}

		if got != tt.wantAuthorized {
			t.Errorf("case %d: want=%v, got=%v", i, tt.wantAuthorized, got)
		}
	}
}

func TestHandleAuthCrossClient(t *testing.T) {
	f, err := makeCrossClientTestFixtures()
	if err != nil {
		t.Fatalf("couldn't make test fixtures: %v", err)
	}

	tests := []struct {
		scopes   []string
		clientID string
		wantCode int
	}{
		{
			scopes:   []string{scope.ScopeGoogleCrossClient + "client_a"},
			clientID: "client_b",
			wantCode: http.StatusBadRequest,
		},
		{
			scopes:   []string{scope.ScopeGoogleCrossClient + "client_b"},
			clientID: "client_a",
			wantCode: http.StatusFound,
		},
		{
			scopes:   []string{scope.ScopeGoogleCrossClient + "client_b"},
			clientID: "client_a",
			wantCode: http.StatusFound,
		},
		{
			scopes:   []string{scope.ScopeGoogleCrossClient + "client_c"},
			clientID: "client_a",
			wantCode: http.StatusFound,
		},
		{
			// Two clients that client_a is authorized to mint tokens for.
			scopes: []string{
				scope.ScopeGoogleCrossClient + "client_c",
				scope.ScopeGoogleCrossClient + "client_b",
			},
			clientID: "client_a",
			wantCode: http.StatusFound,
		},
		{
			// Two clients that client_a is authorized to mint tokens for.
			scopes: []string{
				scope.ScopeGoogleCrossClient + "client_c",
				scope.ScopeGoogleCrossClient + "client_a",
			},
			clientID: "client_b",
			wantCode: http.StatusBadRequest,
		},
	}

	idpcs := []connector.Connector{
		&fakeConnector{loginURL: "http://fake.example.com"},
	}

	for i, tt := range tests {
		hdlr := handleAuthFunc(f.srv, url.URL{}, idpcs, nil, true)
		w := httptest.NewRecorder()

		query := url.Values{
			"response_type": []string{"code"},
			"client_id":     []string{tt.clientID},
			"connector_id":  []string{"fake"},
			"scope":         []string{strings.Join(append([]string{"openid"}, tt.scopes...), " ")},
		}
		u := fmt.Sprintf("http://server.example.com?%s", query.Encode())
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			t.Errorf("case %d: unable to form HTTP request: %v", i, err)
			continue
		}

		hdlr.ServeHTTP(w, req)
		if tt.wantCode != w.Code {
			t.Errorf("case %d: HTTP code mismatch: want=%d got=%d", i, tt.wantCode, w.Code)
			continue
		}
	}

}

func TestServerCodeTokenCrossClient(t *testing.T) {
	f, err := makeCrossClientTestFixtures()
	if err != nil {
		t.Fatalf("Error creating test fixtures: %v", err)
	}
	sm := f.sessionManager

	tests := []struct {
		clientID     string
		offline      bool
		refreshToken string
		crossClients []string

		wantErr bool
		wantAUD []string
		wantAZP string
	}{
		// First test the non-cross-client cases, make sure they're undisturbed:
		{
			// No 'offline_access' in scope, should get empty refresh token.
			clientID:     testClientID,
			refreshToken: "",

			wantAUD: []string{testClientID},
		},
		{
			// Have 'offline_access' in scope, should get non-empty refresh token.
			clientID:     testClientID,
			offline:      true,
			refreshToken: fmt.Sprintf("1/%s", base64.URLEncoding.EncodeToString([]byte("refresh-1"))),

			wantAUD: []string{testClientID},
		},
		// Now test cross-client cases:
		{
			clientID:     "client_a",
			crossClients: []string{"client_b"},

			wantAUD: []string{"client_b"},
			wantAZP: "client_a",
		},
		{
			clientID:     "client_a",
			crossClients: []string{"client_b", "client_a"},

			wantAUD: []string{"client_a", "client_b"},
			wantAZP: "client_a",
		},
	}

	for i, tt := range tests {
		scopes := []string{"openid"}
		if tt.offline {
			scopes = append(scopes, "offline_access")
		}
		for _, client := range tt.crossClients {
			scopes = append(scopes, scope.ScopeGoogleCrossClient+client)
		}

		sessionID, err := sm.NewSession("bogus_idpc", tt.clientID, "bogus", url.URL{}, "", false, scopes)
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}
		_, err = sm.AttachRemoteIdentity(sessionID, oidc.Identity{})
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}

		_, err = sm.AttachUser(sessionID, "ID-1")
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}

		key, err := sm.NewSessionKey(sessionID)
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}

		jwt, token, expiresAt, err := f.srv.CodeToken(f.clientCreds[tt.clientID], key)
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}
		if jwt == nil {
			t.Fatalf("case %d: expect non-nil jwt", i)
		}
		if token != tt.refreshToken {
			t.Errorf("case %d: expect refresh token %q, got %q", i, tt.refreshToken, token)
		}
		if expiresAt.IsZero() {
			t.Errorf("case %d: expect non-zero expiration time", i)
		}

		claims, err := jwt.Claims()
		if err != nil {
			t.Fatalf("case %d: unexpected error getting claims: %v", i, err)
		}

		var gotAUD []string
		if len(tt.wantAUD) < 2 {
			aud, _, err := claims.StringClaim("aud")
			if err != nil {
				t.Fatalf("case %d: unexpected error getting 'aud': %q: raw: %v", i, err, claims["aud"])
			}
			gotAUD = []string{aud}
		} else {
			gotAUD, _, err = claims.StringsClaim("aud")
			if err != nil {
				t.Fatalf("case %d: unexpected error getting 'aud': %v", i, err)
			}
		}

		sort.Strings(gotAUD)
		if diff := pretty.Compare(tt.wantAUD, gotAUD); diff != "" {
			t.Fatalf("case %d: pretty.Compare(tt.wantAUD, gotAUD): %v", i, diff)
		}

		gotAZP, _, err := claims.StringClaim("azp")
		if err != nil {
			if err != nil {
				t.Fatalf("case %d: unexpected error getting 'aud': %v", i, err)
			}
		}

		if gotAZP != tt.wantAZP {
			t.Errorf("case %d: wantAZP=%v, gotAZP=%v", i, tt.wantAZP, gotAZP)
		}
	}
}

package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/coreos/go-oidc/oidc"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/connector"
)

func makeCrossClientTestFixtures() (*testFixtures, error) {
	f, err := makeTestFixtures()
	if err != nil {
		return nil, fmt.Errorf("couldn't make test fixtures: %v", err)
	}

	creds := map[string]oidc.ClientCredentials{}
	for _, cliData := range []struct {
		id         string
		authorized []string
	}{
		{
			id: "client_a",
		}, {
			id:         "client_b",
			authorized: []string{"client_a"},
		}, {
			id:         "client_c",
			authorized: []string{"client_a", "client_b"},
		},
	} {
		u := url.URL{
			Scheme: "https://",
			Path:   cliData.id,
			Host:   "auth.example.com",
		}
		cliCreds, err := f.clientRepo.New(client.Client{
			Credentials: oidc.ClientCredentials{
				ID: cliData.id,
			},
			Metadata: oidc.ClientMetadata{
				RedirectURIs: []url.URL{u},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("Unexpected error creating clients: %v", err)
		}
		creds[cliData.id] = *cliCreds
		err = f.clientRepo.SetTrustedPeers(cliData.id, cliData.authorized)
		if err != nil {
			return nil, fmt.Errorf("Unexpected error setting cross-client authorizers: %v", err)
		}
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
			scopes:   []string{ScopeGoogleCrossClient + "client_a"},
			clientID: "client_b",
			wantCode: http.StatusBadRequest,
		},
		{
			scopes:   []string{ScopeGoogleCrossClient + "client_b"},
			clientID: "client_a",
			wantCode: http.StatusFound,
		},
		{
			scopes:   []string{ScopeGoogleCrossClient + "client_b"},
			clientID: "client_a",
			wantCode: http.StatusFound,
		},
		{
			scopes:   []string{ScopeGoogleCrossClient + "client_c"},
			clientID: "client_a",
			wantCode: http.StatusFound,
		},
		{
			// Two clients that client_a is authorized to mint tokens for.
			scopes: []string{
				ScopeGoogleCrossClient + "client_c",
				ScopeGoogleCrossClient + "client_b",
			},
			clientID: "client_a",
			wantCode: http.StatusFound,
		},
		{
			// Two clients that client_a is authorized to mint tokens for.
			scopes: []string{
				ScopeGoogleCrossClient + "client_c",
				ScopeGoogleCrossClient + "client_a",
			},
			clientID: "client_b",
			wantCode: http.StatusBadRequest,
		},
	}

	idpcs := []connector.Connector{
		&fakeConnector{loginURL: "http://fake.example.com"},
	}

	for i, tt := range tests {
		hdlr := handleAuthFunc(f.srv, idpcs, nil, true)
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

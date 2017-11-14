package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	jose "gopkg.in/square/go-jose.v2"

	"github.com/coreos/dex/storage"
)

func TestParseAuthorizationRequest(t *testing.T) {
	tests := []struct {
		name                   string
		clients                []storage.Client
		supportedResponseTypes []string

		usePOST bool

		queryParams map[string]string

		wantErr bool

		authReqCheck func(*testing.T, *storage.AuthRequest)
	}{
		{
			name: "normal request",
			clients: []storage.Client{
				{
					ID:           "foo",
					RedirectURIs: []string{"https://example.com/foo"},
				},
			},
			supportedResponseTypes: []string{"code"},
			queryParams: map[string]string{
				"client_id":     "foo",
				"redirect_uri":  "https://example.com/foo",
				"response_type": "code",
				"scope":         "openid email profile",
			},
		},
		{
			name: "request with valid id_token_hint",
			clients: []storage.Client{
				{
					ID:           "foo",
					RedirectURIs: []string{"https://example.com/foo"},
				},
			},
			supportedResponseTypes: []string{"code"},
			queryParams: map[string]string{
				// sub: "Cg0wLTM4NS0yODA4OS0wEgRtb2Nr" = {0-385-28089-0, "mock"}
				"id_token_hint": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJDZzB3TFRNNE5TMHlPREE0T1Mwd0VnUnRiMk5yIn0.M1mYqRIYeAaLeo3B7DWj_Nxm589tbworSGffCIgBz04",
				"client_id":     "foo",
				"redirect_uri":  "https://example.com/foo",
				"response_type": "code",
				"scope":         "openid email profile",
			},
			authReqCheck: func(t *testing.T, ar *storage.AuthRequest) {
				if ar.ConnectorID != "mock" {
					t.Errorf("expected connectorID \"mock\", got %v", ar.ConnectorID)
				}
			},
		},
		{
			name: "request with non-jwt id_token_hint",
			clients: []storage.Client{
				{
					ID:           "foo",
					RedirectURIs: []string{"https://example.com/foo"},
				},
			},
			supportedResponseTypes: []string{"code"},
			queryParams: map[string]string{
				"id_token_hint": "notevenajwt",
				"client_id":     "foo",
				"redirect_uri":  "https://example.com/foo",
				"response_type": "code",
				"scope":         "openid email profile",
			},
			wantErr: true,
		},
		{
			name: "request with invalid id_token_hint (bad sub)",
			clients: []storage.Client{
				{
					ID:           "foo",
					RedirectURIs: []string{"https://example.com/foo"},
				},
			},
			supportedResponseTypes: []string{"code"},
			queryParams: map[string]string{
				// sub: "ject"
				"id_token_hint": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJqZWN0In0.GwGd9UBSu2XrfeULp8u3KI0jZPt1ccUIGk1TaCCtLqE",
				"client_id":     "foo",
				"redirect_uri":  "https://example.com/foo",
				"response_type": "code",
				"scope":         "openid email profile",
			},
			wantErr: true,
		},
		{
			name: "POST request",
			clients: []storage.Client{
				{
					ID:           "foo",
					RedirectURIs: []string{"https://example.com/foo"},
				},
			},
			supportedResponseTypes: []string{"code"},
			queryParams: map[string]string{
				"client_id":     "foo",
				"redirect_uri":  "https://example.com/foo",
				"response_type": "code",
				"scope":         "openid email profile",
			},
			usePOST: true,
		},
		{
			name: "invalid client id",
			clients: []storage.Client{
				{
					ID:           "foo",
					RedirectURIs: []string{"https://example.com/foo"},
				},
			},
			supportedResponseTypes: []string{"code"},
			queryParams: map[string]string{
				"client_id":     "bar",
				"redirect_uri":  "https://example.com/foo",
				"response_type": "code",
				"scope":         "openid email profile",
			},
			wantErr: true,
		},
		{
			name: "invalid redirect uri",
			clients: []storage.Client{
				{
					ID:           "bar",
					RedirectURIs: []string{"https://example.com/bar"},
				},
			},
			supportedResponseTypes: []string{"code"},
			queryParams: map[string]string{
				"client_id":     "bar",
				"redirect_uri":  "https://example.com/foo",
				"response_type": "code",
				"scope":         "openid email profile",
			},
			wantErr: true,
		},
		{
			name: "implicit flow",
			clients: []storage.Client{
				{
					ID:           "bar",
					RedirectURIs: []string{"https://example.com/bar"},
				},
			},
			supportedResponseTypes: []string{"code", "id_token", "token"},
			queryParams: map[string]string{
				"client_id":     "bar",
				"redirect_uri":  "https://example.com/bar",
				"response_type": "code id_token",
				"scope":         "openid email profile",
			},
		},
		{
			name: "unsupported response type",
			clients: []storage.Client{
				{
					ID:           "bar",
					RedirectURIs: []string{"https://example.com/bar"},
				},
			},
			supportedResponseTypes: []string{"code"},
			queryParams: map[string]string{
				"client_id":     "bar",
				"redirect_uri":  "https://example.com/bar",
				"response_type": "code id_token",
				"scope":         "openid email profile",
			},
			wantErr: true,
		},
		{
			name: "only token response type",
			clients: []storage.Client{
				{
					ID:           "bar",
					RedirectURIs: []string{"https://example.com/bar"},
				},
			},
			supportedResponseTypes: []string{"code", "id_token", "token"},
			queryParams: map[string]string{
				"client_id":     "bar",
				"redirect_uri":  "https://example.com/bar",
				"response_type": "token",
				"scope":         "openid email profile",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			httpServer, server := newTestServer(ctx, t, func(c *Config) {
				c.SupportedResponseTypes = tc.supportedResponseTypes
				c.Storage = storage.WithStaticClients(c.Storage, tc.clients)
			})
			defer httpServer.Close()

			params := url.Values{}
			for k, v := range tc.queryParams {
				params.Set(k, v)
			}

			var req *http.Request
			if tc.usePOST {
				body := strings.NewReader(params.Encode())
				req = httptest.NewRequest("POST", httpServer.URL+"/auth", body)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else {
				req = httptest.NewRequest("GET", httpServer.URL+"/auth?"+params.Encode(), nil)
			}
			resp, err := server.parseAuthorizationRequest(req)
			if err != nil && !tc.wantErr {
				t.Fatal(err)
			}
			if err == nil && tc.wantErr {
				t.Error("expected error")
			}
			if tc.authReqCheck != nil {
				tc.authReqCheck(t, &resp)
			}
		})
	}
}

const (
	// at_hash value and access_token returned by Google.
	googleAccessTokenHash = "piwt8oCH-K2D9pXlaS1Y-w"
	googleAccessToken     = "ya29.CjHSA1l5WUn8xZ6HanHFzzdHdbXm-14rxnC7JHch9eFIsZkQEGoWzaYG4o7k5f6BnPLj"
	googleSigningAlg      = jose.RS256
)

func TestAccessTokenHash(t *testing.T) {
	atHash, err := accessTokenHash(googleSigningAlg, googleAccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if atHash != googleAccessTokenHash {
		t.Errorf("expected %q got %q", googleAccessTokenHash, atHash)
	}
}

func TestValidRedirectURI(t *testing.T) {
	tests := []struct {
		client      storage.Client
		redirectURI string
		wantValid   bool
	}{
		{
			client: storage.Client{
				RedirectURIs: []string{"http://foo.com/bar"},
			},
			redirectURI: "http://foo.com/bar",
			wantValid:   true,
		},
		{
			client: storage.Client{
				RedirectURIs: []string{"http://foo.com/bar"},
			},
			redirectURI: "http://foo.com/bar/baz",
		},
		{
			client: storage.Client{
				Public: true,
			},
			redirectURI: "urn:ietf:wg:oauth:2.0:oob",
			wantValid:   true,
		},
		{
			client: storage.Client{
				Public: true,
			},
			redirectURI: "http://localhost:8080/",
			wantValid:   true,
		},
		{
			client: storage.Client{
				Public: true,
			},
			redirectURI: "http://localhost:991/bar",
			wantValid:   true,
		},
		{
			client: storage.Client{
				Public: true,
			},
			redirectURI: "http://localhost",
			wantValid:   true,
		},
		{
			client: storage.Client{
				Public: true,
			},
			redirectURI: "http://localhost.localhost:8080/",
			wantValid:   false,
		},
	}
	for _, test := range tests {
		got := validateRedirectURI(test.client, test.redirectURI)
		if got != test.wantValid {
			t.Errorf("client=%#v, redirectURI=%q, wanted valid=%t, got=%t",
				test.client, test.redirectURI, test.wantValid, got)
		}
	}
}

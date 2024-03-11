package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-jose/go-jose/v4"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func TestParseAuthorizationRequest(t *testing.T) {
	tests := []struct {
		name                   string
		clients                []storage.Client
		supportedResponseTypes []string

		usePOST bool

		queryParams map[string]string

		expectedError error
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
			expectedError: &displayedAuthErr{Status: http.StatusNotFound},
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
			expectedError: &displayedAuthErr{Status: http.StatusBadRequest},
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
			expectedError: &redirectedAuthErr{Type: errUnsupportedResponseType},
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
			expectedError: &redirectedAuthErr{Type: errInvalidRequest},
		},
		{
			name: "choose connector_id",
			clients: []storage.Client{
				{
					ID:           "bar",
					RedirectURIs: []string{"https://example.com/bar"},
				},
			},
			supportedResponseTypes: []string{"code", "id_token", "token"},
			queryParams: map[string]string{
				"connector_id":  "mock",
				"client_id":     "bar",
				"redirect_uri":  "https://example.com/bar",
				"response_type": "code id_token",
				"scope":         "openid email profile",
			},
		},
		{
			name: "choose second connector_id",
			clients: []storage.Client{
				{
					ID:           "bar",
					RedirectURIs: []string{"https://example.com/bar"},
				},
			},
			supportedResponseTypes: []string{"code", "id_token", "token"},
			queryParams: map[string]string{
				"connector_id":  "mock2",
				"client_id":     "bar",
				"redirect_uri":  "https://example.com/bar",
				"response_type": "code id_token",
				"scope":         "openid email profile",
			},
		},
		{
			name: "choose invalid connector_id",
			clients: []storage.Client{
				{
					ID:           "bar",
					RedirectURIs: []string{"https://example.com/bar"},
				},
			},
			supportedResponseTypes: []string{"code", "id_token", "token"},
			queryParams: map[string]string{
				"connector_id":  "bogus",
				"client_id":     "bar",
				"redirect_uri":  "https://example.com/bar",
				"response_type": "code id_token",
				"scope":         "openid email profile",
			},
			expectedError: &redirectedAuthErr{Type: errInvalidRequest},
		},
		{
			name: "PKCE code_challenge_method plain",
			clients: []storage.Client{
				{
					ID:           "bar",
					RedirectURIs: []string{"https://example.com/bar"},
				},
			},
			supportedResponseTypes: []string{"code"},
			queryParams: map[string]string{
				"client_id":             "bar",
				"redirect_uri":          "https://example.com/bar",
				"response_type":         "code",
				"code_challenge":        "123",
				"code_challenge_method": "plain",
				"scope":                 "openid email profile",
			},
		},
		{
			name: "PKCE code_challenge_method default plain",
			clients: []storage.Client{
				{
					ID:           "bar",
					RedirectURIs: []string{"https://example.com/bar"},
				},
			},
			supportedResponseTypes: []string{"code"},
			queryParams: map[string]string{
				"client_id":      "bar",
				"redirect_uri":   "https://example.com/bar",
				"response_type":  "code",
				"code_challenge": "123",
				"scope":          "openid email profile",
			},
		},
		{
			name: "PKCE code_challenge_method S256",
			clients: []storage.Client{
				{
					ID:           "bar",
					RedirectURIs: []string{"https://example.com/bar"},
				},
			},
			supportedResponseTypes: []string{"code"},
			queryParams: map[string]string{
				"client_id":             "bar",
				"redirect_uri":          "https://example.com/bar",
				"response_type":         "code",
				"code_challenge":        "123",
				"code_challenge_method": "S256",
				"scope":                 "openid email profile",
			},
		},
		{
			name: "PKCE invalid code_challenge_method",
			clients: []storage.Client{
				{
					ID:           "bar",
					RedirectURIs: []string{"https://example.com/bar"},
				},
			},
			supportedResponseTypes: []string{"code"},
			queryParams: map[string]string{
				"client_id":             "bar",
				"redirect_uri":          "https://example.com/bar",
				"response_type":         "code",
				"code_challenge":        "123",
				"code_challenge_method": "invalid_method",
				"scope":                 "openid email profile",
			},
			expectedError: &redirectedAuthErr{Type: errInvalidRequest},
		},
		{
			name: "No response type",
			clients: []storage.Client{
				{
					ID:           "bar",
					RedirectURIs: []string{"https://example.com/bar"},
				},
			},
			supportedResponseTypes: []string{"code"},
			queryParams: map[string]string{
				"client_id":             "bar",
				"redirect_uri":          "https://example.com/bar",
				"code_challenge":        "123",
				"code_challenge_method": "plain",
				"scope":                 "openid email profile",
			},
			expectedError: &redirectedAuthErr{Type: errInvalidRequest},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			httpServer, server := newTestServerMultipleConnectors(ctx, t, func(c *Config) {
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

			_, err := server.parseAuthorizationRequest(req)
			if tc.expectedError == nil {
				if err != nil {
					t.Errorf("%s: expected no error", tc.name)
				}
			} else {
				switch expectedErr := tc.expectedError.(type) {
				case *redirectedAuthErr:
					e, ok := err.(*redirectedAuthErr)
					if !ok {
						t.Fatalf("%s: expected redirectedAuthErr error", tc.name)
					}
					if e.Type != expectedErr.Type {
						t.Errorf("%s: expected error type %v, got %v", tc.name, expectedErr.Type, e.Type)
					}
					if e.RedirectURI != tc.queryParams["redirect_uri"] {
						t.Errorf("%s: expected error to be returned in redirect to %v", tc.name, tc.queryParams["redirect_uri"])
					}
				case *displayedAuthErr:
					e, ok := err.(*displayedAuthErr)
					if !ok {
						t.Fatalf("%s: expected displayedAuthErr error", tc.name)
					}
					if e.Status != expectedErr.Status {
						t.Errorf("%s: expected http status %v, got %v", tc.name, expectedErr.Status, e.Status)
					}
				default:
					t.Fatalf("%s: unsupported error type", tc.name)
				}
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
			wantValid:   false,
		},
		// These special desktop + device + localhost URIs are allowed by default.
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
			redirectURI: "/device/callback",
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
		// Both Public + RedirectURIs configured: Could e.g. be a PKCE-enabled web app.
		{
			client: storage.Client{
				Public:       true,
				RedirectURIs: []string{"http://foo.com/bar"},
			},
			redirectURI: "http://foo.com/bar",
			wantValid:   true,
		},
		{
			client: storage.Client{
				Public:       true,
				RedirectURIs: []string{"http://foo.com/bar"},
			},
			redirectURI: "http://foo.com/bar/baz",
			wantValid:   false,
		},
		// These special desktop + device + localhost URIs are not allowed implicitly when RedirectURIs is non-empty.
		{
			client: storage.Client{
				Public:       true,
				RedirectURIs: []string{"http://foo.com/bar"},
			},
			redirectURI: "urn:ietf:wg:oauth:2.0:oob",
			wantValid:   false,
		},
		{
			client: storage.Client{
				Public:       true,
				RedirectURIs: []string{"http://foo.com/bar"},
			},
			redirectURI: "/device/callback",
			wantValid:   false,
		},
		{
			client: storage.Client{
				Public:       true,
				RedirectURIs: []string{"http://foo.com/bar"},
			},
			redirectURI: "http://localhost:8080/",
			wantValid:   false,
		},
		{
			client: storage.Client{
				Public:       true,
				RedirectURIs: []string{"http://foo.com/bar"},
			},
			redirectURI: "http://localhost:991/bar",
			wantValid:   false,
		},
		{
			client: storage.Client{
				Public:       true,
				RedirectURIs: []string{"http://foo.com/bar"},
			},
			redirectURI: "http://localhost",
			wantValid:   false,
		},
		// These special desktop + device + localhost URIs can still be specified explicitly.
		{
			client: storage.Client{
				Public:       true,
				RedirectURIs: []string{"http://foo.com/bar", "urn:ietf:wg:oauth:2.0:oob"},
			},
			redirectURI: "urn:ietf:wg:oauth:2.0:oob",
			wantValid:   true,
		},
		{
			client: storage.Client{
				Public:       true,
				RedirectURIs: []string{"http://foo.com/bar", "/device/callback"},
			},
			redirectURI: "/device/callback",
			wantValid:   true,
		},
		{
			client: storage.Client{
				Public:       true,
				RedirectURIs: []string{"http://foo.com/bar", "http://localhost:8080/"},
			},
			redirectURI: "http://localhost:8080/",
			wantValid:   true,
		},
		{
			client: storage.Client{
				Public:       true,
				RedirectURIs: []string{"http://foo.com/bar", "http://localhost:991/bar"},
			},
			redirectURI: "http://localhost:991/bar",
			wantValid:   true,
		},
		{
			client: storage.Client{
				Public:       true,
				RedirectURIs: []string{"http://foo.com/bar", "http://localhost"},
			},
			redirectURI: "http://localhost",
			wantValid:   true,
		},
		// Non-localhost URIs are not allowed implicitly.
		{
			client: storage.Client{
				Public: true,
			},
			redirectURI: "http://foo.com/bar",
			wantValid:   false,
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

func TestStorageKeySet(t *testing.T) {
	s := memory.New(logger)
	if err := s.UpdateKeys(func(keys storage.Keys) (storage.Keys, error) {
		keys.SigningKey = &jose.JSONWebKey{
			Key:       testKey,
			KeyID:     "testkey",
			Algorithm: "RS256",
			Use:       "sig",
		}
		keys.SigningKeyPub = &jose.JSONWebKey{
			Key:       testKey.Public(),
			KeyID:     "testkey",
			Algorithm: "RS256",
			Use:       "sig",
		}
		return keys, nil
	}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		tokenGenerator func() (jwt string, err error)
		wantErr        bool
	}{
		{
			name: "valid token",
			tokenGenerator: func() (string, error) {
				signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: testKey}, nil)
				if err != nil {
					return "", err
				}

				jws, err := signer.Sign([]byte("payload"))
				if err != nil {
					return "", err
				}

				return jws.CompactSerialize()
			},
			wantErr: false,
		},
		{
			name: "token signed by different key",
			tokenGenerator: func() (string, error) {
				key, err := rsa.GenerateKey(rand.Reader, 2048)
				if err != nil {
					return "", err
				}

				signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, nil)
				if err != nil {
					return "", err
				}

				jws, err := signer.Sign([]byte("payload"))
				if err != nil {
					return "", err
				}

				return jws.CompactSerialize()
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			jwt, err := tc.tokenGenerator()
			if err != nil {
				t.Fatal(err)
			}

			keySet := &storageKeySet{s}

			_, err = keySet.VerifySignature(context.Background(), jwt)
			if (err != nil && !tc.wantErr) || (err == nil && tc.wantErr) {
				t.Fatalf("wantErr = %v, but got err = %v", tc.wantErr, err)
			}
		})
	}
}

package authflow

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

func TestParseAuthorizationRequest(t *testing.T) {
	tests := []struct {
		name                   string
		clients                []storage.Client
		supportedResponseTypes []string
		pkce                   PKCEConfig

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
			expectedError: &redirectedAuthErr{Type: oauth2.UnsupportedResponseType},
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
			expectedError: &redirectedAuthErr{Type: oauth2.InvalidRequest},
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
			expectedError: &redirectedAuthErr{Type: oauth2.InvalidRequest},
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
			expectedError: &redirectedAuthErr{Type: oauth2.InvalidRequest},
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
			expectedError: &redirectedAuthErr{Type: oauth2.InvalidRequest},
		},
		{
			name: "PKCE enforced, no code_challenge provided",
			clients: []storage.Client{
				{
					ID:           "bar",
					RedirectURIs: []string{"https://example.com/bar"},
				},
			},
			supportedResponseTypes: []string{"code"},
			pkce: PKCEConfig{
				Enforce:                       true,
				CodeChallengeMethodsSupported: []string{"S256", "plain"},
			},
			queryParams: map[string]string{
				"client_id":     "bar",
				"redirect_uri":  "https://example.com/bar",
				"response_type": "code",
				"scope":         "openid email profile",
			},
			expectedError: &redirectedAuthErr{Type: oauth2.InvalidRequest},
		},
		{
			name: "PKCE enforced, code_challenge provided",
			clients: []storage.Client{
				{
					ID:           "bar",
					RedirectURIs: []string{"https://example.com/bar"},
				},
			},
			supportedResponseTypes: []string{"code"},
			pkce: PKCEConfig{
				Enforce:                       true,
				CodeChallengeMethodsSupported: []string{"S256", "plain"},
			},
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
			name: "PKCE only S256 allowed, plain rejected",
			clients: []storage.Client{
				{
					ID:           "bar",
					RedirectURIs: []string{"https://example.com/bar"},
				},
			},
			supportedResponseTypes: []string{"code"},
			pkce: PKCEConfig{
				CodeChallengeMethodsSupported: []string{"S256"},
			},
			queryParams: map[string]string{
				"client_id":             "bar",
				"redirect_uri":          "https://example.com/bar",
				"response_type":         "code",
				"code_challenge":        "123",
				"code_challenge_method": "plain",
				"scope":                 "openid email profile",
			},
			expectedError: &redirectedAuthErr{Type: oauth2.InvalidRequest},
		},
		{
			name: "PKCE only S256 allowed, S256 accepted",
			clients: []storage.Client{
				{
					ID:           "bar",
					RedirectURIs: []string{"https://example.com/bar"},
				},
			},
			supportedResponseTypes: []string{"code"},
			pkce: PKCEConfig{
				CodeChallengeMethodsSupported: []string{"S256"},
			},
			queryParams: map[string]string{
				"client_id":             "bar",
				"redirect_uri":          "https://example.com/bar",
				"response_type":         "code",
				"code_challenge":        "123",
				"code_challenge_method": "S256",
				"scope":                 "openid email profile",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpServer, server := newTestHandler(t, func(c *testFlowConfig) {
				c.SupportedResponseTypes = toResponseTypeSet(tc.supportedResponseTypes)
				c.Storage = storage.WithStaticClients(c.Storage, tc.clients)
				if len(tc.pkce.CodeChallengeMethodsSupported) > 0 || tc.pkce.Enforce {
					c.PKCE = tc.pkce
				}
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

			_, _, err := server.parseAuthorizationRequest(req)
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
		{
			client: storage.Client{
				Public: true,
			},
			redirectURI: "http://127.0.0.1:8080/",
			wantValid:   true,
		},
		{
			client: storage.Client{
				Public: true,
			},
			redirectURI: "http://127.0.0.1:991/bar",
			wantValid:   true,
		},
		{
			client: storage.Client{
				Public: true,
			},
			redirectURI: "http://127.0.0.1",
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

func TestRedirectedAuthErrHandler(t *testing.T) {
	tests := []struct {
		name        string
		redirectURI string
		state       string
		errType     string
		description string
		wantStatus  int
		wantErr     bool
	}{
		{
			name:        "valid redirect uri with error parameters",
			redirectURI: "https://example.com/callback",
			state:       "state123",
			errType:     oauth2.InvalidRequest,
			description: "Invalid request parameter",
			wantStatus:  http.StatusSeeOther,
			wantErr:     false,
		},
		{
			name:        "valid redirect uri with query params",
			redirectURI: "https://example.com/callback?existing=param&another=value",
			state:       "state456",
			errType:     oauth2.AccessDenied,
			description: "User denied access",
			wantStatus:  http.StatusSeeOther,
			wantErr:     false,
		},
		{
			name:        "valid redirect uri without description",
			redirectURI: "https://example.com/callback",
			state:       "state789",
			errType:     oauth2.ServerError,
			description: "",
			wantStatus:  http.StatusSeeOther,
			wantErr:     false,
		},
		{
			name:        "invalid redirect uri",
			redirectURI: "not a valid url ://",
			state:       "state",
			errType:     oauth2.InvalidRequest,
			description: "Test error",
			wantStatus:  http.StatusBadRequest,
			wantErr:     true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := &redirectedAuthErr{
				State:       tc.state,
				RedirectURI: tc.redirectURI,
				Type:        tc.errType,
				Description: tc.description,
			}

			handler := err.Handler()
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)

			handler.ServeHTTP(w, r)

			if w.Code != tc.wantStatus {
				t.Errorf("expected status %d, got %d", tc.wantStatus, w.Code)
			}

			if tc.wantStatus == http.StatusSeeOther {
				// Verify the redirect location is a valid URL
				location := w.Header().Get("Location")
				if location == "" {
					t.Fatalf("expected Location header, got empty string")
				}

				// Parse the redirect URL to verify it's valid
				redirectURL, parseErr := url.Parse(location)
				if parseErr != nil {
					t.Fatalf("invalid redirect URL: %v", parseErr)
				}

				// Verify error parameters are present in the query string
				query := redirectURL.Query()
				if query.Get("state") != tc.state {
					t.Errorf("expected state %q, got %q", tc.state, query.Get("state"))
				}
				if query.Get("error") != tc.errType {
					t.Errorf("expected error type %q, got %q", tc.errType, query.Get("error"))
				}
				if tc.description != "" && query.Get("error_description") != tc.description {
					t.Errorf("expected error_description %q, got %q", tc.description, query.Get("error_description"))
				}

				// Verify that existing query parameters are preserved
				if tc.name == "valid redirect uri with query params" {
					if query.Get("existing") != "param" {
						t.Errorf("expected existing parameter 'param', got %q", query.Get("existing"))
					}
					if query.Get("another") != "value" {
						t.Errorf("expected another parameter 'value', got %q", query.Get("another"))
					}
				}
			}
		})
	}
}

// signTestIDToken creates a signed JWT with the given claims using the test key.
func signTestIDToken(t *testing.T, claims interface{}) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	require.NoError(t, err)

	joseSigner, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: testKey}, nil)
	require.NoError(t, err)

	jws, err := joseSigner.Sign(payload)
	require.NoError(t, err)

	token, err := jws.CompactSerialize()
	require.NoError(t, err)
	return token
}

func TestValidateIDTokenHint(t *testing.T) {
	sig, err := signer.NewMockSigner(testKey)
	require.NoError(t, err)

	issuerURL, err := url.Parse("https://issuer.example.com")
	require.NoError(t, err)

	s := &Handler{
		Signer:    sig,
		IssuerURL: oauth2.IssuerURL{URL: *issuerURL},
		Logger:    slog.Default(),
	}

	now := time.Now()

	t.Run("valid hint (not expired)", func(t *testing.T) {
		token := signTestIDToken(t, tokens.IDTokenClaims{
			Issuer:  "https://issuer.example.com",
			Subject: "CgNmb28SA2Jhcg",
			Expiry:  now.Add(1 * time.Hour).Unix(),
		})
		idToken, err := s.validateIDTokenHint(t.Context(), token)
		require.NoError(t, err)
		assert.Equal(t, "CgNmb28SA2Jhcg", idToken.Subject)
	})

	t.Run("valid hint (expired)", func(t *testing.T) {
		token := signTestIDToken(t, tokens.IDTokenClaims{
			Issuer:  "https://issuer.example.com",
			Subject: "CgNmb28SA2Jhcg",
			Expiry:  now.Add(-1 * time.Hour).Unix(),
		})
		idToken, err := s.validateIDTokenHint(t.Context(), token)
		require.NoError(t, err)
		assert.Equal(t, "CgNmb28SA2Jhcg", idToken.Subject)
	})

	t.Run("invalid signature", func(t *testing.T) {
		otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		payload, err := json.Marshal(tokens.IDTokenClaims{
			Issuer:  "https://issuer.example.com",
			Subject: "CgNmb28SA2Jhcg",
			Expiry:  now.Add(1 * time.Hour).Unix(),
		})
		require.NoError(t, err)

		joseSigner, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: otherKey}, nil)
		require.NoError(t, err)
		jws, err := joseSigner.Sign(payload)
		require.NoError(t, err)
		token, err := jws.CompactSerialize()
		require.NoError(t, err)

		_, err = s.validateIDTokenHint(t.Context(), token)
		assert.Error(t, err)
	})

	t.Run("wrong issuer", func(t *testing.T) {
		token := signTestIDToken(t, tokens.IDTokenClaims{
			Issuer:  "https://wrong-issuer.example.com",
			Subject: "CgNmb28SA2Jhcg",
			Expiry:  now.Add(1 * time.Hour).Unix(),
		})
		_, err := s.validateIDTokenHint(t.Context(), token)
		assert.Error(t, err)
	})

	t.Run("malformed token", func(t *testing.T) {
		_, err := s.validateIDTokenHint(t.Context(), "not-a-valid-jwt")
		assert.Error(t, err)
	})
}

func TestSessionMatchesHint(t *testing.T) {
	// tokens.GenSubject("foo", "bar") == "CgNmb28SA2Jhcg" (from TestGetSubject)
	assert.True(t, sessionMatchesHint(&storage.AuthSession{UserID: "foo", ConnectorID: "bar"}, "CgNmb28SA2Jhcg"))
	assert.False(t, sessionMatchesHint(&storage.AuthSession{UserID: "other", ConnectorID: "bar"}, "CgNmb28SA2Jhcg"))
	assert.False(t, sessionMatchesHint(&storage.AuthSession{UserID: "foo", ConnectorID: "other"}, "CgNmb28SA2Jhcg"))
	assert.False(t, sessionMatchesHint(nil, "CgNmb28SA2Jhcg"))
}

func TestParseAuthorizationRequest_IDTokenHint(t *testing.T) {
	sig, err := signer.NewMockSigner(testKey)
	require.NoError(t, err)

	now := time.Now()

	t.Run("valid id_token_hint populates subject", func(t *testing.T) {
		httpServer, server := newTestHandler(t, func(c *testFlowConfig) {
			c.SupportedResponseTypes = map[string]bool{"code": true}
			c.Storage = storage.WithStaticClients(c.Storage, []storage.Client{
				{ID: "foo", RedirectURIs: []string{"https://example.com/foo"}},
			})
			c.Signer = sig
		})
		defer httpServer.Close()

		token := signTestIDToken(t, tokens.IDTokenClaims{
			Issuer:  httpServer.URL,
			Subject: "CgNmb28SA2Jhcg",
			Expiry:  now.Add(1 * time.Hour).Unix(),
		})

		params := url.Values{
			"client_id":     {"foo"},
			"redirect_uri":  {"https://example.com/foo"},
			"response_type": {"code"},
			"scope":         {"openid"},
			"id_token_hint": {token},
		}
		req := httptest.NewRequest("GET", httpServer.URL+"/auth?"+params.Encode(), nil)

		_, hintSubject, err := server.parseAuthorizationRequest(req)
		require.NoError(t, err)
		assert.Equal(t, "CgNmb28SA2Jhcg", hintSubject)
	})

	t.Run("invalid id_token_hint returns error", func(t *testing.T) {
		httpServer, server := newTestHandler(t, func(c *testFlowConfig) {
			c.SupportedResponseTypes = map[string]bool{"code": true}
			c.Storage = storage.WithStaticClients(c.Storage, []storage.Client{
				{ID: "foo", RedirectURIs: []string{"https://example.com/foo"}},
			})
			c.Signer = sig
		})
		defer httpServer.Close()

		params := url.Values{
			"client_id":     {"foo"},
			"redirect_uri":  {"https://example.com/foo"},
			"response_type": {"code"},
			"scope":         {"openid"},
			"id_token_hint": {"invalid-token"},
		}
		req := httptest.NewRequest("GET", httpServer.URL+"/auth?"+params.Encode(), nil)

		_, _, err := server.parseAuthorizationRequest(req)
		require.Error(t, err)
		redirectErr, ok := err.(*redirectedAuthErr)
		require.True(t, ok)
		assert.Equal(t, oauth2.InvalidRequest, redirectErr.Type)
	})

	t.Run("no id_token_hint leaves subject empty", func(t *testing.T) {
		httpServer, server := newTestHandler(t, func(c *testFlowConfig) {
			c.SupportedResponseTypes = map[string]bool{"code": true}
			c.Storage = storage.WithStaticClients(c.Storage, []storage.Client{
				{ID: "foo", RedirectURIs: []string{"https://example.com/foo"}},
			})
		})
		defer httpServer.Close()

		params := url.Values{
			"client_id":     {"foo"},
			"redirect_uri":  {"https://example.com/foo"},
			"response_type": {"code"},
			"scope":         {"openid"},
		}
		req := httptest.NewRequest("GET", httpServer.URL+"/auth?"+params.Encode(), nil)

		_, hintSubject, err := server.parseAuthorizationRequest(req)
		require.NoError(t, err)
		assert.Equal(t, "", hintSubject)
	})
}

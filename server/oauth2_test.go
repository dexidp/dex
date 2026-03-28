package server

import (
	"context"
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

	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func TestGetClientID(t *testing.T) {
	cid, err := getClientID(audience{}, "")
	require.Equal(t, "", cid)
	require.Equal(t, "no audience is set, could not find ClientID", err.Error())

	cid, err = getClientID(audience{"a"}, "")
	require.Equal(t, "a", cid)
	require.NoError(t, err)

	cid, err = getClientID(audience{"a", "b"}, "azp")
	require.Equal(t, "azp", cid)
	require.NoError(t, err)
}

func TestGetAudience(t *testing.T) {
	aud := getAudience("client-id", []string{})
	require.Equal(t, aud, audience{"client-id"})

	aud = getAudience("client-id", []string{"ascope"})
	require.Equal(t, aud, audience{"client-id"})

	aud = getAudience("client-id", []string{"ascope", "audience:server:client_id:aa", "audience:server:client_id:bb"})
	require.Equal(t, aud, audience{"aa", "bb", "client-id"})
}

func TestGetSubject(t *testing.T) {
	sub, err := genSubject("foo", "bar")
	require.Equal(t, "CgNmb28SA2Jhcg", sub)
	require.NoError(t, err)
}

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
			expectedError: &redirectedAuthErr{Type: errInvalidRequest},
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
			expectedError: &redirectedAuthErr{Type: errInvalidRequest},
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
			httpServer, server := newTestServerMultipleConnectors(t, func(c *Config) {
				c.SupportedResponseTypes = tc.supportedResponseTypes
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

func TestSignerKeySet(t *testing.T) {
	logger := newLogger(t)
	s := memory.New(logger)
	if err := s.UpdateKeys(t.Context(), func(keys storage.Keys) (storage.Keys, error) {
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

			// Create a mock signer for testing
			sig, err := signer.NewMockSigner(testKey)
			if err != nil {
				t.Fatal(err)
			}

			keySet := &signerKeySet{
				signer: sig,
			}

			_, err = keySet.VerifySignature(t.Context(), jwt)
			if (err != nil && !tc.wantErr) || (err == nil && tc.wantErr) {
				t.Fatalf("wantErr = %v, but got err = %v", tc.wantErr, err)
			}
		})
	}
}

func TestSignerKeySetWithES256LocalSigner(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.DiscardHandler)
	store := memory.New(logger)

	localConfig := signer.LocalConfig{
		KeysRotationPeriod: time.Hour.String(),
		Algorithm:          jose.ES256,
	}
	sig, err := localConfig.Open(ctx, store, time.Hour, time.Now, logger)
	require.NoError(t, err)

	sig.Start(ctx)

	jwt, err := sig.Sign(ctx, []byte("payload"))
	require.NoError(t, err)

	keySet := &signerKeySet{signer: sig}
	payload, err := keySet.VerifySignature(ctx, jwt)
	require.NoError(t, err)
	require.Equal(t, []byte("payload"), payload)
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
			errType:     errInvalidRequest,
			description: "Invalid request parameter",
			wantStatus:  http.StatusSeeOther,
			wantErr:     false,
		},
		{
			name:        "valid redirect uri with query params",
			redirectURI: "https://example.com/callback?existing=param&another=value",
			state:       "state456",
			errType:     errAccessDenied,
			description: "User denied access",
			wantStatus:  http.StatusSeeOther,
			wantErr:     false,
		},
		{
			name:        "valid redirect uri without description",
			redirectURI: "https://example.com/callback",
			state:       "state789",
			errType:     errServerError,
			description: "",
			wantStatus:  http.StatusSeeOther,
			wantErr:     false,
		},
		{
			name:        "invalid redirect uri",
			redirectURI: "not a valid url ://",
			state:       "state",
			errType:     errInvalidRequest,
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

	s := &Server{
		signer:    sig,
		issuerURL: *issuerURL,
		logger:    slog.Default(),
	}

	now := time.Now()

	t.Run("valid hint (not expired)", func(t *testing.T) {
		token := signTestIDToken(t, idTokenClaims{
			Issuer:  "https://issuer.example.com",
			Subject: "CgNmb28SA2Jhcg",
			Expiry:  now.Add(1 * time.Hour).Unix(),
		})
		sub, err := s.validateIDTokenHint(t.Context(), token)
		require.NoError(t, err)
		assert.Equal(t, "CgNmb28SA2Jhcg", sub)
	})

	t.Run("valid hint (expired)", func(t *testing.T) {
		token := signTestIDToken(t, idTokenClaims{
			Issuer:  "https://issuer.example.com",
			Subject: "CgNmb28SA2Jhcg",
			Expiry:  now.Add(-1 * time.Hour).Unix(),
		})
		sub, err := s.validateIDTokenHint(t.Context(), token)
		require.NoError(t, err)
		assert.Equal(t, "CgNmb28SA2Jhcg", sub)
	})

	t.Run("invalid signature", func(t *testing.T) {
		otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		payload, err := json.Marshal(idTokenClaims{
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
		token := signTestIDToken(t, idTokenClaims{
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

func TestNewIDTokenUsesStoredAlgorithmUntilNextRotation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.DiscardHandler)
	store := memory.New(logger)

	now := time.Now().UTC()
	err := store.UpdateKeys(ctx, func(keys storage.Keys) (storage.Keys, error) {
		keys.SigningKey = &jose.JSONWebKey{
			Key:       testKey,
			KeyID:     "legacy-rs256",
			Algorithm: string(jose.RS256),
			Use:       "sig",
		}
		keys.SigningKeyPub = &jose.JSONWebKey{
			Key:       testKey.Public(),
			KeyID:     "legacy-rs256",
			Algorithm: string(jose.RS256),
			Use:       "sig",
		}
		keys.NextRotation = now.Add(time.Hour)
		return keys, nil
	})
	require.NoError(t, err)

	localConfig := signer.LocalConfig{
		KeysRotationPeriod: time.Hour.String(),
		Algorithm:          jose.ES256,
	}
	sig, err := localConfig.Open(ctx, store, time.Hour, func() time.Time { return now }, logger)
	require.NoError(t, err)

	sig.Start(ctx)

	alg, err := sig.Algorithm(ctx)
	require.NoError(t, err)
	require.Equal(t, jose.RS256, alg)

	issuerURL, err := url.Parse("https://issuer.example.com")
	require.NoError(t, err)

	s := &Server{
		signer:           sig,
		issuerURL:        *issuerURL,
		logger:           logger,
		now:              func() time.Time { return now },
		idTokensValidFor: time.Hour,
	}

	accessToken := "test-access-token"
	code := "test-auth-code"
	idToken, _, err := s.newIDToken(
		ctx,
		"test-client",
		storage.Claims{UserID: "1", Username: "jane"},
		[]string{"openid"},
		"nonce",
		accessToken,
		code,
		"test",
		time.Time{},
	)
	require.NoError(t, err)

	keys, err := sig.ValidationKeys(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, keys)

	jws, err := jose.ParseSigned(idToken, []jose.SignatureAlgorithm{jose.RS256})
	require.NoError(t, err)
	require.Len(t, jws.Signatures, 1)
	require.Equal(t, string(jose.RS256), jws.Signatures[0].Protected.Algorithm)

	payload, err := jws.Verify(keys[0])
	require.NoError(t, err)

	var claims struct {
		AccessTokenHash string `json:"at_hash"`
		CodeHash        string `json:"c_hash"`
	}
	err = json.Unmarshal(payload, &claims)
	require.NoError(t, err)

	wantAtHash, err := accessTokenHash(jose.RS256, accessToken)
	require.NoError(t, err)
	require.Equal(t, wantAtHash, claims.AccessTokenHash)

	wantCodeHash, err := accessTokenHash(jose.RS256, code)
	require.NoError(t, err)
	require.Equal(t, wantCodeHash, claims.CodeHash)
}

func TestNewIDTokenContainsJTI(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.DiscardHandler)
	store := memory.New(logger)

	now := time.Now().UTC()
	err := store.UpdateKeys(ctx, func(keys storage.Keys) (storage.Keys, error) {
		keys.SigningKey = &jose.JSONWebKey{
			Key:       testKey,
			KeyID:     "test-rs256",
			Algorithm: string(jose.RS256),
			Use:       "sig",
		}
		keys.SigningKeyPub = &jose.JSONWebKey{
			Key:       testKey.Public(),
			KeyID:     "test-rs256",
			Algorithm: string(jose.RS256),
			Use:       "sig",
		}
		keys.NextRotation = now.Add(time.Hour)
		return keys, nil
	})
	require.NoError(t, err)

	localConfig := signer.LocalConfig{
		KeysRotationPeriod: time.Hour.String(),
		Algorithm:          jose.RS256,
	}
	sig, err := localConfig.Open(ctx, store, time.Hour, func() time.Time { return now }, logger)
	require.NoError(t, err)

	sig.Start(ctx)

	issuerURL, err := url.Parse("https://issuer.example.com")
	require.NoError(t, err)

	s := &Server{
		signer:           sig,
		issuerURL:        *issuerURL,
		logger:           logger,
		now:              func() time.Time { return now },
		idTokensValidFor: time.Hour,
	}

	keys, err := sig.ValidationKeys(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, keys)

	extractJTI := func(t *testing.T, idToken string) string {
		t.Helper()
		jws, err := jose.ParseSigned(idToken, []jose.SignatureAlgorithm{jose.RS256})
		require.NoError(t, err)
		payload, err := jws.Verify(keys[0])
		require.NoError(t, err)
		var claims struct {
			JTI string `json:"jti"`
		}
		err = json.Unmarshal(payload, &claims)
		require.NoError(t, err)
		return claims.JTI
	}

	token1, _, err := s.newIDToken(ctx, "client", storage.Claims{UserID: "1", Username: "alice"}, []string{"openid"}, "n1", "", "", "mock", time.Time{})
	require.NoError(t, err)

	token2, _, err := s.newIDToken(ctx, "client", storage.Claims{UserID: "1", Username: "alice"}, []string{"openid"}, "n2", "", "", "mock", time.Time{})
	require.NoError(t, err)

	jti1 := extractJTI(t, token1)
	jti2 := extractJTI(t, token2)

	assert.NotEmpty(t, jti1, "jti claim must be present and non-empty")
	assert.NotEmpty(t, jti2, "jti claim must be present and non-empty")
	assert.NotEqual(t, jti1, jti2, "each token must have a unique jti")
}

func TestSessionMatchesHint(t *testing.T) {
	// genSubject("foo", "bar") == "CgNmb28SA2Jhcg" (from TestGetSubject)
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
		httpServer, server := newTestServerMultipleConnectors(t, func(c *Config) {
			c.SupportedResponseTypes = []string{"code"}
			c.Storage = storage.WithStaticClients(c.Storage, []storage.Client{
				{ID: "foo", RedirectURIs: []string{"https://example.com/foo"}},
			})
			c.Signer = sig
		})
		defer httpServer.Close()

		token := signTestIDToken(t, idTokenClaims{
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
		httpServer, server := newTestServerMultipleConnectors(t, func(c *Config) {
			c.SupportedResponseTypes = []string{"code"}
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
		assert.Equal(t, errInvalidRequest, redirectErr.Type)
	})

	t.Run("no id_token_hint leaves subject empty", func(t *testing.T) {
		httpServer, server := newTestServerMultipleConnectors(t, func(c *Config) {
			c.SupportedResponseTypes = []string{"code"}
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

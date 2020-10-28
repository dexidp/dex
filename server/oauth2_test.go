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

	lru "github.com/hashicorp/golang-lru"
	"github.com/stretchr/testify/assert"
	"gopkg.in/square/go-jose.v2"

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

		wantErr bool
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
			wantErr: true,
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
			wantErr: true,
		},
	}

	for _, tc := range tests {
		func() {
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
			if err != nil && !tc.wantErr {
				t.Errorf("%s: %v", tc.name, err)
			}
			if err == nil && tc.wantErr {
				t.Errorf("%s: expected error", tc.name)
			}
		}()
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

func TestSplitHttpRedirectUrl(t *testing.T) {
	//  if strings.Index(clientRedirectUrlSpec, "*") == -1 {
	//  no wildcard characters to consider
	//  return nil
	//}

	tests := []struct {
		redirectURI     string
		expectValid     bool
		expectHostSplit *HostSplit
	}{
		{
			redirectURI: "http://*.example.com/%23267#222",
			expectValid: true,
			expectHostSplit: &HostSplit{
				HostSuffix:      "example.com",
				SubDomainPrefix: "*",
			},
		},
		{
			// ip addresses cannot
			redirectURI: "http://1.2.3.4/",
			expectValid: false,
		},
		{
			// cannot act on top-level domain
			redirectURI: "http://*.com/",
			expectValid: false,
		},
		{
			// no wildcard
			redirectURI: "http://example.com/",
			expectValid: false,
		},
		{
			redirectURI: "http://1.2.3.4",
			expectValid: false, // no trailing slash
		},
		{
			// ipv6 address not used for wildcard (dns only)
			redirectURI: "https://[fdda:5cc1:23:4::1f]/foo",
			expectValid: false,
		},
	}

	tAssert := assert.New(t)
	for _, test := range tests {
		testRedirectURL, err := url.Parse(test.redirectURI)
		tAssert.Nilf(err, "Invalid test url %v", test.redirectURI)
		tAssert.NotNilf(testRedirectURL, "Invalid test url %v", test.redirectURI)
		result := splitClobberHTTPRedirectURL(testRedirectURL)
		tAssert.Equal(test.expectValid, result != nil, "Valid result expected for %v", test.redirectURI)
		if test.expectHostSplit != nil {
			tAssert.Equal(result, test.expectHostSplit, "Host split validation failed for %v", test.redirectURI)
		}
		if result != nil {
			fullResult := createClientRedirectClobberMatcher(test.redirectURI)
			tAssert.True(fullResult.HostMatcher != nil)
			tAssert.Equal(fullResult.ClientRedirectURL, testRedirectURL)
		}
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
			// invalid schema
			client: storage.Client{
				RedirectURIs: []string{"http://foo.com/bar"},
			},
			redirectURI: "https://foo.com/bar",
			wantValid:   false,
		},
		{
			// invalid schema with pattern
			client: storage.Client{
				RedirectURIs: []string{"http://*.foo.com/bar"},
			},
			redirectURI: "https://test.foo.com/bar",
			wantValid:   false,
		},
		{
			// check https is valid protocol
			client: storage.Client{
				RedirectURIs: []string{"https://foo.com/bar/baz"},
			},
			redirectURI: "https://foo.com/bar/baz",
			wantValid:   true,
		},
		{
			client: storage.Client{
				RedirectURIs: []string{"http://foo.com/bar"},
			},
			redirectURI: "http://foo.com/bar/baz",
			wantValid:   false,
		},
		{
			// invalid path with pattern
			client: storage.Client{
				RedirectURIs: []string{"http://*.foo.com/bar"},
			},
			redirectURI: "http://test.foo.com/baz",
			wantValid:   false,
		},
		{
			client: storage.Client{
				RedirectURIs: []string{"http://*.foo.com/bar"},
			},
			redirectURI: "http://abc.foo.com/bar",
			wantValid:   true,
		},
		{
			client: storage.Client{
				RedirectURIs: []string{"http://abc.foo.com/bar"},
			},
			redirectURI: "http://abc.foo.com/bar",
			wantValid:   true,
		},
		{
			client: storage.Client{
				RedirectURIs: []string{"http://b*.foo.com/bar"},
			},
			redirectURI: "http://abc.foo.com/bar",
			wantValid:   false,
		},
		{
			client: storage.Client{
				RedirectURIs: []string{"http://b*.foo.com/bar"},
			},
			redirectURI: "http://b.foo.com/bar",
			wantValid:   false,
		},
		{
			client: storage.Client{
				RedirectURIs: []string{"http://b*.foo.com/bar"},
			},
			redirectURI: "http://bar.foo.com/bar",
			wantValid:   true,
		},
		{
			// URL must use http: or https: protocol
			client: storage.Client{
				RedirectURIs: []string{"unknown://*.foo.com/bar"},
			},
			redirectURI: "unknown://abc.foo.com/bar",
			wantValid:   true,
		},
		{
			// wildcard can be located in subdomain as long as not in top two domains
			//  https://abc.*.foo.com ok.
			client: storage.Client{
				RedirectURIs: []string{"http://abc.*.foo.com/bar"},
			},
			redirectURI: "http://abc.123.foo.com/bar",
			wantValid:   true,
		},
		{
			client: storage.Client{
				RedirectURIs: []string{"http://**.foo.com/bar"},
			},
			redirectURI: "http://abc.123.foo.com/bar",
			wantValid:   true,
		},
		{
			client: storage.Client{
				RedirectURIs: []string{"http://**.foo.com/bar"},
			},
			redirectURI: "http://test.foo.com/bar",
			wantValid:   true,
		},
		{
			client: storage.Client{
				RedirectURIs: []string{"http://a**.foo.com/bar"},
			},
			redirectURI: "http://abc.123.foo.com/bar",
			wantValid:   true,
		},
		{
			client: storage.Client{
				RedirectURIs: []string{"http://a**.foo.com/bar"},
			},
			redirectURI: "http://a.foo.com/bar",
			wantValid:   false,
		},
		{
			// wildcard may be prefixed and/or suffixed with additional valid hostname characters
			//  https://pre-*-post.foo.com will work
			client: storage.Client{
				RedirectURIs: []string{"http://pre-*-post.foo.com/bar"},
			},
			redirectURI: "http://pre-dinner-post.foo.com/bar",
			wantValid:   true,
		},
		{
			// valid wildcard will not match a URL more than one subdomain level in place of the wildcard
			//  https://*.foo.com will not work with https://123.abc.foo.com.
			client: storage.Client{
				RedirectURIs: []string{"https://*.foo.com/"},
			},
			redirectURI: "https://123.abc.foo.com/",
			wantValid:   false,
		},
		{
			// check escaping
			client: storage.Client{
				RedirectURIs: []string{"https://*.foo.com/"},
			},
			redirectURI: "https://abc.foo0com/",
			wantValid:   false,
		},
		{
			// with port
			client: storage.Client{
				RedirectURIs: []string{"http://*.sld.com:6000/path"},
			},
			redirectURI: "http://test.sld.com:6000/path",
			wantValid:   true,
		},
		{
			// negative logic with port
			client: storage.Client{
				RedirectURIs: []string{"http://*.sld.com:6000/path"},
			},
			redirectURI: "http://test.sld.com/path",
			wantValid:   false,
		},
		{
			// wildcard must be located in a subdomain within a hostname component.  http://*.com is not permitted.
			client: storage.Client{
				RedirectURIs: []string{"http://*.*.com/bar"},
			},
			redirectURI: "http://test.foo.com/bar",
			wantValid:   false,
		},
		{
			// partial wildcard in SLD is prohibited
			client: storage.Client{
				RedirectURIs: []string{"http://*.foo*.com/bar"},
			},
			redirectURI: "http://test.foo.com/bar",
			wantValid:   false,
		},
		{
			// wildcard in TLD is prohibited
			client: storage.Client{
				RedirectURIs: []string{"http://*.example.*/bar"},
			},
			redirectURI: "http://test.foo.com/bar",
			wantValid:   false,
		},
		{
			// partial wildcard in TLD is prohibited
			client: storage.Client{
				RedirectURIs: []string{"http://*.example.io*/bar"},
			},
			redirectURI: "http://test.foo.com/bar",
			wantValid:   false,
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
			// github.com/dexidp/dex/issues/1300 allow public redirect URLs
			client: storage.Client{
				Public: true,
			},
			redirectURI: "https://localhost",
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

	maxCacheSize := 3
	wildcardCache, _ := lru.NewARC(maxCacheSize)

	for _, test := range tests {
		got := validateRedirectURI(test.client, wildcardCache, test.redirectURI)
		if got != test.wantValid {
			t.Errorf("client=%#v, redirectURI=%q, wanted valid=%t, got=%t",
				test.client, test.redirectURI, test.wantValid, got)
		}
	}

	if wildcardCache.Len() == 0 {
		// check eviction policed
		t.Errorf("cache should be in use")
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

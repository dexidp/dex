package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/coreos/go-oidc/oauth2"
	"github.com/coreos/go-oidc/oidc"
	"github.com/kylelemons/godebug/pretty"
)

func TestClientRegistration(t *testing.T) {
	tests := []struct {
		body string
		code int
	}{
		{"", http.StatusBadRequest},
		{
			`{
				"redirect_uris": [
					"https://client.example.org/callback",
					"https://client.example.org/callback2"
				]
			}`,
			http.StatusCreated,
		},
		{
			// Requesting unsupported client metadata fields (user_info_encrypted).
			`{
				"application_type": "web",
				"redirect_uris":[
					"https://client.example.org/callback",
					"https://client.example.org/callback2"
				],
				"client_name": "My Example",
				"logo_uri": "https://client.example.org/logo.png",
				"subject_type": "pairwise",
				"sector_identifier_uri": "https://other.example.net/file_of_redirect_uris.json",
				"token_endpoint_auth_method": "client_secret_basic",
				"jwks_uri": "https://client.example.org/my_public_keys.jwks",
				"userinfo_encrypted_response_alg": "RSA1_5",
				"userinfo_encrypted_response_enc": "A128CBC-HS256",
				"contacts": ["ve7jtb@example.org", "mary@example.org"],
				"request_uris": [
					"https://client.example.org/rf.txt#qpXaRLh_n93TTR9F252ValdatUQvQiJi5BDub2BeznA"					]
			}`,
			http.StatusBadRequest,
		},
		{
			`{
				"application_type": "web",
				"redirect_uris":[
					"https://client.example.org/callback",
					"https://client.example.org/callback2"
				],
				"client_name": "My Example",
				"logo_uri": "https://client.example.org/logo.png",
				"subject_type": "pairwise",
				"sector_identifier_uri": "https://other.example.net/file_of_redirect_uris.json",
				"token_endpoint_auth_method": "client_secret_basic",
				"jwks_uri": "https://client.example.org/my_public_keys.jwks",
				"contacts": ["ve7jtb@example.org", "mary@example.org"],
				"request_uris": [
					"https://client.example.org/rf.txt#qpXaRLh_n93TTR9F252ValdatUQvQiJi5BDub2BeznA"					]
			}`,
			http.StatusCreated,
		},
	}

	var handler http.Handler
	f := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	})
	testServer := httptest.NewServer(f)

	issuerURL, err := url.Parse(testServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer testServer.Close()

	for i, tt := range tests {
		fixtures, err := makeTestFixtures()
		if err != nil {
			t.Fatal(err)
		}
		fixtures.srv.IssuerURL = *issuerURL
		fixtures.srv.EnableClientRegistration = true

		handler = fixtures.srv.HTTPHandler()

		err = func() error {
			// GET provider config through discovery URL.
			resp, err := http.Get(testServer.URL + "/.well-known/openid-configuration")
			if err != nil {
				return fmt.Errorf("GET config: %v", err)
			}
			var cfg oidc.ProviderConfig
			err = json.NewDecoder(resp.Body).Decode(&cfg)
			resp.Body.Close()
			if err != nil {
				return fmt.Errorf("decode resp: %v", err)
			}

			if cfg.RegistrationEndpoint == nil {
				return errors.New("registration endpoint not available")
			}

			// POST registration request to registration endpoint.
			body := strings.NewReader(tt.body)
			resp, err = http.Post(cfg.RegistrationEndpoint.String(), "application/json", body)
			if err != nil {
				return fmt.Errorf("POSTing client metadata: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tt.code {
				return fmt.Errorf("expected status code=%d, got=%d", tt.code, resp.StatusCode)
			}

			if resp.StatusCode != http.StatusCreated {
				var oauthErr oauth2.Error
				if err := json.NewDecoder(resp.Body).Decode(&oauthErr); err != nil {
					return fmt.Errorf("failed to decode oauth2 error: %v", err)
				}
				if oauthErr.Type == "" {
					return fmt.Errorf("got oauth2 error with no 'error' field")
				}
				return nil
			}

			// Read registration response.
			var r oidc.ClientRegistrationResponse
			if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
				return fmt.Errorf("decode response: %v", err)
			}
			if r.ClientID == "" {
				return fmt.Errorf("no client id in registration response")
			}

			metadata, err := fixtures.clientRepo.Metadata(nil, r.ClientID)
			if err != nil {
				return fmt.Errorf("failed to lookup client id after creation")
			}

			if diff := pretty.Compare(&metadata, &r.ClientMetadata); diff != "" {
				return fmt.Errorf("metadata in response did not match metadata in db: %s", diff)
			}

			return nil
		}()
		if err != nil {
			t.Errorf("case %d: %v", i, err)
		}
	}
}

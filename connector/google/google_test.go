package google

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/option"

	"github.com/dexidp/dex/connector"
)

var (
	//			groups_0
	//		┌───────┤
	//  groups_2 groups_1
	//		│		├────────┐
	//		└──	 user_1	  user_2
	testGroups = map[string][]*admin.Group{
		"user_1@dexidp.com":   {{Email: "groups_2@dexidp.com"}, {Email: "groups_1@dexidp.com"}},
		"user_2@dexidp.com":   {{Email: "groups_1@dexidp.com"}},
		"groups_1@dexidp.com": {{Email: "groups_0@dexidp.com"}},
		"groups_2@dexidp.com": {{Email: "groups_0@dexidp.com"}},
		"groups_0@dexidp.com": {},
	}
	callCounter = make(map[string]int)
)

func testSetup() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/admin/directory/v1/groups/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		userKey := r.URL.Query().Get("userKey")
		if groups, ok := testGroups[userKey]; ok {
			json.NewEncoder(w).Encode(admin.Groups{Groups: groups})
			callCounter[userKey]++
		}
	})

	return httptest.NewServer(mux)
}

func newConnector(config *Config) (*googleConnector, error) {
	log := slog.New(slog.DiscardHandler)
	conn, err := config.Open("id", log)
	if err != nil {
		return nil, err
	}

	googleConn, ok := conn.(*googleConnector)
	if !ok {
		return nil, fmt.Errorf("failed to convert to googleConnector")
	}
	return googleConn, nil
}

func tempServiceAccountKey() (string, error) {
	fd, err := os.CreateTemp("", "google_service_account_key")
	if err != nil {
		return "", err
	}
	defer fd.Close()
	err = json.NewEncoder(fd).Encode(map[string]string{
		"type":                 "service_account",
		"project_id":           "sample-project",
		"private_key_id":       "sample-key-id",
		"private_key":          "-----BEGIN PRIVATE KEY-----\nsample-key\n-----END PRIVATE KEY-----\n",
		"client_id":            "sample-client-id",
		"client_x509_cert_url": "localhost",
	})
	return fd.Name(), err
}

func TestOpen(t *testing.T) {
	ts := testSetup()
	defer ts.Close()

	type testCase struct {
		config      *Config
		expectedErr string

		// string to set in GOOGLE_APPLICATION_CREDENTIALS. As local development environments can
		// already contain ADC, test cases will be built upon this setting this env variable
		adc string
	}

	serviceAccountFilePath, err := tempServiceAccountKey()
	assert.Nil(t, err)

	for name, reference := range map[string]testCase{
		"missing_admin_email": {
			config: &Config{
				ClientID:               "testClient",
				ClientSecret:           "testSecret",
				RedirectURI:            ts.URL + "/callback",
				Scopes:                 []string{"openid", "groups"},
				ServiceAccountFilePath: serviceAccountFilePath,
			},
			expectedErr: "requires the domainToAdminEmail",
		},
		"service_account_key_not_found": {
			config: &Config{
				ClientID:               "testClient",
				ClientSecret:           "testSecret",
				RedirectURI:            ts.URL + "/callback",
				Scopes:                 []string{"openid", "groups"},
				DomainToAdminEmail:     map[string]string{"*": "foo@bar.com"},
				ServiceAccountFilePath: "not_found.json",
			},
			expectedErr: "error reading credentials",
		},
		"service_account_key_valid": {
			config: &Config{
				ClientID:               "testClient",
				ClientSecret:           "testSecret",
				RedirectURI:            ts.URL + "/callback",
				Scopes:                 []string{"openid", "groups"},
				DomainToAdminEmail:     map[string]string{"bar.com": "foo@bar.com"},
				ServiceAccountFilePath: serviceAccountFilePath,
			},
			expectedErr: "",
		},
		"adc": {
			config: &Config{
				ClientID:           "testClient",
				ClientSecret:       "testSecret",
				RedirectURI:        ts.URL + "/callback",
				Scopes:             []string{"openid", "groups"},
				DomainToAdminEmail: map[string]string{"*": "foo@bar.com"},
			},
			adc:         serviceAccountFilePath,
			expectedErr: "",
		},
		"adc_priority": {
			config: &Config{
				ClientID:               "testClient",
				ClientSecret:           "testSecret",
				RedirectURI:            ts.URL + "/callback",
				Scopes:                 []string{"openid", "groups"},
				DomainToAdminEmail:     map[string]string{"*": "foo@bar.com"},
				ServiceAccountFilePath: serviceAccountFilePath,
			},
			adc:         "/dev/null",
			expectedErr: "",
		},
	} {
		reference := reference
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", reference.adc)
			conn, err := newConnector(reference.config)

			if reference.expectedErr == "" {
				assert.Nil(err)
				assert.NotNil(conn)
			} else {
				assert.ErrorContains(err, reference.expectedErr)
			}
		})
	}
}

func TestGetGroups(t *testing.T) {
	ts := testSetup()
	defer ts.Close()

	serviceAccountFilePath, err := tempServiceAccountKey()
	assert.Nil(t, err)

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", serviceAccountFilePath)
	conn, err := newConnector(&Config{
		ClientID:           "testClient",
		ClientSecret:       "testSecret",
		RedirectURI:        ts.URL + "/callback",
		Scopes:             []string{"openid", "groups"},
		DomainToAdminEmail: map[string]string{"*": "admin@dexidp.com"},
	})
	assert.Nil(t, err)

	conn.adminSrv[wildcardDomainToAdminEmail], err = admin.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(ts.URL))
	assert.Nil(t, err)
	type testCase struct {
		userKey                        string
		fetchTransitiveGroupMembership bool
		shouldErr                      bool
		expectedGroups                 []string
	}

	for name, testCase := range map[string]testCase{
		"user1_non_transitive_lookup": {
			userKey:                        "user_1@dexidp.com",
			fetchTransitiveGroupMembership: false,
			shouldErr:                      false,
			expectedGroups:                 []string{"groups_1@dexidp.com", "groups_2@dexidp.com"},
		},
		"user1_transitive_lookup": {
			userKey:                        "user_1@dexidp.com",
			fetchTransitiveGroupMembership: true,
			shouldErr:                      false,
			expectedGroups:                 []string{"groups_0@dexidp.com", "groups_1@dexidp.com", "groups_2@dexidp.com"},
		},
		"user2_non_transitive_lookup": {
			userKey:                        "user_2@dexidp.com",
			fetchTransitiveGroupMembership: false,
			shouldErr:                      false,
			expectedGroups:                 []string{"groups_1@dexidp.com"},
		},
		"user2_transitive_lookup": {
			userKey:                        "user_2@dexidp.com",
			fetchTransitiveGroupMembership: true,
			shouldErr:                      false,
			expectedGroups:                 []string{"groups_0@dexidp.com", "groups_1@dexidp.com"},
		},
	} {
		testCase := testCase
		callCounter = map[string]int{}
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			lookup := make(map[string]struct{})

			groups, err := conn.getGroups(testCase.userKey, testCase.fetchTransitiveGroupMembership, lookup)
			if testCase.shouldErr {
				assert.NotNil(err)
			} else {
				assert.Nil(err)
			}
			assert.ElementsMatch(testCase.expectedGroups, groups)
			t.Logf("[%s] Amount of API calls per userKey: %+v\n", t.Name(), callCounter)
		})
	}
}

func TestDomainToAdminEmailConfig(t *testing.T) {
	ts := testSetup()
	defer ts.Close()

	serviceAccountFilePath, err := tempServiceAccountKey()
	assert.Nil(t, err)

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", serviceAccountFilePath)
	conn, err := newConnector(&Config{
		ClientID:           "testClient",
		ClientSecret:       "testSecret",
		RedirectURI:        ts.URL + "/callback",
		Scopes:             []string{"openid", "groups"},
		DomainToAdminEmail: map[string]string{"dexidp.com": "admin@dexidp.com"},
	})
	assert.Nil(t, err)

	conn.adminSrv["dexidp.com"], err = admin.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(ts.URL))
	assert.Nil(t, err)
	type testCase struct {
		userKey     string
		expectedErr string
	}

	for name, testCase := range map[string]testCase{
		"correct_user_request": {
			userKey:     "user_1@dexidp.com",
			expectedErr: "",
		},
		"wrong_user_request": {
			userKey:     "user_1@foo.bar",
			expectedErr: "unable to find super admin email",
		},
		"wrong_connector_response": {
			userKey:     "user_1_foo.bar",
			expectedErr: "unable to find super admin email",
		},
	} {
		testCase := testCase
		callCounter = map[string]int{}
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			lookup := make(map[string]struct{})

			_, err := conn.getGroups(testCase.userKey, true, lookup)
			if testCase.expectedErr != "" {
				assert.ErrorContains(err, testCase.expectedErr)
			} else {
				assert.Nil(err)
			}
			t.Logf("[%s] Amount of API calls per userKey: %+v\n", t.Name(), callCounter)
		})
	}
}

var gceMetadataFlags = map[string]bool{
	"failOnEmailRequest": false,
}

func mockGCEMetadataServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/computeMetadata/v1/instance/service-accounts/default/email", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		if gceMetadataFlags["failOnEmailRequest"] {
			w.WriteHeader(http.StatusBadRequest)
		}
		json.NewEncoder(w).Encode("my-service-account@example-project.iam.gserviceaccount.com")
	})
	mux.HandleFunc("/computeMetadata/v1/instance/service-accounts/default/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			AccessToken  string `json:"access_token"`
			ExpiresInSec int    `json:"expires_in"`
			TokenType    string `json:"token_type"`
		}{
			AccessToken:  "my-example.token",
			ExpiresInSec: 3600,
			TokenType:    "Bearer",
		})
	})

	return httptest.NewServer(mux)
}

func TestGCEWorkloadIdentity(t *testing.T) {
	ts := testSetup()
	defer ts.Close()

	metadataServer := mockGCEMetadataServer()
	defer metadataServer.Close()
	metadataServerHost := strings.Replace(metadataServer.URL, "http://", "", 1)

	os.Setenv("GCE_METADATA_HOST", metadataServerHost)
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
	os.Setenv("HOME", "/tmp")

	gceMetadataFlags["failOnEmailRequest"] = true
	_, err := newConnector(&Config{
		ClientID:           "testClient",
		ClientSecret:       "testSecret",
		RedirectURI:        ts.URL + "/callback",
		Scopes:             []string{"openid", "groups"},
		DomainToAdminEmail: map[string]string{"dexidp.com": "admin@dexidp.com"},
	})
	assert.Error(t, err)

	gceMetadataFlags["failOnEmailRequest"] = false
	conn, err := newConnector(&Config{
		ClientID:           "testClient",
		ClientSecret:       "testSecret",
		RedirectURI:        ts.URL + "/callback",
		Scopes:             []string{"openid", "groups"},
		DomainToAdminEmail: map[string]string{"dexidp.com": "admin@dexidp.com"},
	})
	assert.Nil(t, err)

	conn.adminSrv["dexidp.com"], err = admin.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(ts.URL))
	assert.Nil(t, err)
	type testCase struct {
		userKey     string
		expectedErr string
	}

	for name, testCase := range map[string]testCase{
		"correct_user_request": {
			userKey:     "user_1@dexidp.com",
			expectedErr: "",
		},
		"wrong_user_request": {
			userKey:     "user_1@foo.bar",
			expectedErr: "unable to find super admin email",
		},
		"wrong_connector_response": {
			userKey:     "user_1_foo.bar",
			expectedErr: "unable to find super admin email",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			lookup := make(map[string]struct{})

			_, err := conn.getGroups(testCase.userKey, true, lookup)
			if testCase.expectedErr != "" {
				assert.ErrorContains(err, testCase.expectedErr)
			} else {
				assert.Nil(err)
			}
		})
	}
}

func TestPromptTypeConfig(t *testing.T) {
	promptTypeLogin := "login"
	cases := []struct {
		name                    string
		promptType              *string
		expectedPromptTypeValue string
	}{
		{
			name:                    "prompt type is nil",
			promptType:              nil,
			expectedPromptTypeValue: "consent",
		},
		{
			name:                    "prompt type is empty",
			promptType:              new(string),
			expectedPromptTypeValue: "",
		},
		{
			name:                    "prompt type is set",
			promptType:              &promptTypeLogin,
			expectedPromptTypeValue: "login",
		},
	}

	ts := testSetup()
	defer ts.Close()

	serviceAccountFilePath, err := tempServiceAccountKey()
	assert.Nil(t, err)

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", serviceAccountFilePath)

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			conn, err := newConnector(&Config{
				ClientID:           "testClient",
				ClientSecret:       "testSecret",
				RedirectURI:        ts.URL + "/callback",
				Scopes:             []string{"openid", "groups", "offline_access"},
				DomainToAdminEmail: map[string]string{"dexidp.com": "admin@dexidp.com"},
				PromptType:         test.promptType,
			})

			assert.Nil(t, err)
			assert.Equal(t, test.expectedPromptTypeValue, conn.promptType)

			loginURL, _, err := conn.LoginURL(connector.Scopes{OfflineAccess: true}, ts.URL+"/callback", "state")
			assert.Nil(t, err)

			urlp, err := url.Parse(loginURL)
			assert.Nil(t, err)

			assert.Equal(t, test.expectedPromptTypeValue, urlp.Query().Get("prompt"))
		})
	}
}

// TestRefreshUpstreamRejection verifies that a refresh Google refuses, as it
// does for a suspended or deleted user, keeps Google's OAuth2 error, so the
// server can tell a rejected refresh token apart from other failures.
func TestRefreshUpstreamRejection(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "Bad Request",
		})
	}))
	defer ts.Close()

	conn := &googleConnector{
		oauth2Config: &oauth2.Config{
			ClientID:     "testClient",
			ClientSecret: "testSecret",
			Endpoint:     oauth2.Endpoint{TokenURL: ts.URL + "/token"},
		},
		logger: slog.New(slog.DiscardHandler),
	}

	_, err := conn.Refresh(t.Context(), connector.Scopes{OfflineAccess: true}, connector.Identity{
		ConnectorData: []byte("refresh-token"),
	})

	var retrieveErr *oauth2.RetrieveError
	require.ErrorAs(t, err, &retrieveErr)
	require.Equal(t, "invalid_grant", retrieveErr.ErrorCode)
}

// tokenExchangeSetup returns a googleConnector whose ID token verification and
// userinfo calls go to local test doubles, a function to sign ID tokens with the
// test key, and the test server.
func tokenExchangeSetup(t *testing.T) (*googleConnector, func(claims map[string]any) string, *httptest.Server) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, nil)
	require.NoError(t, err)

	sign := func(claims map[string]any) string {
		payload, err := json.Marshal(claims)
		require.NoError(t, err)
		jws, err := signer.Sign(payload)
		require.NoError(t, err)
		raw, err := jws.CompactSerialize()
		require.NoError(t, err)
		return raw
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer valid-access-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"sub":            "userinfo-subject",
			"name":           "User One",
			"email":          "user_1@dexidp.com",
			"email_verified": true,
			"hd":             "dexidp.com",
		})
	})
	mux.HandleFunc("/admin/directory/v1/groups/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(admin.Groups{Groups: testGroups[r.URL.Query().Get("userKey")]})
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	keySet := &oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{key.Public()}}
	provider := (&oidc.ProviderConfig{
		IssuerURL:   issuerURL,
		UserInfoURL: ts.URL + "/userinfo",
	}).NewProvider(t.Context())

	conn := &googleConnector{
		provider:         provider,
		verifier:         oidc.NewVerifier(issuerURL, keySet, &oidc.Config{ClientID: "testClient"}),
		exchangeVerifier: oidc.NewVerifier(issuerURL, keySet, &oidc.Config{SkipClientIDCheck: true}),
		logger:           slog.New(slog.DiscardHandler),
		adminSrv:         map[string]*admin.Service{},
	}
	return conn, sign, ts
}

func TestTokenIdentity(t *testing.T) {
	const (
		idTokenType     = "urn:ietf:params:oauth:token-type:id_token"
		accessTokenType = "urn:ietf:params:oauth:token-type:access_token"
	)

	idTokenClaims := func(overrides map[string]any) map[string]any {
		claims := map[string]any{
			"iss":            issuerURL,
			"sub":            "id-token-subject",
			"aud":            "some-other-audience",
			"exp":            time.Now().Add(time.Hour).Unix(),
			"name":           "User One",
			"email":          "user_1@dexidp.com",
			"email_verified": true,
			"hd":             "dexidp.com",
		}
		for k, v := range overrides {
			claims[k] = v
		}
		return claims
	}

	tests := []struct {
		name          string
		hostedDomains []string
		groups        []string
		withGroups    bool
		tokenType     string
		token         func(sign func(map[string]any) string) string
		wantIdentity  connector.Identity
		wantErr       string
	}{
		{
			name:      "id token",
			tokenType: idTokenType,
			token:     func(sign func(map[string]any) string) string { return sign(idTokenClaims(nil)) },
			wantIdentity: connector.Identity{
				UserID:        "id-token-subject",
				Username:      "User One",
				Email:         "user_1@dexidp.com",
				EmailVerified: true,
				ConnectorData: []byte{},
			},
		},
		{
			name:       "id token with groups",
			tokenType:  idTokenType,
			withGroups: true,
			groups:     []string{"groups_1@dexidp.com"},
			token:      func(sign func(map[string]any) string) string { return sign(idTokenClaims(nil)) },
			wantIdentity: connector.Identity{
				UserID:        "id-token-subject",
				Username:      "User One",
				Email:         "user_1@dexidp.com",
				EmailVerified: true,
				ConnectorData: []byte{},
				Groups:        []string{"groups_1@dexidp.com"},
			},
		},
		{
			name:       "id token user not in required groups",
			tokenType:  idTokenType,
			withGroups: true,
			groups:     []string{"groups_0@dexidp.com"},
			token:      func(sign func(map[string]any) string) string { return sign(idTokenClaims(nil)) },
			wantErr:    "not in any of the required groups",
		},
		{
			name:      "id token from another issuer",
			tokenType: idTokenType,
			token: func(sign func(map[string]any) string) string {
				return sign(idTokenClaims(map[string]any{"iss": "https://example.com"}))
			},
			wantErr: "failed to verify token",
		},
		{
			name:      "expired id token",
			tokenType: idTokenType,
			token: func(sign func(map[string]any) string) string {
				return sign(idTokenClaims(map[string]any{"exp": time.Now().Add(-time.Hour).Unix()}))
			},
			wantErr: "failed to verify token",
		},
		{
			name:          "id token from unexpected hosted domain",
			hostedDomains: []string{"example.com"},
			tokenType:     idTokenType,
			token:         func(sign func(map[string]any) string) string { return sign(idTokenClaims(nil)) },
			wantErr:       "unexpected hd claim",
		},
		{
			name:          "access token",
			hostedDomains: []string{"dexidp.com"},
			tokenType:     accessTokenType,
			token:         func(func(map[string]any) string) string { return "valid-access-token" },
			wantIdentity: connector.Identity{
				UserID:        "userinfo-subject",
				Username:      "User One",
				Email:         "user_1@dexidp.com",
				EmailVerified: true,
				ConnectorData: []byte{},
			},
		},
		{
			name:      "access token rejected by userinfo",
			tokenType: accessTokenType,
			token:     func(func(map[string]any) string) string { return "bad-access-token" },
			wantErr:   "error loading userinfo",
		},
		{
			name:      "unknown token type",
			tokenType: "urn:ietf:params:oauth:token-type:saml2",
			token:     func(func(map[string]any) string) string { return "token" },
			wantErr:   "unknown token type",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn, sign, ts := tokenExchangeSetup(t)
			conn.hostedDomains = tc.hostedDomains
			conn.groups = tc.groups
			if tc.withGroups {
				srv, err := admin.NewService(t.Context(), option.WithoutAuthentication(), option.WithEndpoint(ts.URL))
				require.NoError(t, err)
				conn.adminSrv[wildcardDomainToAdminEmail] = srv
			}

			identity, err := conn.TokenIdentity(t.Context(), tc.tokenType, tc.token(sign))
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantIdentity, identity)
		})
	}
}

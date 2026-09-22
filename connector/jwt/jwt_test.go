package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/dexidp/dex/connector"
)

var logger = slog.New(slog.DiscardHandler)

type testProvider struct {
	server   *httptest.Server
	key      *jose.JSONWebKey
	otherKey *jose.JSONWebKey
}

func (p *testProvider) issuer() string { return p.server.URL }

func (p *testProvider) jwksURL() string { return p.server.URL + "/keys" }

// newTestProvider starts a provider exposing a JWKS and a discovery document.
func newTestProvider(t *testing.T) *testProvider {
	t.Helper()

	key := newSigningKey(t, "keyID")
	otherKey := newSigningKey(t, "otherKeyID")

	mux := http.NewServeMux()
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		public := key.Public()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&jose.JSONWebKeySet{Keys: []jose.JSONWebKey{public}})
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&map[string]string{
			"issuer":                 server.URL,
			"authorization_endpoint": server.URL + "/authorize",
			"token_endpoint":         server.URL + "/token",
			"jwks_uri":               server.URL + "/keys",
		})
	})

	return &testProvider{server: server, key: key, otherKey: otherKey}
}

func newSigningKey(t *testing.T, kid string) *jose.JSONWebKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}
	return &jose.JSONWebKey{Key: key, KeyID: kid, Algorithm: string(jose.RS256), Use: "sig"}
}

func signToken(t *testing.T, key *jose.JSONWebKey, claims map[string]interface{}) string {
	t.Helper()

	signer, err := jose.NewSigner(jose.SigningKey{Key: key, Algorithm: jose.RS256}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("failed to marshal claims: %v", err)
	}

	signature, err := signer.Sign(payload)
	if err != nil {
		t.Fatalf("failed to sign claims: %v", err)
	}

	raw, err := signature.CompactSerialize()
	if err != nil {
		t.Fatalf("failed to serialize token: %v", err)
	}
	return raw
}

func newConnector(t *testing.T, config Config) *jwtConnector {
	t.Helper()

	conn, err := config.Open("jwt", logger)
	if err != nil {
		t.Fatalf("unable to open connector: %v", err)
	}
	t.Cleanup(func() { conn.(*jwtConnector).Close() })

	jwtConn, ok := conn.(*jwtConnector)
	if !ok {
		t.Fatalf("expected *jwtConnector, got %T", conn)
	}
	return jwtConn
}

func requestWithToken(t *testing.T, header, value string) *http.Request {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, "/callback/jwt", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	if value != "" {
		req.Header.Set(header, value)
	}
	return req
}

func baseClaims(issuer string) map[string]interface{} {
	return map[string]interface{}{
		"iss":                issuer,
		"aud":                "argocd",
		"exp":                time.Now().Add(time.Hour).Unix(),
		"iat":                time.Now().Unix(),
		"sub":                "user-id",
		"name":               "Test User",
		"preferred_username": "testuser",
		"email":              "testuser@example.com",
		"email_verified":     true,
		"groups":             []string{"platform", "developers"},
	}
}

func TestOpenRequiresIssuer(t *testing.T) {
	provider := newTestProvider(t)

	_, err := (&Config{JWKSURL: provider.jwksURL()}).Open("jwt", logger)
	if err == nil {
		t.Fatal("expected an error when issuer is empty")
	}
}

func TestOpenValidatesRoleMapping(t *testing.T) {
	provider := newTestProvider(t)

	tests := map[string]RoleMapping{
		"missing claim":         {Value: "platform", Groups: []string{"argocd-admins"}},
		"no matcher":            {Claim: "department", Groups: []string{"argocd-admins"}},
		"conflicting matchers":  {Claim: "department", Value: "platform", ValueRegex: ".*", Groups: []string{"argocd-admins"}},
		"invalid value regex":   {Claim: "department", ValueRegex: "([a-z", Groups: []string{"argocd-admins"}},
		"valid regex is parsed": {Claim: "department", ValueRegex: "^platform$", Groups: []string{"argocd-admins"}},
	}

	for name, mapping := range tests {
		t.Run(name, func(t *testing.T) {
			config := Config{
				Issuer:      provider.issuer(),
				JWKSURL:     provider.jwksURL(),
				RoleMapping: []RoleMapping{mapping},
			}

			conn, err := config.Open("jwt", logger)
			if name == "valid regex is parsed" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				conn.(*jwtConnector).Close()
				return
			}
			if err == nil {
				conn.(*jwtConnector).Close()
				t.Fatal("expected a config validation error")
			}
		})
	}
}

func TestOpenValidatesGroupsFilter(t *testing.T) {
	provider := newTestProvider(t)

	config := Config{Issuer: provider.issuer(), JWKSURL: provider.jwksURL()}
	config.ClaimMutations.FilterGroupClaims.GroupsFilter = "([a-z"

	if _, err := config.Open("jwt", logger); err == nil {
		t.Fatal("expected an error for an invalid groupsFilter regex")
	}
}

func TestOpenDiscoversJWKSFromIssuer(t *testing.T) {
	provider := newTestProvider(t)

	conn := newConnector(t, Config{Issuer: provider.issuer()})

	req := requestWithToken(t, "Authorization", "Bearer "+signToken(t, provider.key, baseClaims(provider.issuer())))
	ident, err := conn.HandleCallback(connector.Scopes{Groups: true}, nil, req)
	if err != nil {
		t.Fatalf("expected a valid identity, got %v", err)
	}
	if ident.UserID != "user-id" {
		t.Fatalf("expected user ID %q, got %q", "user-id", ident.UserID)
	}
}

func TestOpenFailsWhenIssuerUnreachable(t *testing.T) {
	provider := newTestProvider(t)
	provider.server.Close()

	if _, err := (&Config{Issuer: provider.issuer()}).Open("jwt", logger); err == nil {
		t.Fatal("expected an error when discovery is unreachable")
	}
}

func TestLoginURL(t *testing.T) {
	provider := newTestProvider(t)
	conn := newConnector(t, Config{Issuer: provider.issuer(), JWKSURL: provider.jwksURL()})

	loginURL, data, err := conn.LoginURL(connector.Scopes{}, "https://dex.example.com/callback", "some-state")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != nil {
		t.Fatalf("expected no connector data, got %v", data)
	}
	if want := "https://dex.example.com/callback/jwt?state=some-state"; loginURL != want {
		t.Fatalf("expected %q, got %q", want, loginURL)
	}
}

func TestHandleCallbackIdentity(t *testing.T) {
	provider := newTestProvider(t)
	conn := newConnector(t, Config{
		Issuer:    provider.issuer(),
		JWKSURL:   provider.jwksURL(),
		Audiences: []string{"argocd"},
	})

	req := requestWithToken(t, "Authorization", "Bearer "+signToken(t, provider.key, baseClaims(provider.issuer())))
	ident, err := conn.HandleCallback(connector.Scopes{Groups: true}, nil, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := connector.Identity{
		UserID:            "user-id",
		Username:          "Test User",
		PreferredUsername: "testuser",
		Email:             "testuser@example.com",
		EmailVerified:     true,
		Groups:            []string{"platform", "developers"},
	}
	assertIdentity(t, expected, ident)
}

func TestHandleCallbackTokenExtraction(t *testing.T) {
	provider := newTestProvider(t)

	tests := []struct {
		name      string
		header    string
		setHeader string
		value     func(token string) string
		wantErr   bool
	}{
		{name: "bearer prefix", header: "Authorization", setHeader: "Authorization", value: func(tok string) string { return "Bearer " + tok }},
		{name: "lowercase bearer prefix", header: "Authorization", setHeader: "Authorization", value: func(tok string) string { return "bearer " + tok }},
		{name: "no prefix", header: "Authorization", setHeader: "Authorization", value: func(tok string) string { return tok }},
		{name: "surrounding whitespace", header: "Authorization", setHeader: "Authorization", value: func(tok string) string { return "  Bearer  " + tok + " " }},
		{name: "custom header", header: "X-Access-Token", setHeader: "X-Access-Token", value: func(tok string) string { return tok }},
		{name: "token on the wrong header", header: "X-Access-Token", setHeader: "Authorization", value: func(tok string) string { return tok }, wantErr: true},
		{name: "empty header", header: "Authorization", setHeader: "Authorization", value: func(string) string { return "" }, wantErr: true},
		{name: "bearer prefix without token", header: "Authorization", setHeader: "Authorization", value: func(string) string { return "Bearer " }, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn := newConnector(t, Config{Issuer: provider.issuer(), JWKSURL: provider.jwksURL(), Header: tc.header})

			token := signToken(t, provider.key, baseClaims(provider.issuer()))
			req := requestWithToken(t, tc.setHeader, tc.value(token))

			_, err := conn.HandleCallback(connector.Scopes{Groups: true}, nil, req)
			if tc.wantErr && err == nil {
				t.Fatal("expected an error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestHandleCallbackMissingHeader(t *testing.T) {
	provider := newTestProvider(t)
	conn := newConnector(t, Config{Issuer: provider.issuer(), JWKSURL: provider.jwksURL()})

	req := requestWithToken(t, "Authorization", "")
	if _, err := conn.HandleCallback(connector.Scopes{}, nil, req); err == nil {
		t.Fatal("expected an error when the header is absent")
	}
}

func TestHandleCallbackVerification(t *testing.T) {
	provider := newTestProvider(t)

	tests := []struct {
		name   string
		config func(c *Config)
		claims func(c map[string]interface{})
		key    func(p *testProvider) *jose.JSONWebKey
		raw    string
	}{
		{
			name:   "expired token",
			claims: func(c map[string]interface{}) { c["exp"] = time.Now().Add(-time.Hour).Unix() },
		},
		{
			name:   "not yet valid token",
			claims: func(c map[string]interface{}) { c["nbf"] = time.Now().Add(time.Hour).Unix() },
		},
		{
			name:   "wrong issuer",
			claims: func(c map[string]interface{}) { c["iss"] = "https://evil.example.com" },
		},
		{
			name:   "unexpected audience",
			config: func(c *Config) { c.Audiences = []string{"argocd"} },
			claims: func(c map[string]interface{}) { c["aud"] = "other-service" },
		},
		{
			name: "unknown signing key",
			key:  func(p *testProvider) *jose.JSONWebKey { return p.otherKey },
		},
		{
			name:   "unsupported signing algorithm",
			config: func(c *Config) { c.SupportedSigningAlgs = []string{"ES256"} },
		},
		{
			name: "malformed token",
			raw:  "not-a-jwt",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := Config{Issuer: provider.issuer(), JWKSURL: provider.jwksURL()}
			if tc.config != nil {
				tc.config(&config)
			}
			conn := newConnector(t, config)

			raw := tc.raw
			if raw == "" {
				claims := baseClaims(provider.issuer())
				if tc.claims != nil {
					tc.claims(claims)
				}
				key := provider.key
				if tc.key != nil {
					key = tc.key(provider)
				}
				raw = signToken(t, key, claims)
			}

			req := requestWithToken(t, "Authorization", "Bearer "+raw)
			if _, err := conn.HandleCallback(connector.Scopes{Groups: true}, nil, req); err == nil {
				t.Fatal("expected the token to be rejected")
			}
		})
	}
}

// The issuer is enforced even when keys are loaded from an explicit JWKS URL,
// since jwksURL only overrides key retrieval.
func TestIssuerEnforcedWithExplicitJWKSURL(t *testing.T) {
	provider := newTestProvider(t)
	conn := newConnector(t, Config{Issuer: "https://expected.example.com", JWKSURL: provider.jwksURL()})

	claims := baseClaims("https://other.example.com")
	req := requestWithToken(t, "Authorization", "Bearer "+signToken(t, provider.key, claims))

	if _, err := conn.HandleCallback(connector.Scopes{}, nil, req); err == nil {
		t.Fatal("expected a token from an unexpected issuer to be rejected")
	}
}

func TestHandleCallbackMultipleAudiences(t *testing.T) {
	provider := newTestProvider(t)
	conn := newConnector(t, Config{
		Issuer:    provider.issuer(),
		JWKSURL:   provider.jwksURL(),
		Audiences: []string{"argocd", "dex"},
	})

	claims := baseClaims(provider.issuer())
	claims["aud"] = []string{"other-service", "dex"}

	req := requestWithToken(t, "Authorization", "Bearer "+signToken(t, provider.key, claims))
	if _, err := conn.HandleCallback(connector.Scopes{}, nil, req); err != nil {
		t.Fatalf("expected the token to be accepted, got %v", err)
	}
}

func TestHandleCallbackClaimMapping(t *testing.T) {
	provider := newTestProvider(t)

	config := Config{
		Issuer:      provider.issuer(),
		JWKSURL:     provider.jwksURL(),
		UserIDKey:   "user_id",
		UserNameKey: "full_name",
	}
	config.ClaimMapping.PreferredUsernameKey = "login"
	config.ClaimMapping.EmailKey = "mail"
	config.ClaimMapping.GroupsKey = "roles"
	conn := newConnector(t, config)

	claims := baseClaims(provider.issuer())
	claims["user_id"] = "custom-id"
	claims["full_name"] = "Custom Name"
	claims["login"] = "custom-login"
	claims["mail"] = "custom@example.com"
	claims["roles"] = []string{"custom-group"}

	req := requestWithToken(t, "Authorization", "Bearer "+signToken(t, provider.key, claims))
	ident, err := conn.HandleCallback(connector.Scopes{Groups: true}, nil, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := connector.Identity{
		UserID:            "custom-id",
		Username:          "Custom Name",
		PreferredUsername: "custom-login",
		Email:             "custom@example.com",
		EmailVerified:     true,
		Groups:            []string{"custom-group"},
	}
	assertIdentity(t, expected, ident)
}

func TestHandleCallbackMissingClaims(t *testing.T) {
	provider := newTestProvider(t)
	conn := newConnector(t, Config{Issuer: provider.issuer(), JWKSURL: provider.jwksURL()})

	for _, claim := range []string{"sub", "name"} {
		t.Run("missing "+claim, func(t *testing.T) {
			claims := baseClaims(provider.issuer())
			delete(claims, claim)

			req := requestWithToken(t, "Authorization", "Bearer "+signToken(t, provider.key, claims))
			if _, err := conn.HandleCallback(connector.Scopes{}, nil, req); err == nil {
				t.Fatalf("expected an error when the %q claim is missing", claim)
			}
		})
	}
}

func TestHandleCallbackEmailVerified(t *testing.T) {
	provider := newTestProvider(t)

	t.Run("missing email_verified is rejected", func(t *testing.T) {
		conn := newConnector(t, Config{Issuer: provider.issuer(), JWKSURL: provider.jwksURL()})

		claims := baseClaims(provider.issuer())
		delete(claims, "email_verified")

		req := requestWithToken(t, "Authorization", "Bearer "+signToken(t, provider.key, claims))
		if _, err := conn.HandleCallback(connector.Scopes{}, nil, req); err == nil {
			t.Fatal("expected an error when email_verified is missing")
		}
	})

	t.Run("email_verified false can be overridden", func(t *testing.T) {
		conn := newConnector(t, Config{Issuer: provider.issuer(), JWKSURL: provider.jwksURL(), InsecureSkipEmailVerified: true})

		claims := baseClaims(provider.issuer())
		claims["email_verified"] = false

		req := requestWithToken(t, "Authorization", "Bearer "+signToken(t, provider.key, claims))
		ident, err := conn.HandleCallback(connector.Scopes{}, nil, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ident.EmailVerified {
			t.Fatal("expected email to be reported as verified")
		}
	})

	t.Run("token without an email needs no verification", func(t *testing.T) {
		conn := newConnector(t, Config{Issuer: provider.issuer(), JWKSURL: provider.jwksURL()})

		claims := baseClaims(provider.issuer())
		delete(claims, "email_verified")
		delete(claims, "email")

		req := requestWithToken(t, "Authorization", "Bearer "+signToken(t, provider.key, claims))
		ident, err := conn.HandleCallback(connector.Scopes{}, nil, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ident.EmailVerified {
			t.Fatal("expected email_verified to be false")
		}
	})
}

func TestHandleCallbackGroups(t *testing.T) {
	provider := newTestProvider(t)

	tests := []struct {
		name    string
		config  func(c *Config)
		claims  func(c map[string]interface{})
		groups  []string
		wantErr bool
	}{
		{
			name:   "groups claim as a single string",
			claims: func(c map[string]interface{}) { c["groups"] = "platform" },
			groups: []string{"platform"},
		},
		{
			name: "groups claim as objects",
			claims: func(c map[string]interface{}) {
				c["groups"] = []interface{}{map[string]interface{}{"name": "platform"}}
			},
			groups: []string{"platform"},
		},
		{
			name:    "malformed groups claim",
			claims:  func(c map[string]interface{}) { c["groups"] = []interface{}{42} },
			wantErr: true,
		},
		{
			name:   "no groups claim",
			claims: func(c map[string]interface{}) { delete(c, "groups") },
			groups: nil,
		},
		{
			name:   "group filter",
			config: func(c *Config) { c.ClaimMutations.FilterGroupClaims.GroupsFilter = "^dev" },
			groups: []string{"developers"},
		},
		{
			name: "group prefix and suffix",
			config: func(c *Config) {
				c.ClaimMutations.ModifyGroupNames.Prefix = "dex:"
				c.ClaimMutations.ModifyGroupNames.Suffix = ":upstream"
			},
			groups: []string{"dex:platform:upstream", "dex:developers:upstream"},
		},
		{
			name:   "allowed groups filters the identity",
			config: func(c *Config) { c.AllowedGroups = []string{"developers"} },
			groups: []string{"developers"},
		},
		{
			name:    "allowed groups rejects the login",
			config:  func(c *Config) { c.AllowedGroups = []string{"nobody"} },
			wantErr: true,
		},
		{
			name: "new group from claims",
			config: func(c *Config) {
				c.ClaimMutations.NewGroupFromClaims = []NewGroupFromClaims{{
					Claims:    []string{"organization", "team"},
					Delimiter: ":",
					Prefix:    "org",
				}}
			},
			claims: func(c map[string]interface{}) {
				c["organization"] = "acme"
				c["team"] = "platform"
			},
			groups: []string{"platform", "developers", "org:acme:platform"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := Config{Issuer: provider.issuer(), JWKSURL: provider.jwksURL()}
			if tc.config != nil {
				tc.config(&config)
			}
			conn := newConnector(t, config)

			claims := baseClaims(provider.issuer())
			if tc.claims != nil {
				tc.claims(claims)
			}

			req := requestWithToken(t, "Authorization", "Bearer "+signToken(t, provider.key, claims))
			ident, err := conn.HandleCallback(connector.Scopes{Groups: true}, nil, req)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertGroups(t, tc.groups, ident.Groups)
		})
	}
}

func TestHandleCallbackAllowedGroupsError(t *testing.T) {
	provider := newTestProvider(t)
	conn := newConnector(t, Config{
		Issuer:        provider.issuer(),
		JWKSURL:       provider.jwksURL(),
		AllowedGroups: []string{"nobody"},
	})

	req := requestWithToken(t, "Authorization", "Bearer "+signToken(t, provider.key, baseClaims(provider.issuer())))
	_, err := conn.HandleCallback(connector.Scopes{Groups: true}, nil, req)

	var groupsErr *connector.UserNotInRequiredGroupsError
	if !errors.As(err, &groupsErr) {
		t.Fatalf("expected a UserNotInRequiredGroupsError, got %v", err)
	}
	if groupsErr.UserID != "user-id" {
		t.Fatalf("expected user ID %q, got %q", "user-id", groupsErr.UserID)
	}
}

func TestHandleCallbackRoleMapping(t *testing.T) {
	provider := newTestProvider(t)

	tests := []struct {
		name     string
		mappings []RoleMapping
		config   func(c *Config)
		claims   func(c map[string]interface{})
		groups   []string
	}{
		{
			name:     "exact value match",
			mappings: []RoleMapping{{Claim: "department", Value: "platform", Groups: []string{"argocd-admins"}}},
			claims:   func(c map[string]interface{}) { c["department"] = "platform" },
			groups:   []string{"platform", "developers", "argocd-admins"},
		},
		{
			name:     "exact value mismatch",
			mappings: []RoleMapping{{Claim: "department", Value: "platform", Groups: []string{"argocd-admins"}}},
			claims:   func(c map[string]interface{}) { c["department"] = "finance" },
			groups:   []string{"platform", "developers"},
		},
		{
			name:     "regex match on a multi valued claim",
			mappings: []RoleMapping{{Claim: "roles", ValueRegex: "^sre-.*$", Groups: []string{"argocd-sre"}}},
			claims:   func(c map[string]interface{}) { c["roles"] = []string{"billing", "sre-oncall"} },
			groups:   []string{"platform", "developers", "argocd-sre"},
		},
		{
			name:     "missing claim adds nothing",
			mappings: []RoleMapping{{Claim: "department", Value: "platform", Groups: []string{"argocd-admins"}}},
			groups:   []string{"platform", "developers"},
		},
		{
			name: "duplicate groups are collapsed",
			mappings: []RoleMapping{
				{Claim: "department", Value: "platform", Groups: []string{"argocd-admins"}},
				{Claim: "roles", ValueRegex: ".*", Groups: []string{"argocd-admins", "developers"}},
			},
			claims: func(c map[string]interface{}) {
				c["department"] = "platform"
				c["roles"] = []string{"anything"}
			},
			groups: []string{"platform", "developers", "argocd-admins"},
		},
		{
			// Mapped role names must reach Argo CD's RBAC verbatim, so the
			// prefix/suffix of modifyGroupNames must not be applied to them.
			name:     "mapped roles are not prefixed or suffixed",
			mappings: []RoleMapping{{Claim: "department", Value: "platform", Groups: []string{"argocd-admins"}}},
			config: func(c *Config) {
				c.ClaimMutations.ModifyGroupNames.Prefix = "dex:"
				c.ClaimMutations.ModifyGroupNames.Suffix = ":upstream"
			},
			claims: func(c map[string]interface{}) { c["department"] = "platform" },
			groups: []string{"dex:platform:upstream", "dex:developers:upstream", "argocd-admins"},
		},
		{
			name:     "malformed claim is ignored",
			mappings: []RoleMapping{{Claim: "department", Value: "platform", Groups: []string{"argocd-admins"}}},
			claims:   func(c map[string]interface{}) { c["department"] = []interface{}{42} },
			groups:   []string{"platform", "developers"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := Config{Issuer: provider.issuer(), JWKSURL: provider.jwksURL(), RoleMapping: tc.mappings}
			if tc.config != nil {
				tc.config(&config)
			}
			conn := newConnector(t, config)

			claims := baseClaims(provider.issuer())
			if tc.claims != nil {
				tc.claims(claims)
			}

			req := requestWithToken(t, "Authorization", "Bearer "+signToken(t, provider.key, claims))
			ident, err := conn.HandleCallback(connector.Scopes{Groups: true}, nil, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertGroups(t, tc.groups, ident.Groups)
		})
	}
}

func assertIdentity(t *testing.T, expected, actual connector.Identity) {
	t.Helper()

	if expected.UserID != actual.UserID {
		t.Errorf("expected user ID %q, got %q", expected.UserID, actual.UserID)
	}
	if expected.Username != actual.Username {
		t.Errorf("expected username %q, got %q", expected.Username, actual.Username)
	}
	if expected.PreferredUsername != actual.PreferredUsername {
		t.Errorf("expected preferred username %q, got %q", expected.PreferredUsername, actual.PreferredUsername)
	}
	if expected.Email != actual.Email {
		t.Errorf("expected email %q, got %q", expected.Email, actual.Email)
	}
	if expected.EmailVerified != actual.EmailVerified {
		t.Errorf("expected email verified %v, got %v", expected.EmailVerified, actual.EmailVerified)
	}
	assertGroups(t, expected.Groups, actual.Groups)
}

func assertGroups(t *testing.T, expected, actual []string) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Fatalf("expected groups %v, got %v", expected, actual)
	}
	for i := range expected {
		if expected[i] != actual[i] {
			t.Fatalf("expected groups %v, got %v", expected, actual)
		}
	}
}

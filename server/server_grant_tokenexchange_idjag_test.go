package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/discovery"
	"github.com/dexidp/dex/server/grants"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

const idJAGAudience = "https://chat.example/"

func newIDJAGTestServer(t *testing.T, policies map[string]grants.IDJAGPolicy, updateConfig func(c *Config)) (*httptest.Server, *Server) {
	t.Helper()
	ctx := t.Context()
	return newTestServer(t, func(c *Config) {
		require.NoError(t, c.Storage.CreateClient(ctx, storage.Client{ID: "client_1", Secret: "secret_1", TrustedPeers: []string{"other_client"}}))
		require.NoError(t, c.Storage.CreateClient(ctx, storage.Client{ID: "public_client", Public: true}))
		require.NoError(t, c.Storage.CreateClient(ctx, storage.Client{ID: "public_client_with_secret", Public: true, Secret: "secret_2"}))
		c.IDJAG = &grants.IDJAGConfig{Policies: policies}
		if updateConfig != nil {
			updateConfig(c)
		}
	})
}

// signSubjectToken mints an ID token the way dex does after a login through connID.
func signSubjectToken(t *testing.T, s *Server, clientID, connID string, emailVerified bool, scopes ...string) string {
	t.Helper()
	token, _, err := s.issuer.SignIDToken(t.Context(), tokens.Authorization{
		Client: storage.Client{ID: clientID},
		Claims: storage.Claims{
			UserID:        "0-385-28089-0",
			Email:         "kilgore@kilgore.trout",
			EmailVerified: emailVerified,
		},
		Scopes:      append([]string{"openid", "email"}, scopes...),
		ConnectorID: connID,
	}, "", "")
	require.NoError(t, err)
	return token
}

func idJAGRequest(subjectToken string) url.Values {
	vals := url.Values{}
	vals.Set("grant_type", oauth2.GrantTypeTokenExchange)
	vals.Set("requested_token_type", oauth2.TokenTypeIDJAG)
	vals.Set("subject_token_type", oauth2.TokenTypeID)
	vals.Set("subject_token", subjectToken)
	vals.Set("connector_id", "mock")
	vals.Set("audience", idJAGAudience)
	vals.Set("client_id", "client_1")
	vals.Set("client_secret", "secret_1")
	return vals
}

func postToken(t *testing.T, s *Server, httpServer *httptest.Server, vals url.Values) *httptest.ResponseRecorder {
	t.Helper()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, httpServer.URL+"/token", strings.NewReader(vals.Encode()))
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	s.ServeHTTP(rr, req)
	return rr
}

func TestTokenExchangeIDJAG(t *testing.T) {
	httpServer, s := newIDJAGTestServer(t, map[string]grants.IDJAGPolicy{
		"client_1": {AllowedAudiences: []string{idJAGAudience}, AllowedScopes: []string{"chat.read"}},
	}, nil)
	defer httpServer.Close()

	subjectToken := signSubjectToken(t, s, "client_1", "mock", true)
	vals := idJAGRequest(subjectToken)
	vals.Set("resource", "https://chat.example/api")
	vals.Set("scope", "chat.read")

	rr := postToken(t, s, httpServer, vals)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var res tokens.Response
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&res))
	require.Equal(t, oauth2.TokenTypeIDJAG, res.IssuedTokenType)
	require.Equal(t, "N_A", res.TokenType)
	require.Equal(t, 300, res.ExpiresIn)
	require.Empty(t, res.Scope, "scope is omitted when it equals the requested scope")

	jws, err := jose.ParseSignedCompact(res.AccessToken, []jose.SignatureAlgorithm{jose.RS256})
	require.NoError(t, err)
	require.Equal(t, "oauth-id-jag+jwt", jws.Signatures[0].Protected.ExtraHeaders[jose.HeaderType])
	payload, err := jws.Verify(&testKey.PublicKey)
	require.NoError(t, err)

	var claims tokens.IDJAGClaims
	require.NoError(t, json.Unmarshal(payload, &claims))
	require.Equal(t, httpServer.URL, claims.Issuer)
	require.Equal(t, jwtSubject(t, subjectToken), claims.Subject, "sub is the one in the subject token")
	require.Equal(t, idJAGAudience, claims.Audience)
	require.Equal(t, "client_1", claims.ClientID)
	require.Equal(t, "https://chat.example/api", claims.Resource)
	require.Equal(t, "chat.read", claims.Scope)
	require.Equal(t, "kilgore@kilgore.trout", claims.Email)
	require.NotEmpty(t, claims.JWTID)
	require.Equal(t, int64(300), claims.Expiry-claims.IssuedAt)
}

func TestTokenExchangeIDJAGScopeReduced(t *testing.T) {
	httpServer, s := newIDJAGTestServer(t, map[string]grants.IDJAGPolicy{
		"client_1": {AllowedAudiences: []string{idJAGAudience}, AllowedScopes: []string{"chat.read"}},
	}, nil)
	defer httpServer.Close()

	vals := idJAGRequest(signSubjectToken(t, s, "client_1", "mock", true))
	vals.Set("scope", "chat.read chat.write chat.read")

	rr := postToken(t, s, httpServer, vals)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var res tokens.Response
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&res))
	require.Equal(t, "chat.read", res.Scope, "scope is required when it differs from the requested scope")
	require.Equal(t, "chat.read", decodeIDJAG(t, res.AccessToken).Scope)
}

func TestTokenExchangeIDJAGES256(t *testing.T) {
	httpServer, s := newIDJAGTestServer(t, map[string]grants.IDJAGPolicy{
		"client_1": {AllowedAudiences: []string{idJAGAudience}},
	}, func(c *Config) {
		localConfig := signer.LocalConfig{KeysRotationPeriod: time.Hour.String(), Algorithm: jose.ES256}
		sig, err := localConfig.Open(t.Context(), c.Storage, time.Hour, time.Now, c.Logger)
		require.NoError(t, err)
		c.Signer = sig
	})
	defer httpServer.Close()

	rr := postToken(t, s, httpServer, idJAGRequest(signSubjectToken(t, s, "client_1", "mock", true)))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var res tokens.Response
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&res))
	jws, err := jose.ParseSignedCompact(res.AccessToken, []jose.SignatureAlgorithm{jose.ES256})
	require.NoError(t, err)
	require.Equal(t, "oauth-id-jag+jwt", jws.Signatures[0].Protected.ExtraHeaders[jose.HeaderType])
}

func TestTokenExchangeIDJAGMetrics(t *testing.T) {
	var registry *prometheus.Registry
	httpServer, s := newIDJAGTestServer(t, map[string]grants.IDJAGPolicy{
		"client_1": {AllowedAudiences: []string{idJAGAudience}, AllowedScopes: []string{"chat.read"}},
	}, func(c *Config) {
		registry = c.PrometheusRegistry
	})
	defer httpServer.Close()

	subjectToken := signSubjectToken(t, s, "client_1", "mock", true)
	issued := idJAGRequest(subjectToken)
	issued.Set("scope", "chat.read chat.write")
	unknownAudience := idJAGRequest(subjectToken)
	unknownAudience.Set("audience", "https://unknown.example/")
	for _, vals := range []url.Values{issued, unknownAudience, idJAGRequest("foobar")} {
		postToken(t, s, httpServer, vals)
	}

	require.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP dex_id_jag_policy_rejections_total Total number of ID-JAG policy rejections by reason.
# TYPE dex_id_jag_policy_rejections_total counter
dex_id_jag_policy_rejections_total{reason="unknown_audience"} 1
# HELP dex_id_jag_requests_total Total number of ID-JAG token exchange requests.
# TYPE dex_id_jag_requests_total counter
dex_id_jag_requests_total{result="denied"} 1
dex_id_jag_requests_total{result="issued"} 1
dex_id_jag_requests_total{result="rejected"} 1
# HELP dex_id_jag_scope_modifications_total Total number of ID-JAG requests where policy reduced the requested scopes.
# TYPE dex_id_jag_scope_modifications_total counter
dex_id_jag_scope_modifications_total 1
`), "dex_id_jag_requests_total", "dex_id_jag_policy_rejections_total", "dex_id_jag_scope_modifications_total"))
}

func TestTokenExchangeIDJAGUnverifiedEmail(t *testing.T) {
	httpServer, s := newIDJAGTestServer(t, map[string]grants.IDJAGPolicy{
		"client_1": {AllowedAudiences: []string{idJAGAudience}},
	}, nil)
	defer httpServer.Close()

	rr := postToken(t, s, httpServer, idJAGRequest(signSubjectToken(t, s, "client_1", "mock", false)))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var res tokens.Response
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&res))
	claims := decodeIDJAG(t, res.AccessToken)
	require.Empty(t, claims.Email)
}

func TestTokenExchangeIDJAGCustomExpiry(t *testing.T) {
	httpServer, s := newIDJAGTestServer(t, map[string]grants.IDJAGPolicy{
		"client_1": {AllowedAudiences: []string{idJAGAudience}},
	}, func(c *Config) {
		c.IDJAG.ValidFor = 10 * time.Minute
	})
	defer httpServer.Close()

	rr := postToken(t, s, httpServer, idJAGRequest(signSubjectToken(t, s, "client_1", "mock", true)))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var res tokens.Response
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&res))
	require.Equal(t, 600, res.ExpiresIn)
}

func TestTokenExchangeIDJAGRejected(t *testing.T) {
	policies := map[string]grants.IDJAGPolicy{
		"client_1":                  {AllowedAudiences: []string{idJAGAudience}, AllowedScopes: []string{"chat.read"}},
		"public_client":             {AllowedAudiences: []string{idJAGAudience}},
		"public_client_with_secret": {AllowedAudiences: []string{idJAGAudience}},
		"other_client":              {AllowedAudiences: []string{"https://calendar.example/"}},
	}

	tests := []struct {
		name string
		// modify adjusts a valid request; subject is a valid subject token for client_1.
		modify    func(vals url.Values, subject func(clientID, connID string, scopes ...string) string)
		wantCode  int
		wantError string
	}{
		{
			name: "public client",
			modify: func(vals url.Values, subject func(string, string, ...string) string) {
				vals.Set("client_id", "public_client")
				vals.Del("client_secret")
				vals.Set("subject_token", subject("public_client", "mock"))
			},
			wantCode:  http.StatusUnauthorized,
			wantError: oauth2.InvalidClient,
		},
		{
			name: "public client with a secret",
			modify: func(vals url.Values, subject func(string, string, ...string) string) {
				vals.Set("client_id", "public_client_with_secret")
				vals.Set("client_secret", "secret_2")
				vals.Set("subject_token", subject("public_client_with_secret", "mock"))
			},
			wantCode:  http.StatusUnauthorized,
			wantError: oauth2.InvalidClient,
		},
		{
			name:      "missing connector_id",
			modify:    func(vals url.Values, _ func(string, string, ...string) string) { vals.Del("connector_id") },
			wantCode:  http.StatusBadRequest,
			wantError: oauth2.InvalidRequest,
		},
		{
			name:      "missing audience",
			modify:    func(vals url.Values, _ func(string, string, ...string) string) { vals.Del("audience") },
			wantCode:  http.StatusBadRequest,
			wantError: oauth2.InvalidRequest,
		},
		{
			name: "more than one audience",
			modify: func(vals url.Values, _ func(string, string, ...string) string) {
				vals.Add("audience", "https://calendar.example/")
			},
			wantCode:  http.StatusBadRequest,
			wantError: oauth2.InvalidRequest,
		},
		{
			name: "more than one resource",
			modify: func(vals url.Values, _ func(string, string, ...string) string) {
				vals.Add("resource", "https://chat.example/a")
				vals.Add("resource", "https://chat.example/b")
			},
			wantCode:  http.StatusBadRequest,
			wantError: oauth2.InvalidRequest,
		},
		{
			name:      "relative resource",
			modify:    func(vals url.Values, _ func(string, string, ...string) string) { vals.Set("resource", "/api") },
			wantCode:  http.StatusBadRequest,
			wantError: oauth2.InvalidTarget,
		},
		{
			name: "resource with a fragment",
			modify: func(vals url.Values, _ func(string, string, ...string) string) {
				vals.Set("resource", "https://chat.example/api#x")
			},
			wantCode:  http.StatusBadRequest,
			wantError: oauth2.InvalidTarget,
		},
		{
			name: "access token subject_token_type",
			modify: func(vals url.Values, _ func(string, string, ...string) string) {
				vals.Set("subject_token_type", oauth2.TokenTypeAccess)
			},
			wantCode:  http.StatusBadRequest,
			wantError: oauth2.InvalidRequest,
		},
		{
			name:      "subject_token not signed by dex",
			modify:    func(vals url.Values, _ func(string, string, ...string) string) { vals.Set("subject_token", "foobar") },
			wantCode:  http.StatusBadRequest,
			wantError: oauth2.InvalidRequest,
		},
		{
			name: "subject_token issued to another client",
			modify: func(vals url.Values, subject func(string, string, ...string) string) {
				vals.Set("subject_token", subject("other_client", "mock"))
			},
			wantCode:  http.StatusBadRequest,
			wantError: oauth2.InvalidRequest,
		},
		{
			name: "subject_token issued to a client that has this one as a cross-client peer",
			modify: func(vals url.Values, subject func(string, string, ...string) string) {
				vals.Set("subject_token", subject("other_client", "mock", "audience:server:client_id:client_1"))
			},
			wantCode:  http.StatusBadRequest,
			wantError: oauth2.InvalidRequest,
		},
		{
			name: "subject_token from another connector",
			modify: func(vals url.Values, subject func(string, string, ...string) string) {
				vals.Set("subject_token", subject("client_1", "other"))
			},
			wantCode:  http.StatusBadRequest,
			wantError: oauth2.InvalidRequest,
		},
		{
			name: "audience unknown to dex",
			modify: func(vals url.Values, _ func(string, string, ...string) string) {
				vals.Set("audience", "https://unknown.example/")
			},
			wantCode:  http.StatusBadRequest,
			wantError: oauth2.InvalidTarget,
		},
		{
			name: "audience not allowed for the client",
			modify: func(vals url.Values, _ func(string, string, ...string) string) {
				vals.Set("audience", "https://calendar.example/")
			},
			wantCode:  http.StatusBadRequest,
			wantError: oauth2.AccessDenied,
		},
		{
			name:      "no requested scope allowed",
			modify:    func(vals url.Values, _ func(string, string, ...string) string) { vals.Set("scope", "chat.write") },
			wantCode:  http.StatusBadRequest,
			wantError: oauth2.InvalidScope,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpServer, s := newIDJAGTestServer(t, policies, nil)
			defer httpServer.Close()

			subject := func(clientID, connID string, scopes ...string) string {
				return signSubjectToken(t, s, clientID, connID, true, scopes...)
			}
			vals := idJAGRequest(subject("client_1", "mock"))
			tc.modify(vals, subject)

			rr := postToken(t, s, httpServer, vals)
			require.Equal(t, tc.wantCode, rr.Code, rr.Body.String())

			var res struct {
				Error string `json:"error"`
			}
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&res))
			require.Equal(t, tc.wantError, res.Error)
		})
	}
}

func TestTokenExchangeIDJAGClientWithoutPolicy(t *testing.T) {
	httpServer, s := newIDJAGTestServer(t, map[string]grants.IDJAGPolicy{
		"other_client": {AllowedAudiences: []string{idJAGAudience}},
	}, nil)
	defer httpServer.Close()

	rr := postToken(t, s, httpServer, idJAGRequest(signSubjectToken(t, s, "client_1", "mock", true)))
	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	require.Contains(t, rr.Body.String(), oauth2.AccessDenied)
}

func TestTokenExchangeIDJAGDisabled(t *testing.T) {
	httpServer, s := newIDJAGTestServer(t, nil, func(c *Config) {
		c.IDJAG = nil
	})
	defer httpServer.Close()

	rr := postToken(t, s, httpServer, idJAGRequest(signSubjectToken(t, s, "client_1", "mock", true)))
	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	require.Contains(t, rr.Body.String(), "Invalid requested_token_type.")
}

func TestTokenExchangeIDJAGEnabledPlainExchange(t *testing.T) {
	httpServer, s := newIDJAGTestServer(t, map[string]grants.IDJAGPolicy{
		"client_1": {AllowedAudiences: []string{idJAGAudience}},
	}, nil)
	defer httpServer.Close()

	vals := idJAGRequest("foobar")
	vals.Set("requested_token_type", oauth2.TokenTypeID)
	vals.Del("audience")

	rr := postToken(t, s, httpServer, vals)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var res tokens.Response
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&res))
	require.Equal(t, oauth2.TokenTypeID, res.IssuedTokenType)
	require.Equal(t, "bearer", res.TokenType)

	jws, err := jose.ParseSignedCompact(res.AccessToken, []jose.SignatureAlgorithm{jose.RS256})
	require.NoError(t, err)
	require.NotContains(t, jws.Signatures[0].Protected.ExtraHeaders, jose.HeaderType)
}

func TestTokenExchangeIDJAGExpiredSubject(t *testing.T) {
	now := time.Now()
	httpServer, s := newIDJAGTestServer(t, map[string]grants.IDJAGPolicy{
		"client_1": {AllowedAudiences: []string{idJAGAudience}},
	}, func(c *Config) {
		c.Now = func() time.Time { return now }
	})
	defer httpServer.Close()

	subjectToken := signSubjectToken(t, s, "client_1", "mock", true)
	now = now.Add(25 * time.Hour)

	rr := postToken(t, s, httpServer, idJAGRequest(subjectToken))
	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	require.Contains(t, rr.Body.String(), oauth2.InvalidRequest)
}

func TestDiscoveryIDJAG(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		httpServer, s := newTestServer(t, func(c *Config) {
			if enabled {
				c.IDJAG = &grants.IDJAGConfig{}
			}
		})
		defer httpServer.Close()

		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil))
		require.Equal(t, http.StatusOK, rr.Code)

		var res discovery.Document
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&res))
		if enabled {
			require.Equal(t, []string{oauth2.TokenTypeIDJAG}, res.IdentityChainingTokenTypes)
		} else {
			require.Empty(t, res.IdentityChainingTokenTypes)
		}
	}
}

func decodeIDJAG(t *testing.T, token string) tokens.IDJAGClaims {
	t.Helper()
	jws, err := jose.ParseSignedCompact(token, []jose.SignatureAlgorithm{jose.RS256})
	require.NoError(t, err)
	var claims tokens.IDJAGClaims
	require.NoError(t, json.Unmarshal(jws.UnsafePayloadWithoutVerification(), &claims))
	return claims
}

func jwtSubject(t *testing.T, token string) string {
	t.Helper()
	jws, err := jose.ParseSignedCompact(token, []jose.SignatureAlgorithm{jose.RS256})
	require.NoError(t, err)
	var claims struct {
		Subject string `json:"sub"`
	}
	require.NoError(t, json.Unmarshal(jws.UnsafePayloadWithoutVerification(), &claims))
	return claims.Subject
}

// TestTokenExchangeIDJAGNotADexToken checks that tokens dex signs for other
// purposes are not accepted as an ID token, and that an ID-JAG is not accepted
// where dex verifies its own tokens.
func TestTokenExchangeIDJAGNotADexToken(t *testing.T) {
	var sig signer.Signer
	httpServer, s := newIDJAGTestServer(t, map[string]grants.IDJAGPolicy{
		"client_1": {AllowedAudiences: []string{idJAGAudience, "client_1"}},
	}, func(c *Config) {
		sig = c.Signer
	})
	defer httpServer.Close()

	vals := idJAGRequest(signSubjectToken(t, s, "client_1", "mock", true))
	vals.Set("audience", "client_1")
	rr := postToken(t, s, httpServer, vals)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var res tokens.Response
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&res))
	idJAG := res.AccessToken

	rr = postToken(t, s, httpServer, idJAGRequest(idJAG))
	require.Equal(t, http.StatusBadRequest, rr.Code, "an ID-JAG is not a subject token: %s", rr.Body.String())

	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, httpServer.URL+"/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+idJAG)
	s.ServeHTTP(rr, req)
	require.NotEqual(t, http.StatusOK, rr.Code, "an ID-JAG is not an access token: %s", rr.Body.String())

	now := time.Now()
	logoutToken, err := json.Marshal(map[string]any{
		"iss":    httpServer.URL,
		"sub":    jwtSubject(t, signSubjectToken(t, s, "client_1", "mock", true)),
		"aud":    "client_1",
		"iat":    now.Unix(),
		"exp":    now.Add(time.Minute).Unix(),
		"jti":    "logout",
		"events": map[string]any{"http://schemas.openid.net/event/backchannel-logout": map[string]any{}},
	})
	require.NoError(t, err)
	signed, err := sig.Sign(t.Context(), logoutToken)
	require.NoError(t, err)
	rr = postToken(t, s, httpServer, idJAGRequest(signed))
	require.Equal(t, http.StatusBadRequest, rr.Code, "a logout token is not a subject token: %s", rr.Body.String())
}

func TestTokenExchangeIDJAGEndedSession(t *testing.T) {
	ctx := t.Context()
	httpServer, s := newTestServerWithSessions(t, func(c *Config) {
		require.NoError(t, c.Storage.CreateClient(ctx, storage.Client{ID: "bound", Secret: "secret", RefreshTokenLifetime: storage.RefreshTokenLifetimeSession}))
		require.NoError(t, c.Storage.CreateClient(ctx, storage.Client{ID: "client_1", Secret: "secret_1"}))
		c.IDJAG = &grants.IDJAGConfig{Policies: map[string]grants.IDJAGPolicy{
			"bound":    {AllowedAudiences: []string{idJAGAudience}},
			"client_1": {AllowedAudiences: []string{idJAGAudience}},
		}}
	})
	defer httpServer.Close()

	for clientID, secret := range map[string]string{"bound": "secret", "client_1": "secret_1"} {
		subjectToken, _, err := s.issuer.SignIDToken(ctx, tokens.Authorization{
			Client:      storage.Client{ID: clientID},
			Claims:      storage.Claims{UserID: "0-385-28089-0"},
			Scopes:      []string{"openid"},
			ConnectorID: "mock",
			SessionID:   "ended-session",
		}, "", "")
		require.NoError(t, err)

		vals := idJAGRequest(subjectToken)
		vals.Set("client_id", clientID)
		vals.Set("client_secret", secret)
		rr := postToken(t, s, httpServer, vals)
		if clientID == "bound" {
			require.Equal(t, http.StatusBadRequest, rr.Code, "the session of a session-bound client has ended: %s", rr.Body.String())
		} else {
			require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
		}
	}
}

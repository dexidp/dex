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

func TestTokenExchangeIDJAGDelegation(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	for _, test := range []struct {
		name      string
		edit      func(url.Values, map[string]any, map[string]any, *Config)
		wantError string
	}{
		{name: "actor"},
		{name: "configured lifetime", edit: func(_ url.Values, subject, actor map[string]any, _ *Config) {
			subject["exp"] = now.Add(10 * time.Minute).Unix()
			actor["exp"] = float64(now.Add(20 * time.Minute).Unix())
		}},
		{name: "chain", edit: func(_ url.Values, subject, _ map[string]any, _ *Config) {
			subject["act"] = map[string]any{"iss": "https://prior.example", "sub": "prior"}
			subject["scope"] = "chat.read"
		}},
		{name: "missing actor type", edit: func(values url.Values, _, _ map[string]any, _ *Config) { values.Del("actor_token_type") }, wantError: oauth2.InvalidRequest},
		{name: "unsupported actor type", edit: func(values url.Values, _, _ map[string]any, _ *Config) {
			values.Set("actor_token_type", oauth2.TokenTypeAccess)
		}, wantError: oauth2.InvalidRequest},
		{name: "invalid actor", edit: func(_ url.Values, _, actor map[string]any, _ *Config) { clear(actor) }, wantError: oauth2.InvalidRequest},
		{name: "actor audience", edit: func(_ url.Values, _, actor map[string]any, _ *Config) { actor["aud"] = "other" }, wantError: oauth2.InvalidRequest},
		{name: "may act denied", edit: func(_ url.Values, subject, _ map[string]any, _ *Config) {
			subject["may_act"] = map[string]any{"sub": "other"}
		}, wantError: oauth2.InvalidRequest},
		{name: "may act allowed", edit: func(_ url.Values, subject, _ map[string]any, _ *Config) {
			subject["may_act"] = map[string]any{"sub": delegationActorSubject, "iss": delegationActorIssuer}
		}},
		{name: "actor cannot disappear", edit: func(values url.Values, subject, _ map[string]any, _ *Config) {
			dropActor(values)
			subject["act"] = map[string]any{"sub": "prior"}
		}, wantError: oauth2.InvalidRequest},
		{name: "may act without actor", edit: func(values url.Values, subject, _ map[string]any, _ *Config) {
			dropActor(values)
			subject["may_act"] = map[string]any{"sub": delegationActorSubject}
		}, wantError: oauth2.InvalidRequest},
		{name: "scope cannot widen", edit: func(_ url.Values, subject, _ map[string]any, _ *Config) {
			subject["act"] = map[string]any{"sub": "prior"}
			subject["scope"] = "chat.write"
		}, wantError: oauth2.InvalidScope},
		{name: "delegation disabled", edit: func(_ url.Values, _, _ map[string]any, config *Config) { config.TokenExchangeDelegation = nil }, wantError: oauth2.InvalidRequest},
		{name: "target denied", edit: func(values url.Values, _, _ map[string]any, _ *Config) {
			values.Set("audience", "https://unknown.example")
		}, wantError: oauth2.InvalidTarget},
		{name: "subject expiry", edit: func(_ url.Values, subject, _ map[string]any, _ *Config) { subject["exp"] = now.Add(time.Minute).Unix() }},
	} {
		t.Run(test.name, func(t *testing.T) {
			subject := map[string]any{"aud": "gateway", "iat": now.Unix(), "exp": now.Add(4 * time.Minute).Unix()}
			actor := map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject, "aud": "gateway", "exp": float64(now.Add(2 * time.Minute).Unix())}
			values := exchangeValues("gateway", "mock", "", oauth2.TokenTypeID, true)
			values.Set("requested_token_type", oauth2.TokenTypeIDJAG)
			values.Set("audience", idJAGAudience)
			values.Set("scope", "chat.read chat.write")
			var signing signer.Signer
			httpServer, server, _, actorConnector := delegationServer(t, now, func(config *Config) {
				signing = config.Signer
				config.IDJAG = &grants.IDJAGConfig{Policies: map[string]grants.IDJAGPolicy{"gateway": {AllowedAudiences: []string{idJAGAudience}, AllowedScopes: []string{"chat.read"}}}}
				if test.edit != nil {
					test.edit(values, subject, actor, config)
				}
			})
			defer httpServer.Close()
			actorConnector.Claims = actor
			subject["iss"] = httpServer.URL
			subject["sub"] = jwtSubject(t, signSubjectToken(t, server, "gateway", "mock", true))
			payload, err := json.Marshal(subject)
			require.NoError(t, err)
			signed, err := signing.Sign(t.Context(), payload)
			require.NoError(t, err)
			values.Set("subject_token", signed)
			request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, httpServer.URL+"/token", strings.NewReader(values.Encode()))
			require.NoError(t, err)
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response, err := httpServer.Client().Do(request)
			require.NoError(t, err)
			defer response.Body.Close()
			if test.wantError != "" {
				require.Equal(t, http.StatusBadRequest, response.StatusCode)
				var failure struct {
					Error string `json:"error"`
				}
				require.NoError(t, json.NewDecoder(response.Body).Decode(&failure))
				require.Equal(t, test.wantError, failure.Error)
				return
			}
			require.Equal(t, http.StatusOK, response.StatusCode)
			var result tokens.Response
			require.NoError(t, json.NewDecoder(response.Body).Decode(&result))
			require.Equal(t, "N_A", result.TokenType)
			require.Equal(t, oauth2.TokenTypeIDJAG, result.IssuedTokenType)
			require.Equal(t, "chat.read", result.Scope)
			signature, err := jose.ParseSignedCompact(result.AccessToken, []jose.SignatureAlgorithm{jose.RS256})
			require.NoError(t, err)
			verified, err := signature.Verify(&testKey.PublicKey)
			require.NoError(t, err)
			var claims tokens.IDJAGClaims
			require.NoError(t, json.Unmarshal(verified, &claims))
			require.Equal(t, oauth2.JWTTypeIDJAG, signature.Signatures[0].Protected.ExtraHeaders[jose.HeaderType])
			require.Equal(t, idJAGAudience, claims.Audience)
			require.Equal(t, subject["sub"], claims.Subject)
			require.Equal(t, delegationActorSubject, claims.Act.Subject)
			require.Equal(t, delegationActorIssuer, claims.Act.Issuer)
			require.Equal(t, "chat.read", claims.Scope)
			expiry := now.Add(2 * time.Minute).Unix()
			if test.name == "subject expiry" {
				expiry = now.Add(time.Minute).Unix()
			}
			if test.name == "configured lifetime" {
				expiry = now.Add(5 * time.Minute).Unix()
			}
			require.Equal(t, expiry, claims.Expiry)
			require.Equal(t, int(expiry-now.Unix()), result.ExpiresIn)
			if test.name == "chain" {
				require.Equal(t, "prior", claims.Act.Act.Subject)
			}
			require.NotContains(t, string(verified), "actor_token")
		})
	}
}

func TestTokenExchangeActorOIDC(t *testing.T) {
	var actorIssuer *httptest.Server
	actorIssuer = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/keys" {
			json.NewEncoder(writer).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &testKey.PublicKey, KeyID: "actor", Algorithm: "RS256", Use: "sig"}}})
			return
		}
		json.NewEncoder(writer).Encode(map[string]any{"issuer": actorIssuer.URL, "authorization_endpoint": actorIssuer.URL + "/auth", "token_endpoint": actorIssuer.URL + "/token", "jwks_uri": actorIssuer.URL + "/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
	}))
	defer actorIssuer.Close()
	connectorConfig, err := json.Marshal(map[string]any{"issuer": actorIssuer.URL, "clientID": "gateway", "clientSecret": "secret"})
	require.NoError(t, err)
	httpServer, server := newTestServerWith(t, []storage.Connector{mockConnector("mock"), {ID: "actor", Type: "oidc", Config: connectorConfig}}, func(config *Config) {
		require.NoError(t, config.Storage.CreateClient(t.Context(), storage.Client{ID: "gateway", Secret: "secret"}))
		config.TokenExchangeDelegation = []grants.DelegationPolicy{{ClientID: "gateway", ActorConnectors: []string{"actor"}}}
		config.IDJAG = &grants.IDJAGConfig{Policies: map[string]grants.IDJAGPolicy{"gateway": {AllowedAudiences: []string{idJAGAudience}, AllowedScopes: []string{"chat.read"}}}}
	})
	defer httpServer.Close()
	actorSigner, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: &jose.JSONWebKey{Key: testKey, KeyID: "actor"}}, nil)
	require.NoError(t, err)
	for _, output := range []string{oauth2.TokenTypeAccess, oauth2.TokenTypeIDJAG} {
		for _, condition := range []string{"valid", "expired", "tampered"} {
			t.Run(output+"/"+condition, func(t *testing.T) {
				expiry := time.Now().Add(time.Minute)
				if condition == "expired" {
					expiry = time.Now().Add(-time.Minute)
				}
				claims, err := json.Marshal(map[string]any{"iss": actorIssuer.URL, "sub": "workload", "aud": "gateway", "exp": expiry.Unix()})
				require.NoError(t, err)
				signature, err := actorSigner.Sign(claims)
				require.NoError(t, err)
				actorToken, err := signature.CompactSerialize()
				require.NoError(t, err)
				if condition == "tampered" {
					actorToken += "x"
				}
				values := exchangeValues("gateway", "", signSubjectToken(t, server, "gateway", "mock", true), oauth2.TokenTypeID, true)
				values.Set("actor_token", actorToken)
				values.Set("requested_token_type", output)
				values.Set("scope", "openid")
				if output == oauth2.TokenTypeIDJAG {
					values.Set("connector_id", "mock")
					values.Set("audience", idJAGAudience)
					values.Set("scope", "chat.read")
				}
				request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, httpServer.URL+"/token", strings.NewReader(values.Encode()))
				require.NoError(t, err)
				request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				response, err := httpServer.Client().Do(request)
				require.NoError(t, err)
				defer response.Body.Close()
				if condition != "valid" {
					require.Equal(t, http.StatusBadRequest, response.StatusCode)
					return
				}
				require.Equal(t, http.StatusOK, response.StatusCode)
				var result tokens.Response
				require.NoError(t, json.NewDecoder(response.Body).Decode(&result))
				signed, err := jose.ParseSignedCompact(result.AccessToken, []jose.SignatureAlgorithm{jose.RS256})
				require.NoError(t, err)
				payload, err := signed.Verify(&testKey.PublicKey)
				require.NoError(t, err)
				var issued struct {
					Act tokens.ActorClaims `json:"act"`
				}
				require.NoError(t, json.Unmarshal(payload, &issued))
				require.Equal(t, "workload", issued.Act.Subject)
				require.Equal(t, actorIssuer.URL, issued.Act.Issuer)
			})
		}
	}
}

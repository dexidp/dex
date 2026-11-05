package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/connector/mock"
	"github.com/dexidp/dex/server/grants"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

const (
	delegationActorIssuer  = "https://actors.example.test"
	delegationActorSubject = "workload/session-1"
)

// delegationServer serves a subject connector "mock" and an actor connector
// "actor". Client "gateway" delegates through "actor" and may name "backend",
// "relay" and "untrusted" (which does not trust it) as audiences; "relay"
// delegates again a token gateway obtained for it; "plain" exchanges without
// an actor; "public" is a public client listed for delegation.
func delegationServer(t *testing.T, now time.Time, updateConfig func(c *Config)) (*httptest.Server, *Server, *mock.Callback, *mock.Callback) {
	t.Helper()
	ctx := t.Context()
	actor := mockConnector("actor")
	actor.GrantTypes = []string{oauth2.GrantTypeTokenExchange}
	httpServer, s := newTestServerWith(t, []storage.Connector{mockConnector("mock"), actor}, func(c *Config) {
		c.AllowedGrantTypes = []string{oauth2.GrantTypeTokenExchange, oauth2.GrantTypeRefreshToken, oauth2.GrantTypeClientCredentials}
		c.Now = func() time.Time { return now }
		c.IDTokensValidFor = time.Hour
		c.TokenExchangeDelegation = []grants.DelegationPolicy{
			{ClientID: "gateway", ActorConnectors: []string{"actor"}, AllowedAudiences: []string{"backend", "relay", "untrusted"}, AllowedScopes: []string{"tools:read", "tools:write"}},
			{ClientID: "relay", ActorConnectors: []string{"actor"}, AllowedScopes: []string{"tools:read"}},
			{ClientID: "public", ActorConnectors: []string{"actor"}},
		}
		for _, client := range []storage.Client{
			{ID: "gateway", Secret: "secret", TrustedPeers: []string{"plain"}},
			{ID: "plain", Secret: "secret"},
			{ID: "public", Secret: "secret", Public: true},
			{ID: "backend", Secret: "secret", TrustedPeers: []string{"gateway"}},
			{ID: "relay", Secret: "secret", TrustedPeers: []string{"gateway"}},
			{ID: "untrusted", Secret: "secret"},
		} {
			require.NoError(t, c.Storage.CreateClient(ctx, client))
		}
		if updateConfig != nil {
			updateConfig(c)
		}
	})
	subjectConn, ok := s.connectors.Cached("mock")
	require.True(t, ok)
	actorConn, ok := s.connectors.Cached("actor")
	require.True(t, ok)
	return httpServer, s, subjectConn.Connector.(*mock.Callback), actorConn.Connector.(*mock.Callback)
}

func postToken(t *testing.T, s *Server, httpServer *httptest.Server, vals url.Values) *httptest.ResponseRecorder {
	t.Helper()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, httpServer.URL+"/token", strings.NewReader(vals.Encode()))
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	s.ServeHTTP(rr, req)
	return rr
}

// jwtPayload decodes a token's claims without verifying it.
func jwtPayload(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims map[string]any
	require.NoError(t, json.Unmarshal(raw, &claims))
	return claims
}

func introspect(t *testing.T, s *Server, httpServer *httptest.Server, token string) map[string]any {
	t.Helper()
	data := url.Values{}
	data.Set("token", token)
	req := httptest.NewRequest(http.MethodPost, httpServer.URL+"/token/introspect", strings.NewReader(data.Encode()))
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	ir := httptest.NewRecorder()
	s.ServeHTTP(ir, req)
	require.Equal(t, http.StatusOK, ir.Code, ir.Body.String())
	var intro map[string]any
	require.NoError(t, json.Unmarshal(ir.Body.Bytes(), &intro))
	return intro
}

func dropActor(v url.Values) {
	v.Del("actor_token")
	v.Del("actor_token_type")
	v.Del("actor_connector_id")
}

func TestHandleTokenExchangeDelegation(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	subjectClaims := func() map[string]any {
		return map[string]any{"iss": "https://people.example.test", "sub": "person-1", "aud": "gateway", "exp": float64(now.Add(30 * time.Minute).Unix())}
	}
	actorClaims := func() map[string]any {
		return map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject, "aud": "gateway", "exp": float64(now.Add(10 * time.Minute).Unix())}
	}
	// request builds a complete delegated exchange for "gateway"; cases edit it.
	request := func() url.Values {
		vals := make(url.Values)
		vals.Set("grant_type", oauth2.GrantTypeTokenExchange)
		vals.Set("client_id", "gateway")
		vals.Set("client_secret", "secret")
		vals.Set("connector_id", "mock")
		vals.Set("scope", "openid")
		vals.Set("subject_token", "subject")
		vals.Set("subject_token_type", oauth2.TokenTypeID)
		vals.Set("actor_connector_id", "actor")
		vals.Set("actor_token", "actor")
		vals.Set("actor_token_type", oauth2.TokenTypeJWT)
		return vals
	}
	plain := func(v url.Values) {
		v.Set("client_id", "plain")
		dropActor(v)
	}

	tests := []struct {
		name           string
		edit           func(vals url.Values)
		subject        func(claims map[string]any)
		actor          func(claims map[string]any)
		subjectInvalid bool
		actorInvalid   bool
		expectedCode   int
		expectedType   string
		expectedError  string
		expectedAud    any
		expectedScope  string
	}{
		{name: "delegated access token", expectedCode: http.StatusOK},
		{name: "delegated id token", edit: func(v url.Values) { v.Set("requested_token_type", oauth2.TokenTypeID) }, expectedCode: http.StatusOK},
		{name: "id_token actor type", edit: func(v url.Values) { v.Set("actor_token_type", oauth2.TokenTypeID) }, expectedCode: http.StatusOK},
		{name: "array actor audience", actor: func(c map[string]any) { c["aud"] = []any{"other", "gateway"} }, expectedCode: http.StatusOK},
		{name: "may_act sub and iss", subject: func(c map[string]any) {
			c["may_act"] = map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject}
		}, expectedCode: http.StatusOK},
		{name: "may_act sub with a single actor connector", subject: func(c map[string]any) { c["may_act"] = map[string]any{"sub": delegationActorSubject} }, expectedCode: http.StatusOK},
		{name: "may_act client_id", subject: func(c map[string]any) {
			c["may_act"] = map[string]any{"client_id": "gateway", "email": "a@example.test"}
		}, expectedCode: http.StatusOK},
		{name: "audience parameter", edit: func(v url.Values) { v.Set("audience", "backend") }, expectedCode: http.StatusOK, expectedAud: []any{"backend", "gateway"}},
		{name: "audience own client", edit: func(v url.Values) { v.Set("audience", "gateway") }, expectedCode: http.StatusOK, expectedAud: "gateway"},
		{name: "allowed scope", edit: func(v url.Values) { v.Set("scope", "openid tools:read") }, expectedCode: http.StatusOK, expectedScope: "openid tools:read"},
		{name: "plain client without actor", edit: plain, expectedCode: http.StatusOK},
		{name: "plain client, may_act names it", edit: plain, subject: func(c map[string]any) { c["may_act"] = map[string]any{"client_id": "plain"} }, expectedCode: http.StatusOK},

		{name: "actor_token without type", edit: func(v url.Values) { v.Del("actor_token_type") }, expectedCode: http.StatusBadRequest, expectedError: "actor_token and actor_token_type must be sent together"},
		{name: "actor_token_type without token", edit: func(v url.Values) { v.Del("actor_token") }, expectedCode: http.StatusBadRequest, expectedError: "actor_token and actor_token_type must be sent together"},
		{name: "plain client with actor", edit: func(v url.Values) { v.Set("client_id", "plain") }, expectedCode: http.StatusBadRequest, expectedError: "delegation is not enabled for this client"},
		{name: "public client", edit: func(v url.Values) { v.Set("client_id", "public") }, expectedCode: http.StatusBadRequest, expectedError: "delegation is not enabled for this client"},
		{name: "delegating client without actor", edit: dropActor, expectedCode: http.StatusBadRequest, expectedError: "this client must present an actor_token"},
		{name: "actor_token without actor_connector_id", edit: func(v url.Values) { v.Del("actor_connector_id") }, expectedCode: http.StatusBadRequest, expectedError: "actor_connector_id is required with actor_token"},
		{name: "actor connector as subject connector", edit: func(v url.Values) { v.Set("connector_id", "actor") }, expectedCode: http.StatusBadRequest, expectedError: "connector only verifies actor tokens"},
		{name: "actor connector as subject connector for plain client", edit: func(v url.Values) {
			plain(v)
			v.Set("connector_id", "actor")
		}, expectedCode: http.StatusBadRequest, expectedError: "connector only verifies actor tokens"},
		{name: "access_token subject", edit: func(v url.Values) { v.Set("subject_token_type", oauth2.TokenTypeAccess) }, expectedCode: http.StatusBadRequest, expectedError: "token type not supported for delegation"},
		{name: "access_token actor", edit: func(v url.Values) { v.Set("actor_token_type", oauth2.TokenTypeAccess) }, expectedCode: http.StatusBadRequest, expectedError: "token type not supported for delegation"},
		{name: "actor connector not in policy", edit: func(v url.Values) { v.Set("actor_connector_id", "mock") }, expectedCode: http.StatusBadRequest, expectedError: "actor connector not allowed for this client"},
		{name: "subject token does not verify", subjectInvalid: true, expectedCode: http.StatusBadRequest, expectedError: "invalid subject_token"},
		{name: "subject token does not verify, impersonation", edit: plain, subjectInvalid: true, expectedCode: http.StatusBadRequest, expectedError: "invalid subject_token"},
		{name: "actor token does not verify", actorInvalid: true, expectedCode: http.StatusBadRequest, expectedError: "invalid actor_token"},
		{name: "actor audience is another client", actor: func(c map[string]any) { c["aud"] = "other" }, expectedCode: http.StatusBadRequest, expectedError: "actor token audience does not include this client"},
		{name: "actor without audience", actor: func(c map[string]any) { delete(c, "aud") }, expectedCode: http.StatusBadRequest, expectedError: "actor token audience does not include this client"},
		{name: "subject act without sub", subject: func(c map[string]any) { c["act"] = map[string]any{"iss": "https://earlier.example.test"} }, expectedCode: http.StatusBadRequest, expectedError: "invalid act claim in subject token"},
		{name: "plain client with delegated subject", edit: plain, subject: func(c map[string]any) { c["act"] = map[string]any{"sub": "earlier"} }, expectedCode: http.StatusBadRequest, expectedError: "delegated subject tokens require an actor_token"},
		{name: "may_act names another actor", subject: func(c map[string]any) { c["may_act"] = map[string]any{"sub": "someone-else"} }, expectedCode: http.StatusBadRequest, expectedError: "actor is not authorized by the subject token's may_act claim"},
		{name: "may_act with an unknown member", subject: func(c map[string]any) { c["may_act"] = map[string]any{"client_id": "gateway", "groups": []any{"x"}} }, expectedCode: http.StatusBadRequest, expectedError: "actor is not authorized by the subject token's may_act claim"},
		{name: "plain client, may_act names a sub", edit: plain, subject: func(c map[string]any) { c["may_act"] = map[string]any{"sub": delegationActorSubject} }, expectedCode: http.StatusBadRequest, expectedError: "actor is not authorized by the subject token's may_act claim"},
		{name: "plain client, may_act names another client", edit: plain, subject: func(c map[string]any) { c["may_act"] = map[string]any{"client_id": "gateway"} }, expectedCode: http.StatusBadRequest, expectedError: "actor is not authorized by the subject token's may_act claim"},
		{name: "subject audience is another client", subject: func(c map[string]any) { c["aud"] = []any{"other"} }, expectedCode: http.StatusBadRequest, expectedError: "subject token audience does not include this client"},
		{name: "audience outside the policy", edit: func(v url.Values) { v.Set("audience", "plain") }, expectedCode: http.StatusBadRequest, expectedType: oauth2.InvalidTarget, expectedError: "audience not allowed for delegation"},
		{name: "audience in the policy that does not trust the client", edit: func(v url.Values) { v.Set("audience", "untrusted") }, expectedCode: http.StatusBadRequest, expectedType: oauth2.InvalidTarget, expectedError: "audience not allowed for delegation"},
		{name: "resource", edit: func(v url.Values) { v.Set("resource", "https://backend.example.test") }, expectedCode: http.StatusBadRequest, expectedType: oauth2.InvalidTarget, expectedError: "resource is not supported, use audience"},
		{name: "cross-client scope", edit: func(v url.Values) { v.Set("scope", "openid audience:server:client_id:backend") }, expectedCode: http.StatusBadRequest, expectedType: oauth2.InvalidScope, expectedError: "use the audience parameter for delegation"},
		{name: "scope outside the policy", edit: func(v url.Values) { v.Set("scope", "openid admin") }, expectedCode: http.StatusBadRequest, expectedType: oauth2.InvalidScope, expectedError: "scope not allowed for delegation"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpServer, s, subjectConn, actorConn := delegationServer(t, now, nil)
			defer httpServer.Close()

			if !tc.subjectInvalid {
				subjectConn.Claims = subjectClaims()
				if tc.subject != nil {
					tc.subject(subjectConn.Claims)
				}
			} else {
				subjectConn.Claims = nil
				subjectConn.TokenIdentityFails = true
			}
			if !tc.actorInvalid {
				actorConn.Claims = actorClaims()
				if tc.actor != nil {
					tc.actor(actorConn.Claims)
				}
			}
			vals := request()
			if tc.edit != nil {
				tc.edit(vals)
			}

			rr := postToken(t, s, httpServer, vals)
			require.Equal(t, tc.expectedCode, rr.Code, rr.Body.String())
			if tc.expectedCode != http.StatusOK {
				var oerr map[string]string
				require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &oerr))
				expectedType := tc.expectedType
				if expectedType == "" {
					expectedType = oauth2.InvalidRequest
				}
				require.Equal(t, expectedType, oerr["error"])
				require.Equal(t, tc.expectedError, oerr["error_description"])
				return
			}

			var res tokens.Response
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
			claims := jwtPayload(t, res.AccessToken)
			if vals.Get("client_id") == "plain" {
				require.NotContains(t, claims, "act")
				require.NotContains(t, claims, "client_id")
				require.NotContains(t, claims, "scope")
				require.Empty(t, res.Scope)
				require.Equal(t, float64(now.Add(time.Hour).Unix()), claims["exp"], "connector impersonation keeps its default lifetime")
				return
			}
			require.Equal(t, map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject}, claims["act"])
			require.Equal(t, "gateway", claims["client_id"])
			require.Equal(t, float64(now.Add(10*time.Minute).Unix()), claims["exp"], "bounded by the actor token")
			require.Equal(t, 600, res.ExpiresIn, "expires_in matches the capped exp")
			expectedAud := tc.expectedAud
			if expectedAud == nil {
				expectedAud = "gateway"
			}
			require.Equal(t, expectedAud, claims["aud"])
			expectedScope := tc.expectedScope
			if expectedScope == "" {
				expectedScope = "openid"
			}
			require.Equal(t, expectedScope, claims["scope"])
			require.Equal(t, expectedScope, res.Scope, "the response reports the granted scopes")
			require.NotContains(t, claims, "may_act")
		})
	}
}

// Actor parameters on other grants are ignored (RFC 6749 §3.2), so the grant
// answers as it would without them.
func TestHandleTokenExchangeDelegationParamsIgnoredElsewhere(t *testing.T) {
	httpServer, s, _, _ := delegationServer(t, time.Now(), nil)
	defer httpServer.Close()
	vals := url.Values{}
	vals.Set("grant_type", oauth2.GrantTypeClientCredentials)
	vals.Set("client_id", "plain")
	vals.Set("client_secret", "secret")
	vals.Set("actor_token", "actor")
	vals.Set("actor_token_type", oauth2.TokenTypeJWT)
	rr := postToken(t, s, httpServer, vals)
	require.NotContains(t, rr.Body.String(), "actor_token")
}

func TestHandleTokenExchangeDelegationBoundedBySubject(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	httpServer, s, subjectConn, actorConn := delegationServer(t, now, nil)
	defer httpServer.Close()
	subjectConn.Claims = map[string]any{"sub": "person-1", "aud": "gateway", "exp": float64(now.Add(5 * time.Minute).Unix())}
	actorConn.Claims = map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject, "aud": "gateway", "exp": float64(now.Add(10 * time.Minute).Unix())}

	vals := exchangeValues("gateway", "mock", "subject", oauth2.TokenTypeID, true)
	vals.Del("scope")
	rr := postToken(t, s, httpServer, vals)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var res tokens.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
	claims := jwtPayload(t, res.AccessToken)
	require.Equal(t, float64(now.Add(5*time.Minute).Unix()), claims["exp"])
	require.NotContains(t, claims, "scope", "no scopes, no scope claim")
	require.Empty(t, res.Scope)
}

func TestHandleTokenExchangeDelegationIntrospection(t *testing.T) {
	httpServer, s, subjectConn, actorConn := delegationServer(t, time.Now(), nil)
	defer httpServer.Close()
	subjectConn.Claims = map[string]any{"sub": "person-1", "aud": "gateway"}
	actorConn.Claims = map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject, "aud": "gateway"}

	vals := exchangeValues("gateway", "mock", "subject", oauth2.TokenTypeID, true)
	vals.Set("scope", "openid tools:read")
	token := exchangedToken(t, s, httpServer, vals)

	intro := introspect(t, s, httpServer, token)
	require.Equal(t, true, intro["active"])
	require.Equal(t, map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject}, intro["act"], "act is a top-level member")
	require.Equal(t, "openid tools:read", intro["scope"])
	require.NotContains(t, intro["ext"], "act")
}

// exchangeValues is a token-exchange form for client, with or without the actor.
func exchangeValues(client, connectorID, subjectToken, subjectType string, withActor bool) url.Values {
	vals := make(url.Values)
	vals.Set("grant_type", oauth2.GrantTypeTokenExchange)
	vals.Set("client_id", client)
	vals.Set("client_secret", "secret")
	setNonEmpty(vals, "connector_id", connectorID)
	vals.Set("scope", "openid email profile groups")
	vals.Set("subject_token", subjectToken)
	vals.Set("subject_token_type", subjectType)
	if withActor {
		vals.Set("actor_connector_id", "actor")
		vals.Set("actor_token", "actor")
		vals.Set("actor_token_type", oauth2.TokenTypeJWT)
	}
	return vals
}

func exchangedToken(t *testing.T, s *Server, httpServer *httptest.Server, vals url.Values) string {
	t.Helper()
	rr := postToken(t, s, httpServer, vals)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var res tokens.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
	return res.AccessToken
}

// A subject token Dex issued itself is verified with Dex's keys when no
// connector_id is sent, and the issued token keeps its sub.
func TestHandleTokenExchangeSelfIssuedSubject(t *testing.T) {
	now := time.Now()
	httpServer, s, subjectConn, actorConn := delegationServer(t, now, nil)
	defer httpServer.Close()
	subjectConn.Claims = map[string]any{"sub": "person-1", "aud": []any{"gateway", "plain"}}
	actorConn.Claims = map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject, "aud": "gateway"}

	t.Run("impersonation keeps sub and never outlives the subject", func(t *testing.T) {
		first := exchangedToken(t, s, httpServer, exchangeValues("plain", "mock", "subject", oauth2.TokenTypeID, false))
		second := exchangedToken(t, s, httpServer, exchangeValues("plain", "", first, oauth2.TokenTypeAccess, false))
		a, b := jwtPayload(t, first), jwtPayload(t, second)
		require.Equal(t, a["sub"], b["sub"])
		require.Equal(t, a["email"], b["email"])
		require.Equal(t, a["groups"], b["groups"])
		require.Equal(t, a["name"], b["name"])
		require.NotContains(t, b, "act")
		require.Equal(t, a["exp"], b["exp"])
	})

	t.Run("delegation keeps sub", func(t *testing.T) {
		forGateway := exchangeValues("plain", "mock", "subject", oauth2.TokenTypeID, false)
		forGateway.Set("scope", "openid audience:server:client_id:gateway")
		first := exchangedToken(t, s, httpServer, forGateway)
		second := exchangedToken(t, s, httpServer, exchangeValues("gateway", "", first, oauth2.TokenTypeAccess, true))
		a, b := jwtPayload(t, first), jwtPayload(t, second)
		require.Equal(t, a["sub"], b["sub"])
		require.Equal(t, map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject}, b["act"])
	})

	t.Run("issued to another client", func(t *testing.T) {
		first := exchangedToken(t, s, httpServer, exchangeValues("plain", "mock", "subject", oauth2.TokenTypeID, false))
		rr := postToken(t, s, httpServer, exchangeValues("gateway", "", first, oauth2.TokenTypeAccess, true))
		require.Equal(t, http.StatusBadRequest, rr.Code)
		require.Contains(t, rr.Body.String(), "subject token audience does not include this client")
	})

	t.Run("delegated token without actor", func(t *testing.T) {
		forBackend := exchangeValues("gateway", "mock", "subject", oauth2.TokenTypeID, true)
		forBackend.Set("audience", "backend")
		first := exchangedToken(t, s, httpServer, forBackend)
		rr := postToken(t, s, httpServer, exchangeValues("backend", "", first, oauth2.TokenTypeAccess, false))
		require.Equal(t, http.StatusBadRequest, rr.Code)
		require.Contains(t, rr.Body.String(), "delegated subject tokens require an actor_token")
	})

	t.Run("not a Dex token", func(t *testing.T) {
		rr := postToken(t, s, httpServer, exchangeValues("plain", "", "foobar", oauth2.TokenTypeID, false))
		require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
		require.Contains(t, rr.Body.String(), "invalid subject_token")
	})
}

// The connector encoded in a Dex-issued subject token's sub must pass the
// requesting client's connector policy and must not be an actor connector.
func TestHandleTokenExchangeSelfIssuedSubjectConnector(t *testing.T) {
	httpServer, s, _, _ := delegationServer(t, time.Now(), func(c *Config) {
		require.NoError(t, c.Storage.CreateClient(t.Context(), storage.Client{ID: "restricted", Secret: "secret", AllowedConnectors: []string{"other"}}))
	})
	defer httpServer.Close()
	sign := func(client, connectorID string) string {
		token, _, err := s.issuer.SignAccessToken(t.Context(), tokens.Authorization{
			Client: storage.Client{ID: client}, Claims: storage.Claims{UserID: "person-1"}, ConnectorID: connectorID,
		})
		require.NoError(t, err)
		return token
	}

	rr := postToken(t, s, httpServer, exchangeValues("restricted", "", sign("restricted", "mock"), oauth2.TokenTypeAccess, false))
	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	require.Contains(t, rr.Body.String(), "Connector not allowed for this client.")

	rr = postToken(t, s, httpServer, exchangeValues("plain", "", sign("plain", "actor"), oauth2.TokenTypeAccess, false))
	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	require.Contains(t, rr.Body.String(), "connector only verifies actor tokens")
}

// A Dex-issued subject token bound to a session is refused once the session
// ends, and only exchanged by a client that is bound to it as well; the issued
// token keeps the sid, so introspection follows the session.
func TestHandleTokenExchangeSelfIssuedSubjectSession(t *testing.T) {
	httpServer, s := newTestServerWithSessions(t, func(c *Config) {
		for _, client := range []storage.Client{
			{ID: "app", Secret: "secret", RefreshTokenLifetime: storage.RefreshTokenLifetimeSession},
			{ID: "bound", Secret: "secret", RefreshTokenLifetime: storage.RefreshTokenLifetimeSession, TrustedPeers: []string{"app"}},
			{ID: "unbound", Secret: "secret", TrustedPeers: []string{"app"}},
		} {
			require.NoError(t, c.Storage.CreateClient(t.Context(), client))
		}
	})
	defer httpServer.Close()
	newTestAuthSession(t, s, "person-1", "mock", "session-1")
	token, _, err := s.issuer.SignAccessToken(t.Context(), tokens.Authorization{
		Client:      storage.Client{ID: "app"},
		Claims:      storage.Claims{UserID: "person-1"},
		Scopes:      []string{"openid", "audience:server:client_id:bound", "audience:server:client_id:unbound"},
		ConnectorID: "mock",
		SessionID:   "session-1",
	})
	require.NoError(t, err)

	rr := postToken(t, s, httpServer, exchangeValues("unbound", "", token, oauth2.TokenTypeAccess, false))
	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	require.Contains(t, rr.Body.String(), "subject token is bound to a session this client does not end with")

	exchanged := exchangedToken(t, s, httpServer, exchangeValues("bound", "", token, oauth2.TokenTypeAccess, false))
	require.Equal(t, "session-1", jwtPayload(t, exchanged)["sid"])
	require.Equal(t, true, introspect(t, s, httpServer, exchanged)["active"])

	require.NoError(t, s.storage.DeleteAuthSession(t.Context(), "session-1"))
	require.Equal(t, false, introspect(t, s, httpServer, exchanged)["active"], "ending the session deactivates the exchanged token")
	rr = postToken(t, s, httpServer, exchangeValues("bound", "", token, oauth2.TokenTypeAccess, false))
	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	require.Contains(t, rr.Body.String(), "subject token session has ended")
}

// The DEP's chain: gateway obtains a delegated token for relay, and relay
// exchanges it with its own actor token. The earlier actor is nested (RFC 8693
// §4.1), and scope and exp only narrow.
func TestHandleTokenExchangeDelegationChain(t *testing.T) {
	now := time.Now()
	httpServer, s, subjectConn, actorConn := delegationServer(t, now, nil)
	defer httpServer.Close()
	subjectConn.Claims = map[string]any{"sub": "person-1", "aud": "gateway"}
	actorConn.Claims = map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject, "aud": "gateway", "exp": float64(now.Add(10 * time.Minute).Unix())}

	forRelay := exchangeValues("gateway", "mock", "subject", oauth2.TokenTypeID, true)
	forRelay.Set("scope", "openid tools:read tools:write")
	forRelay.Set("audience", "relay")
	first := exchangedToken(t, s, httpServer, forRelay)
	a := jwtPayload(t, first)
	require.Equal(t, "openid tools:read tools:write", a["scope"])
	require.Equal(t, []any{"relay", "gateway"}, a["aud"])

	actorConn.Claims = map[string]any{"iss": "https://cluster.example.test", "sub": "system:serviceaccount:apps:relay", "aud": "relay", "exp": float64(now.Add(time.Hour).Unix())}
	again := exchangeValues("relay", "", first, oauth2.TokenTypeAccess, true)
	again.Set("scope", "tools:read")
	rr := postToken(t, s, httpServer, again)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var res tokens.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
	b := jwtPayload(t, res.AccessToken)

	require.Equal(t, a["sub"], b["sub"])
	require.Equal(t, "relay", b["client_id"])
	require.Equal(t, map[string]any{
		"iss": "https://cluster.example.test",
		"sub": "system:serviceaccount:apps:relay",
		"act": map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject},
	}, b["act"])
	require.Equal(t, "tools:read", b["scope"])
	require.Equal(t, "tools:read", res.Scope)
	require.Equal(t, a["exp"], b["exp"], "never outlives the subject token")

	intro := introspect(t, s, httpServer, res.AccessToken)
	require.Equal(t, b["act"], intro["act"])
	require.Equal(t, "tools:read", intro["scope"])

	t.Run("a scope the subject does not carry", func(t *testing.T) {
		v := exchangeValues("relay", "", first, oauth2.TokenTypeAccess, true)
		v.Set("scope", "tools:read")
		forOnlyRead := exchangeValues("gateway", "mock", "subject", oauth2.TokenTypeID, true)
		forOnlyRead.Set("scope", "openid tools:write")
		forOnlyRead.Set("audience", "relay")
		actorConn.Claims = map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject, "aud": "gateway"}
		v.Set("subject_token", exchangedToken(t, s, httpServer, forOnlyRead))
		actorConn.Claims = map[string]any{"iss": "https://cluster.example.test", "sub": "system:serviceaccount:apps:relay", "aud": "relay"}
		rr := postToken(t, s, httpServer, v)
		require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
		require.Contains(t, rr.Body.String(), "subject token does not carry the requested scopes")
	})

	t.Run("a delegated subject without scope grants none", func(t *testing.T) {
		noScope := exchangeValues("gateway", "mock", "subject", oauth2.TokenTypeID, true)
		noScope.Del("scope")
		noScope.Set("audience", "relay")
		actorConn.Claims = map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject, "aud": "gateway"}
		subjectToken := exchangedToken(t, s, httpServer, noScope)
		require.NotContains(t, jwtPayload(t, subjectToken), "scope")
		actorConn.Claims = map[string]any{"iss": "https://cluster.example.test", "sub": "system:serviceaccount:apps:relay", "aud": "relay"}
		v := exchangeValues("relay", "", subjectToken, oauth2.TokenTypeAccess, true)
		v.Set("scope", "tools:read")
		rr := postToken(t, s, httpServer, v)
		require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
		require.Contains(t, rr.Body.String(), "subject token does not carry the requested scopes")
	})
}

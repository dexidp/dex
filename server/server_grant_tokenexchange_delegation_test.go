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
// "actor", with client "gateway" delegating through "actor", "plain" exchanging
// without an actor, and "public" a public client listed for delegation.
func delegationServer(t *testing.T, now time.Time) (*httptest.Server, *Server, *mock.Callback, *mock.Callback) {
	t.Helper()
	ctx := t.Context()
	actor := mockConnector("actor")
	actor.GrantTypes = []string{oauth2.GrantTypeTokenExchange}
	httpServer, s := newTestServerWith(t, []storage.Connector{mockConnector("mock"), actor}, func(c *Config) {
		c.AllowedGrantTypes = []string{oauth2.GrantTypeTokenExchange, oauth2.GrantTypeRefreshToken, oauth2.GrantTypeClientCredentials}
		c.Now = func() time.Time { return now }
		c.IDTokensValidFor = time.Hour
		c.TokenExchangeDelegation = []grants.DelegationPolicy{
			{ClientID: "gateway", ActorConnectors: []string{"actor"}, Audiences: []string{"backend"}},
			{ClientID: "public", ActorConnectors: []string{"actor"}},
		}
		for _, client := range []storage.Client{
			{ID: "gateway", Secret: "secret", TrustedPeers: []string{"plain"}},
			{ID: "plain", Secret: "secret"},
			{ID: "public", Secret: "secret", Public: true},
			{ID: "backend", Secret: "secret", TrustedPeers: []string{"gateway"}},
		} {
			require.NoError(t, c.Storage.CreateClient(ctx, client))
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

	tests := []struct {
		name          string
		edit          func(vals url.Values)
		subject       func(claims map[string]any)
		actor         func(claims map[string]any)
		actorInvalid  bool
		expectedCode  int
		expectedType  string
		expectedError string
	}{
		{name: "delegated access token", expectedCode: http.StatusOK},
		{name: "delegated id token", edit: func(v url.Values) { v.Set("requested_token_type", oauth2.TokenTypeID) }, expectedCode: http.StatusOK},
		{name: "id_token actor type", edit: func(v url.Values) { v.Set("actor_token_type", oauth2.TokenTypeID) }, expectedCode: http.StatusOK},
		{name: "array audience", actor: func(c map[string]any) { c["aud"] = []any{"other", "gateway"} }, expectedCode: http.StatusOK},
		{name: "may_act satisfied", subject: func(c map[string]any) {
			c["may_act"] = map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject}
		}, expectedCode: http.StatusOK},
		{name: "cross-client audience", edit: func(v url.Values) { v.Set("scope", "openid audience:server:client_id:backend") }, expectedCode: http.StatusOK},
		{name: "plain client without actor", edit: func(v url.Values) {
			v.Set("client_id", "plain")
			v.Del("actor_token")
			v.Del("actor_token_type")
			v.Del("actor_connector_id")
		}, expectedCode: http.StatusOK},

		{name: "actor_token without type", edit: func(v url.Values) { v.Del("actor_token_type") }, expectedCode: http.StatusBadRequest, expectedError: "actor_token and actor_token_type must be sent together"},
		{name: "actor_token_type without token", edit: func(v url.Values) { v.Del("actor_token") }, expectedCode: http.StatusBadRequest, expectedError: "actor_token and actor_token_type must be sent together"},
		{name: "plain client with actor", edit: func(v url.Values) { v.Set("client_id", "plain") }, expectedCode: http.StatusBadRequest, expectedError: "delegation is not enabled for this client"},
		{name: "public client", edit: func(v url.Values) { v.Set("client_id", "public") }, expectedCode: http.StatusBadRequest, expectedError: "delegation is not enabled for this client"},
		{name: "delegating client without actor", edit: func(v url.Values) {
			v.Del("actor_token")
			v.Del("actor_token_type")
		}, expectedCode: http.StatusBadRequest, expectedError: "this client must present an actor_token"},
		{name: "actor connector as subject connector", edit: func(v url.Values) { v.Set("connector_id", "actor") }, expectedCode: http.StatusBadRequest, expectedError: "connector only verifies actor tokens"},
		{name: "actor connector as subject connector for plain client", edit: func(v url.Values) {
			v.Set("client_id", "plain")
			v.Set("connector_id", "actor")
			v.Del("actor_token")
			v.Del("actor_token_type")
		}, expectedCode: http.StatusBadRequest, expectedError: "connector only verifies actor tokens"},
		{name: "access_token subject", edit: func(v url.Values) { v.Set("subject_token_type", oauth2.TokenTypeAccess) }, expectedCode: http.StatusBadRequest, expectedError: "token type not supported for delegation"},
		{name: "access_token actor", edit: func(v url.Values) { v.Set("actor_token_type", oauth2.TokenTypeAccess) }, expectedCode: http.StatusBadRequest, expectedError: "token type not supported for delegation"},
		{name: "actor connector not in policy", edit: func(v url.Values) { v.Set("actor_connector_id", "mock") }, expectedCode: http.StatusBadRequest, expectedError: "actor connector not allowed for this client"},
		{name: "actor token does not verify", actorInvalid: true, expectedCode: http.StatusBadRequest, expectedError: "invalid actor_token"},
		{name: "actor audience is another client", actor: func(c map[string]any) { c["aud"] = "other" }, expectedCode: http.StatusBadRequest, expectedError: "actor token audience does not include this client"},
		{name: "actor without audience", actor: func(c map[string]any) { delete(c, "aud") }, expectedCode: http.StatusBadRequest, expectedError: "actor token audience does not include this client"},
		{name: "subject already delegated", subject: func(c map[string]any) { c["act"] = map[string]any{"sub": "earlier"} }, expectedCode: http.StatusBadRequest, expectedError: "delegated subject tokens are not supported"},
		{name: "may_act names another actor", subject: func(c map[string]any) { c["may_act"] = map[string]any{"sub": "someone-else"} }, expectedCode: http.StatusBadRequest, expectedError: "actor is not authorized by the subject token's may_act claim"},
		{name: "subject audience is another client", subject: func(c map[string]any) { c["aud"] = []any{"other"} }, expectedCode: http.StatusBadRequest, expectedError: "subject token audience does not include this client"},
		{name: "audience outside the policy", edit: func(v url.Values) { v.Set("scope", "openid audience:server:client_id:plain") }, expectedCode: http.StatusBadRequest, expectedType: oauth2.InvalidTarget, expectedError: "audience not allowed for delegation"},
		{name: "actor_token on refresh grant", edit: func(v url.Values) { v.Set("grant_type", oauth2.GrantTypeRefreshToken) }, expectedCode: http.StatusBadRequest, expectedError: "actor_token is only valid for token exchange"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpServer, s, subjectConn, actorConn := delegationServer(t, now)
			defer httpServer.Close()

			subjectConn.Claims = subjectClaims()
			if tc.subject != nil {
				tc.subject(subjectConn.Claims)
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
				require.Equal(t, float64(now.Add(time.Hour).Unix()), claims["exp"], "no actor: default lifetime")
				return
			}
			require.Equal(t, map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject}, claims["act"])
			require.Equal(t, float64(now.Add(10*time.Minute).Unix()), claims["exp"], "bounded by the actor token")
			if strings.Contains(vals.Get("scope"), "backend") {
				require.ElementsMatch(t, []any{"backend", "gateway"}, claims["aud"])
			} else {
				require.Equal(t, "gateway", claims["aud"], "the actor adds no audience")
			}
			require.NotContains(t, claims, "may_act")
		})
	}
}

func TestHandleTokenExchangeDelegationBoundedBySubject(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	httpServer, s, subjectConn, actorConn := delegationServer(t, now)
	defer httpServer.Close()
	subjectConn.Claims = map[string]any{"sub": "person-1", "aud": "gateway", "exp": float64(now.Add(5 * time.Minute).Unix())}
	actorConn.Claims = map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject, "aud": "gateway", "exp": float64(now.Add(10 * time.Minute).Unix())}

	vals := make(url.Values)
	vals.Set("grant_type", oauth2.GrantTypeTokenExchange)
	vals.Set("client_id", "gateway")
	vals.Set("client_secret", "secret")
	vals.Set("connector_id", "mock")
	vals.Set("subject_token", "subject")
	vals.Set("subject_token_type", oauth2.TokenTypeID)
	vals.Set("actor_connector_id", "actor")
	vals.Set("actor_token", "actor")
	vals.Set("actor_token_type", oauth2.TokenTypeJWT)
	rr := postToken(t, s, httpServer, vals)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var res tokens.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
	require.Equal(t, float64(now.Add(5*time.Minute).Unix()), jwtPayload(t, res.AccessToken)["exp"])
}

func TestHandleTokenExchangeDelegationIntrospection(t *testing.T) {
	now := time.Now()
	httpServer, s, subjectConn, actorConn := delegationServer(t, now)
	defer httpServer.Close()
	subjectConn.Claims = map[string]any{"sub": "person-1", "aud": "gateway"}
	actorConn.Claims = map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject, "aud": "gateway"}

	vals := make(url.Values)
	vals.Set("grant_type", oauth2.GrantTypeTokenExchange)
	vals.Set("client_id", "gateway")
	vals.Set("client_secret", "secret")
	vals.Set("connector_id", "mock")
	vals.Set("subject_token", "subject")
	vals.Set("subject_token_type", oauth2.TokenTypeID)
	vals.Set("actor_connector_id", "actor")
	vals.Set("actor_token", "actor")
	vals.Set("actor_token_type", oauth2.TokenTypeJWT)
	rr := postToken(t, s, httpServer, vals)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var res tokens.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))

	data := url.Values{}
	data.Set("token", res.AccessToken)
	req := httptest.NewRequest(http.MethodPost, httpServer.URL+"/token/introspect", strings.NewReader(data.Encode()))
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	ir := httptest.NewRecorder()
	s.ServeHTTP(ir, req)
	require.Equal(t, http.StatusOK, ir.Code, ir.Body.String())

	var intro map[string]any
	require.NoError(t, json.Unmarshal(ir.Body.Bytes(), &intro))
	require.Equal(t, true, intro["active"])
	require.Equal(t, map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject}, intro["act"], "act is a top-level member")
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
	httpServer, s, subjectConn, actorConn := delegationServer(t, now)
	defer httpServer.Close()
	subjectConn.Claims = map[string]any{"sub": "person-1", "aud": []any{"gateway", "plain"}}
	actorConn.Claims = map[string]any{"iss": delegationActorIssuer, "sub": delegationActorSubject, "aud": "gateway"}

	t.Run("impersonation keeps sub", func(t *testing.T) {
		first := exchangedToken(t, s, httpServer, exchangeValues("plain", "mock", "subject", oauth2.TokenTypeID, false))
		second := exchangedToken(t, s, httpServer, exchangeValues("plain", "", first, oauth2.TokenTypeAccess, false))
		a, b := jwtPayload(t, first), jwtPayload(t, second)
		require.Equal(t, a["sub"], b["sub"])
		require.Equal(t, a["email"], b["email"])
		require.Equal(t, a["groups"], b["groups"])
		require.Equal(t, a["name"], b["name"])
		require.NotContains(t, b, "act")
		require.LessOrEqual(t, b["exp"], a["exp"], "never outlives the subject token")
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

	t.Run("delegated token as subject", func(t *testing.T) {
		first := exchangedToken(t, s, httpServer, exchangeValues("gateway", "mock", "subject", oauth2.TokenTypeID, true))
		rr := postToken(t, s, httpServer, exchangeValues("gateway", "", first, oauth2.TokenTypeAccess, true))
		require.Equal(t, http.StatusBadRequest, rr.Code)
		require.Contains(t, rr.Body.String(), "delegated subject tokens are not supported")
	})

	t.Run("not a Dex token", func(t *testing.T) {
		rr := postToken(t, s, httpServer, exchangeValues("plain", "", "foobar", oauth2.TokenTypeID, false))
		require.Equal(t, http.StatusUnauthorized, rr.Code, rr.Body.String())
	})
}

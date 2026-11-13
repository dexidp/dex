package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	xoauth2 "golang.org/x/oauth2"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/session"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func mockRefreshTokenTestStorage(t *testing.T, s storage.Storage, useObsolete bool) {
	ctx := t.Context()
	c := storage.Client{
		ID:           "test",
		Secret:       "barfoo",
		RedirectURIs: []string{"foo://bar.com/", "https://auth.example.com"},
		Name:         "dex client",
		LogoURL:      "https://goo.gl/JIyzIC",
	}

	err := s.CreateClient(ctx, c)
	require.NoError(t, err)

	c1 := storage.Connector{
		ID:     "test",
		Type:   "mockCallback",
		Name:   "mockCallback",
		Config: nil,
	}

	err = s.CreateConnector(ctx, c1)
	require.NoError(t, err)

	refresh := storage.RefreshToken{
		ID:            "test",
		Token:         "bar",
		ObsoleteToken: "",
		Nonce:         "foo",
		ClientID:      "test",
		ConnectorID:   "test",
		Scopes:        []string{"openid", "email", "profile"},
		CreatedAt:     time.Now().UTC().Round(time.Millisecond),
		LastUsed:      time.Now().UTC().Round(time.Millisecond),
		Claims: storage.Claims{
			UserID:        "1",
			Username:      "jane",
			Email:         "jane.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
		ConnectorData: []byte(`{"some":"data"}`),
	}

	if useObsolete {
		refresh.Token = "testtest"
		refresh.ObsoleteToken = "bar"
	}

	err = s.CreateRefresh(ctx, refresh)
	require.NoError(t, err)

	offlineSessions := storage.OfflineSessions{
		UserID:        "1",
		ConnID:        "test",
		Refresh:       map[string]*storage.RefreshTokenRef{"test": {ID: "test", ClientID: "test"}},
		ConnectorData: nil,
	}

	err = s.CreateOfflineSessions(ctx, offlineSessions)
	require.NoError(t, err)
}

func TestRefreshTokenExpirationScenarios(t *testing.T) {
	t0 := time.Now()
	tests := []struct {
		name        string
		policy      *tokens.RefreshStrategy
		useObsolete bool
		refreshID   string // refresh token ID sent by the client; defaults to "test"
		error       string
	}{
		{
			name:   "Normal",
			policy: tokens.NewRefreshStrategy(true, 0, 0, 0, nil),
			error:  ``,
		},
		{
			name:   "Not expired because used",
			policy: tokens.NewRefreshStrategy(false, 0, time.Second*60, 0, func() time.Time { return t0.Add(time.Second * 25) }),
			error:  ``,
		},
		{
			name:   "Expired because not used",
			policy: tokens.NewRefreshStrategy(false, 0, time.Second*60, 0, func() time.Time { return t0.Add(time.Hour) }),
			error:  `{"error":"invalid_grant","error_description":"Refresh token expired."}`,
		},
		{
			name:   "Absolutely expired",
			policy: tokens.NewRefreshStrategy(true, time.Second*60, 0, 0, func() time.Time { return t0.Add(time.Hour) }),
			error:  `{"error":"invalid_grant","error_description":"Refresh token expired."}`,
		},
		{
			name:        "Obsolete tokens are allowed",
			useObsolete: true,
			policy:      tokens.NewRefreshStrategy(true, 0, 0, time.Second*30, func() time.Time { return t0.Add(time.Second * 25) }),
			error:       ``,
		},
		{
			name:        "Obsolete tokens are not allowed",
			useObsolete: true,
			policy:      tokens.NewRefreshStrategy(true, 0, 0, 0, func() time.Time { return t0.Add(time.Second * 25) }),
			error:       `{"error":"invalid_grant","error_description":"Refresh token is invalid or has already been claimed by another client."}`,
		},
		{
			name:        "Obsolete tokens are allowed but token is expired globally",
			useObsolete: true,
			policy:      tokens.NewRefreshStrategy(true, time.Second*20, 0, time.Second*30, func() time.Time { return t0.Add(time.Second * 25) }),
			error:       `{"error":"invalid_grant","error_description":"Refresh token expired."}`,
		},
		{
			name:      "Unknown token, for example replaced by a newer login",
			policy:    tokens.NewRefreshStrategy(true, 0, 0, 0, nil),
			refreshID: "replaced",
			error:     `{"error":"invalid_grant","error_description":"Refresh token is invalid or has already been claimed by another client."}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(*testing.T) {
			// Setup a dex server.
			httpServer, s := newTestServer(t, func(c *Config) {
				c.RefreshTokenPolicy = tc.policy
				c.Now = func() time.Time { return t0 }
			})
			defer httpServer.Close()

			mockRefreshTokenTestStorage(t, s.storage, tc.useObsolete)

			u, err := url.Parse(s.issuerURL.String())
			require.NoError(t, err)

			refreshID := "test"
			if tc.refreshID != "" {
				refreshID = tc.refreshID
			}
			tokenData, err := internal.Marshal(&internal.RefreshToken{RefreshId: refreshID, Token: "bar"})
			require.NoError(t, err)

			u.Path = path.Join(u.Path, "/token")
			v := url.Values{}
			v.Add("grant_type", "refresh_token")
			v.Add("refresh_token", tokenData)

			req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(v.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
			req.SetBasicAuth("test", "barfoo")

			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)

			if tc.error == "" {
				require.Equal(t, 200, rr.Code)
			} else {
				require.Equal(t, rr.Body.String(), tc.error)
				return
			}

			// Check that we received expected refresh token
			var ref struct {
				Token string `json:"refresh_token"`
			}
			err = json.Unmarshal(rr.Body.Bytes(), &ref)
			require.NoError(t, err)

			if !tc.policy.RotationEnabled() {
				require.Equal(t, tokenData, ref.Token)
			} else {
				require.NotEqual(t, tokenData, ref.Token)
			}

			if tc.useObsolete {
				updatedTokenData, err := internal.Marshal(&internal.RefreshToken{RefreshId: "test", Token: "testtest"})
				require.NoError(t, err)
				require.Equal(t, updatedTokenData, ref.Token)
			}
		})
	}
}

// decodeJWTClaims decodes the payload of a JWT token without verifying the signature.
func decodeJWTClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.SplitN(token, ".", 3)
	require.Len(t, parts, 3, "JWT should have 3 parts")

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var claims map[string]any
	err = json.Unmarshal(payload, &claims)
	require.NoError(t, err)
	return claims
}

func TestRefreshTokenAuthTime(t *testing.T) {
	t0 := time.Now().UTC().Round(time.Second)
	loginTime := t0.Add(-10 * time.Minute)

	tests := []struct {
		name               string
		sessionConfig      *session.Config
		createUserIdentity bool
		wantAuthTime       bool
		wantHTTPError      bool
	}{
		{
			name: "sessions enabled with user identity",
			sessionConfig: &session.Config{
				CookieName:       "dex_session",
				AbsoluteLifetime: 24 * time.Hour,
			},
			createUserIdentity: true,
			wantAuthTime:       true,
		},
		{
			name:               "sessions disabled",
			sessionConfig:      nil,
			createUserIdentity: false,
			wantAuthTime:       false,
		},
		{
			name: "sessions enabled but user identity missing",
			sessionConfig: &session.Config{
				CookieName:       "dex_session",
				AbsoluteLifetime: 24 * time.Hour,
			},
			createUserIdentity: false,
			wantHTTPError:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpServer, s := newTestServer(t, func(c *Config) {
				c.Now = func() time.Time { return t0 }
				c.SessionConfig = tc.sessionConfig
			})
			defer httpServer.Close()

			mockRefreshTokenTestStorage(t, s.storage, false)

			if tc.createUserIdentity {
				// UserIdentity must match the refresh token's Claims.UserID ("1")
				// because updateRefreshToken looks it up by that ID.
				err := s.storage.CreateUserIdentity(t.Context(), storage.UserIdentity{
					UserID:      "1",
					ConnectorID: "test",
					Claims: storage.Claims{
						UserID:        "1",
						Username:      "jane",
						Email:         "jane.doe@example.com",
						EmailVerified: true,
						Groups:        []string{"a", "b"},
					},
					CreatedAt: loginTime,
					LastLogin: loginTime,
				})
				require.NoError(t, err)
			}

			u, err := url.Parse(s.issuerURL.String())
			require.NoError(t, err)

			tokenData, err := internal.Marshal(&internal.RefreshToken{RefreshId: "test", Token: "bar"})
			require.NoError(t, err)

			u.Path = path.Join(u.Path, "/token")
			v := url.Values{}
			v.Add("grant_type", "refresh_token")
			v.Add("refresh_token", tokenData)

			req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(v.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
			req.SetBasicAuth("test", "barfoo")

			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)

			if tc.wantHTTPError {
				assert.Equal(t, http.StatusInternalServerError, rr.Code)
				return
			}
			require.Equal(t, http.StatusOK, rr.Code)

			var resp struct {
				AccessToken  string `json:"access_token"`
				IDToken      string `json:"id_token"`
				RefreshToken string `json:"refresh_token"`
			}
			err = json.Unmarshal(rr.Body.Bytes(), &resp)
			require.NoError(t, err)

			accessClaims := decodeJWTClaims(t, resp.AccessToken)

			if tc.wantAuthTime {
				assert.Equal(t, float64(loginTime.Unix()), accessClaims["auth_time"],
					"access token auth_time should match UserIdentity.LastLogin")
			} else {
				assert.Nil(t, accessClaims["auth_time"],
					"access token should not have auth_time when sessions are disabled")
			}

			if tc.wantAuthTime {
				idClaims := decodeJWTClaims(t, resp.IDToken)
				assert.Equal(t, float64(loginTime.Unix()), idClaims["auth_time"],
					"id token auth_time should match UserIdentity.LastLogin")
			}
		})
	}
}

// failingRefreshConnector implements connector.CallbackConnector and connector.RefreshConnector
// but always returns an error on Refresh, proving that the upstream is not contacted.
type failingRefreshConnector struct {
	identity connector.Identity
	// err is returned by Refresh. When nil, a generic upstream error is returned.
	err error
}

func (f *failingRefreshConnector) LoginURL(_ connector.Scopes, callbackURL, state string) (string, []byte, error) {
	u, _ := url.Parse(callbackURL)
	v := u.Query()
	v.Set("state", state)
	u.RawQuery = v.Encode()
	return u.String(), nil, nil
}

func (f *failingRefreshConnector) HandleCallback(_ connector.Scopes, _ []byte, _ *http.Request) (connector.Identity, error) {
	return f.identity, nil
}

func (f *failingRefreshConnector) Refresh(_ context.Context, _ connector.Scopes, _ connector.Identity) (connector.Identity, error) {
	if f.err != nil {
		return connector.Identity{}, f.err
	}
	return connector.Identity{}, errors.New("upstream: refresh token expired")
}

// failingUpdateStorage fails every refresh token update, as a storage outage
// would during rotation.
type failingUpdateStorage struct {
	storage.Storage
}

func (failingUpdateStorage) UpdateRefreshToken(context.Context, string, func(storage.RefreshToken) (storage.RefreshToken, error)) error {
	return errors.New("storage unavailable")
}

// TestRefreshUpstreamRejection verifies that an upstream refusing the refresh
// with invalid_grant is passed on to the client as invalid_grant and logged at
// info, while any other failure stays a server error logged at error.
func TestRefreshUpstreamRejection(t *testing.T) {
	rejected := &xoauth2.RetrieveError{
		ErrorCode:        "invalid_grant",
		ErrorDescription: "The refresh token is invalid or expired.",
	}

	tests := []struct {
		name        string
		err         error // returned by the connector's Refresh; nil keeps the mock connector
		failStorage bool
		wantStatus  int
		wantError   string
	}{
		{
			name:       "upstream rejects the refresh token",
			err:        fmt.Errorf("oidc: failed to get refresh token: %w", rejected),
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_grant",
		},
		{
			// The shape a connector gets when its HTTP client refreshes the token.
			name:       "upstream rejects the refresh token behind an HTTP client",
			err:        fmt.Errorf("get user: %w", &url.Error{Op: "Get", URL: "https://upstream.example.com/user", Err: rejected}),
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_grant",
		},
		{
			name:       "connector finds the user deleted upstream",
			err:        fmt.Errorf("ldap: user not found %q: %w", "jane", connector.ErrUserRevoked),
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_grant",
		},
		{
			name:       "upstream rejects the client",
			err:        fmt.Errorf("oidc: failed to get refresh token: %w", &xoauth2.RetrieveError{ErrorCode: "invalid_client"}),
			wantStatus: http.StatusInternalServerError,
			wantError:  "invalid_request",
		},
		{
			name:        "storage fails during rotation",
			failStorage: true,
			wantStatus:  http.StatusInternalServerError,
			wantError:   "invalid_request",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			httpServer, s := newTestServer(t, func(c *Config) {
				c.Logger = slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
				if tc.failStorage {
					c.Storage = failingUpdateStorage{Storage: memory.New(newLogger(t))}
				}
			})
			defer httpServer.Close()

			mockRefreshTokenTestStorage(t, s.storage, false)
			if tc.err != nil {
				s.connectors.Set("test", connectors.Connector{
					Connector: &failingRefreshConnector{err: tc.err},
				})
			}

			tokenData, err := internal.Marshal(&internal.RefreshToken{RefreshId: "test", Token: "bar"})
			require.NoError(t, err)

			rr := doRefreshGrant(t, s, tokenData, 0)
			require.Equal(t, tc.wantStatus, rr.Code, "body: %s", rr.Body.String())

			var resp struct {
				Error string `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			assert.Equal(t, tc.wantError, resp.Error)

			logs := logBuf.String()
			if tc.wantStatus == http.StatusBadRequest {
				assert.Contains(t, logs, `level=INFO msg="failed to rotate refresh token"`)
				assert.NotContains(t, logs, "level=ERROR")
			} else {
				assert.Contains(t, logs, `level=ERROR msg="failed to rotate refresh token"`)
			}
		})
	}
}

func TestRefreshDisconnectsUpstreamWhenSessionsEnabled(t *testing.T) {
	t0 := time.Now().UTC().Round(time.Second)
	loginTime := t0.Add(-10 * time.Minute)

	tests := []struct {
		name               string
		sessionsEnabled    bool
		createUserIdentity bool
		wantOK             bool
	}{
		{
			name:               "sessions enabled - uses user identity, skips upstream",
			sessionsEnabled:    true,
			createUserIdentity: true,
			wantOK:             true,
		},
		{
			name:               "sessions enabled without user identity - fails",
			sessionsEnabled:    true,
			createUserIdentity: false,
			wantOK:             false,
		},
		{
			name:               "sessions disabled - upstream failure returns error",
			sessionsEnabled:    false,
			createUserIdentity: false,
			wantOK:             false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpServer, s := newTestServer(t, func(c *Config) {
				c.Now = func() time.Time { return t0 }
				if tc.sessionsEnabled {
					c.SessionConfig = &session.Config{
						CookieName:       "dex_session",
						AbsoluteLifetime: 24 * time.Hour,
					}
				}
			})
			defer httpServer.Close()

			mockRefreshTokenTestStorage(t, s.storage, false)

			// Replace the connector with one that always fails on Refresh.
			// When sessions are enabled this connector should never be called;
			// when sessions are disabled, the failure proves the error path works.
			s.connectors.Set("test", connectors.Connector{
				Connector: &failingRefreshConnector{
					identity: connector.Identity{
						UserID:   "0-385-28089-0",
						Username: "Kilgore Trout",
						Email:    "kilgore@kilgore.trout",
					},
				},
			})

			if tc.createUserIdentity {
				err := s.storage.CreateUserIdentity(t.Context(), storage.UserIdentity{
					UserID:      "1",
					ConnectorID: "test",
					Claims: storage.Claims{
						UserID:        "1",
						Username:      "jane",
						Email:         "jane.doe@example.com",
						EmailVerified: true,
						Groups:        []string{"a", "b"},
					},
					CreatedAt: loginTime,
					LastLogin: loginTime,
				})
				require.NoError(t, err)
			}

			u, err := url.Parse(s.issuerURL.String())
			require.NoError(t, err)

			tokenData, err := internal.Marshal(&internal.RefreshToken{RefreshId: "test", Token: "bar"})
			require.NoError(t, err)

			u.Path = path.Join(u.Path, "/token")
			v := url.Values{}
			v.Add("grant_type", "refresh_token")
			v.Add("refresh_token", tokenData)

			req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(v.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
			req.SetBasicAuth("test", "barfoo")

			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)

			if tc.wantOK {
				require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())

				var resp struct {
					IDToken string `json:"id_token"`
				}
				err = json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)

				// Verify the returned claims match UserIdentity, not the connector.
				claims := decodeJWTClaims(t, resp.IDToken)
				assert.Equal(t, "jane.doe@example.com", claims["email"])
				assert.Equal(t, "jane", claims["name"])
			} else {
				require.NotEqual(t, http.StatusOK, rr.Code,
					"expected error when sessions disabled or user identity missing")
			}
		})
	}
}

// TestHandleRefreshTokenAllowedConnectors verifies the refresh grant enforces
// the client's AllowedConnectors (mirrors TestHandleTokenExchangeAllowedConnectors
// and TestHandlePasswordAllowedConnectors — every token grant must enforce it).
func TestHandleRefreshTokenAllowedConnectors(t *testing.T) {
	tests := []struct {
		name              string
		allowedConnectors []string
		expectedCode      int
	}{
		{"connector in allowed list", []string{"test"}, http.StatusOK},
		{"connector matches non-first entry", []string{"other", "test"}, http.StatusOK},
		{"connector not in allowed list", []string{"other"}, http.StatusBadRequest},
		{"empty allowed list permits any connector", nil, http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			httpServer, s := newTestServer(t, func(c *Config) {
				c.RefreshTokenPolicy = tokens.NewRefreshStrategy(true, 0, 0, 0, nil)
			})
			defer httpServer.Close()

			mockRefreshTokenTestStorage(t, s.storage, false)
			require.NoError(t, s.storage.UpdateClient(ctx, "test", func(c storage.Client) (storage.Client, error) {
				c.AllowedConnectors = tc.allowedConnectors
				return c, nil
			}))

			tokenData, err := internal.Marshal(&internal.RefreshToken{RefreshId: "test", Token: "bar"})
			require.NoError(t, err)

			u, err := url.Parse(s.issuerURL.String())
			require.NoError(t, err)
			u.Path = path.Join(u.Path, "/token")

			v := url.Values{}
			v.Add("grant_type", "refresh_token")
			v.Add("refresh_token", tokenData)

			req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(v.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.SetBasicAuth("test", "barfoo")

			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)
			require.Equal(t, tc.expectedCode, rr.Code, rr.Body.String())
			if tc.expectedCode == http.StatusBadRequest {
				require.Contains(t, rr.Body.String(), "Connector not allowed",
					"rejection must be for the connector policy, not an unrelated reason")
			}
		})
	}
}

// TestRefreshTokenSID pins what a refresh does with the sid claim.
//
// A refresh carries the claim across unchanged, alive session or not, and never
// hands one to a token minted outside a browser flow. What the end of that session
// costs the token is the client's declaration: nothing for a standalone one,
// everything for a session-bound one.
func TestRefreshTokenSID(t *testing.T) {
	// The session's own id is the sid; nothing is derived any more.
	const sid = "sessionid"

	tests := []struct {
		name string
		// origin is the sid recorded on the offline-session reference: what flow
		// the refresh token came from.
		origin string
		// live is the id of the session that exists at refresh time, empty for none.
		live string
		// lifetime is the client's declared refresh token lifetime, empty for the
		// default (standalone).
		lifetime string
		wantSID  any
		// wantRefused expects the refresh to be turned away instead of issuing.
		wantRefused bool
	}{
		{
			name:    "browser flow, session alive",
			origin:  sid,
			live:    sid,
			wantSID: sid,
		},
		{
			// The sid says where the token came from, not that the session is still
			// there. Dropping it made the token active again at the next refresh.
			name:    "browser flow, session ended",
			origin:  sid,
			live:    "",
			wantSID: sid,
		},
		{
			name:    "browser flow, user signed in again",
			origin:  sid,
			live:    "anothersession",
			wantSID: sid,
		},
		{
			name:    "token from a flow with no browser",
			origin:  "",
			live:    sid,
			wantSID: nil,
		},
		{
			name:     "session-bound client, session alive",
			origin:   sid,
			live:     sid,
			lifetime: storage.RefreshTokenLifetimeSession,
			wantSID:  sid,
		},
		{
			// The whole point of the setting.
			name:        "session-bound client, session ended",
			origin:      sid,
			live:        "",
			lifetime:    storage.RefreshTokenLifetimeSession,
			wantRefused: true,
		},
		{
			name:        "session-bound client, user signed in again",
			origin:      sid,
			live:        "anothersession",
			lifetime:    storage.RefreshTokenLifetimeSession,
			wantRefused: true,
		},
		{
			// Nothing bound it to a session in the first place, so there is no
			// session for it to have outlived.
			name:     "session-bound client, token from a flow with no browser",
			origin:   "",
			live:     sid,
			lifetime: storage.RefreshTokenLifetimeSession,
			wantSID:  nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpServer, s := newTestServerWithSessions(t, nil)
			defer httpServer.Close()

			ctx := t.Context()
			mockRefreshTokenTestStorage(t, s.storage, false)

			require.NoError(t, s.storage.UpdateOfflineSessions(ctx, "1", "test",
				func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
					old.Refresh["test"].SessionID = tc.origin
					return old, nil
				}))

			if tc.lifetime != "" {
				require.NoError(t, s.storage.UpdateClient(ctx, "test",
					func(old storage.Client) (storage.Client, error) {
						old.RefreshTokenLifetime = tc.lifetime
						return old, nil
					}))
			}

			require.NoError(t, s.storage.CreateUserIdentity(ctx, storage.UserIdentity{
				UserID: "1", ConnectorID: "test",
				Claims:    storage.Claims{UserID: "1", Username: "jane"},
				LastLogin: time.Now(),
			}))

			if tc.live != "" {
				require.NoError(t, s.storage.CreateAuthSession(ctx, storage.AuthSession{
					ID: tc.live, Secret: testSessionSecret(tc.live),
					UserID: "1", ConnectorID: "test",
					CreatedAt: time.Now(), LastActivity: time.Now(),
				}))
			}

			tokenData, err := internal.Marshal(&internal.RefreshToken{RefreshId: "test", Token: "bar"})
			require.NoError(t, err)

			u, err := url.Parse(s.issuerURL.String())
			require.NoError(t, err)
			u.Path = path.Join(u.Path, "/token")

			v := url.Values{}
			v.Add("grant_type", "refresh_token")
			v.Add("refresh_token", tokenData)

			req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(v.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.SetBasicAuth("test", "barfoo")

			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)

			if tc.wantRefused {
				require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
				require.Contains(t, rr.Body.String(), "invalid_grant")

				// Spent, not merely refused: a token left in storage only fails for
				// as long as the check holds.
				_, err := s.storage.GetRefresh(ctx, "test")
				require.ErrorIs(t, err, storage.ErrNotFound)
				return
			}

			require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

			var resp struct {
				AccessToken string `json:"access_token"`
				IDToken     string `json:"id_token"`
			}
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

			assert.Equal(t, tc.wantSID, decodeJWTClaims(t, resp.AccessToken)["sid"])
			assert.Equal(t, tc.wantSID, decodeJWTClaims(t, resp.IDToken)["sid"])
		})
	}
}

// reentrancyGuard fails the storage reads the refresh grant can reach from
// inside an UpdateRefreshToken updater. A deadlock on SQLite or memory is a
// hang; the guard turns the same code path into an error, so a regression
// fails the test instead of timing out. Single-request use only.
type reentrancyGuard struct {
	storage.Storage
	inUpdate bool
}

func (g *reentrancyGuard) UpdateRefreshToken(ctx context.Context, id string, updater func(r storage.RefreshToken) (storage.RefreshToken, error)) error {
	g.inUpdate = true
	defer func() { g.inUpdate = false }()
	return g.Storage.UpdateRefreshToken(ctx, id, updater)
}

func (g *reentrancyGuard) GetPassword(ctx context.Context, email string) (storage.Password, error) {
	if g.inUpdate {
		return storage.Password{}, fmt.Errorf("GetPassword: re-entrant read from inside the UpdateRefreshToken updater")
	}
	return g.Storage.GetPassword(ctx, email)
}

func (g *reentrancyGuard) GetOfflineSessions(ctx context.Context, userID string, connID string) (storage.OfflineSessions, error) {
	if g.inUpdate {
		return storage.OfflineSessions{}, fmt.Errorf("GetOfflineSessions: re-entrant read from inside the UpdateRefreshToken updater")
	}
	return g.Storage.GetOfflineSessions(ctx, userID, connID)
}

// seedLocalPasswordDB opens the local password connector with one
// storage-backed password user (as gRPC CreatePassword / SCIM provisioning
// create) and a confidential client.
func seedLocalPasswordDB(t *testing.T, s *Server) {
	t.Helper()
	ctx := t.Context()

	err := s.storage.CreateClient(ctx, storage.Client{
		ID:           "test",
		Secret:       "barfoo",
		RedirectURIs: []string{"foo://bar.com/", "https://auth.example.com"},
		Name:         "dex client",
	})
	require.NoError(t, err)

	localConn := storage.Connector{ID: "local", Type: connectors.LocalConnector, Name: "Email", ResourceVersion: "1"}
	require.NoError(t, s.storage.CreateConnector(ctx, localConn))
	_, err = s.connectors.Open(localConn)
	require.NoError(t, err)

	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)
	require.NoError(t, s.storage.CreatePassword(ctx, storage.Password{
		Email:         "user@example.com",
		Username:      "user-login",
		Name:          "User Full Name",
		EmailVerified: boolPtr(false),
		UserID:        "user-id",
		Groups:        []string{"team-a"},
		Hash:          hash,
	}))
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
}

// doPasswordGrant exchanges the seeded user's credentials for a refresh token.
func doPasswordGrant(t *testing.T, s *Server) string {
	t.Helper()

	u, err := url.Parse(s.issuerURL.String())
	require.NoError(t, err)
	u.Path = path.Join(u.Path, "/token")

	v := url.Values{}
	v.Add("scope", "openid profile email offline_access")
	v.Add("grant_type", "password")
	v.Add("username", "user@example.com")
	v.Add("password", "secret")

	req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(v.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("test", "barfoo")

	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, "password grant failed: %s", rr.Body.String())

	var resp tokenResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.RefreshToken, "no refresh token in password grant response")
	return resp.RefreshToken
}

// doRefreshGrant redeems a refresh token and reports the response recorder. A
// non-zero timeout bounds the request both ways: the request context, so a
// deadlock surfaces as a context error where database/sql honors it, and a
// watchdog around the call, because a wedged single-connection pool can block
// past any context. Either way a regression fails in bounded time.
func doRefreshGrant(t *testing.T, s *Server, refreshToken string, timeout time.Duration) *httptest.ResponseRecorder {
	t.Helper()

	u, err := url.Parse(s.issuerURL.String())
	require.NoError(t, err)
	u.Path = path.Join(u.Path, "/token")

	v := url.Values{}
	v.Add("grant_type", "refresh_token")
	v.Add("refresh_token", refreshToken)

	req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(v.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("test", "barfoo")
	if timeout > 0 {
		ctx, cancel := context.WithTimeout(req.Context(), timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

	rr := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		s.ServeHTTP(rr, req)
		close(done)
	}()
	if timeout > 0 {
		select {
		case <-done:
		case <-time.After(timeout + 5*time.Second):
			t.Fatalf("refresh grant did not complete within %s: the rotation transaction is deadlocked on its own storage read", timeout)
		}
	} else {
		<-done
	}
	return rr
}

// TestRefreshGrantStorageBackedPasswordDB refreshes a token issued to a user
// created through the password store, end to end on memory storage. The guard
// fails any storage read from inside the rotation transactions, so this passes
// only when the identity is resolved between them (#5017).
func TestRefreshGrantStorageBackedPasswordDB(t *testing.T) {
	httpServer, s := newTestServer(t, func(c *Config) {
		c.PasswordConnector = "local"
		c.Storage = &reentrancyGuard{Storage: memory.New(newLogger(t))}
	})
	defer httpServer.Close()
	seedLocalPasswordDB(t, s)

	first := doPasswordGrant(t, s)

	rr := doRefreshGrant(t, s, first, 0)
	require.Equal(t, http.StatusOK, rr.Code, "refresh grant failed: %s", rr.Body.String())
	var second tokenResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &second))
	require.NotEqual(t, first, second.RefreshToken, "refresh token was not rotated")

	// The rotated token refreshes too: not a one-off surviving the transaction.
	rr = doRefreshGrant(t, s, second.RefreshToken, 0)
	require.Equal(t, http.StatusOK, rr.Code, "chained refresh failed: %s", rr.Body.String())
	var third tokenResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &third))
	require.NotEqual(t, second.RefreshToken, third.RefreshToken, "chained refresh did not rotate")
}

func TestRefreshDoesNotLogConnectorData(t *testing.T) {
	const upstreamToken = "upstream-refresh-token-must-not-be-logged"

	var logBuf bytes.Buffer
	httpServer, s := newTestServer(t, func(c *Config) {
		c.Logger = slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	})
	defer httpServer.Close()

	mockRefreshTokenTestStorage(t, s.storage, false)
	// Connectors keep upstream credentials in the connector data, for example the
	// upstream refresh token, so it must stay out of the logs at every level.
	err := s.storage.UpdateRefreshToken(t.Context(), "test", func(r storage.RefreshToken) (storage.RefreshToken, error) {
		r.ConnectorData = []byte(`{"refreshToken":"` + upstreamToken + `"}`)
		return r, nil
	})
	require.NoError(t, err)

	tokenData, err := internal.Marshal(&internal.RefreshToken{RefreshId: "test", Token: "bar"})
	require.NoError(t, err)

	v := url.Values{}
	v.Add("grant_type", "refresh_token")
	v.Add("refresh_token", tokenData)
	req := httptest.NewRequest(http.MethodPost, httpServer.URL+"/token", strings.NewReader(v.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("test", "barfoo")

	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	logs := logBuf.String()
	require.Contains(t, logs, "connector data before refresh", "the refresh debug logs were not written")
	require.Contains(t, logs, "saved connector data", "the refresh debug logs were not written")
	require.NotContains(t, logs, upstreamToken)
	// The mock connector returns its own connector data on refresh, so also check that
	// no log line carries the connector data itself.
	require.NotContains(t, logs, "connector_data=")
}

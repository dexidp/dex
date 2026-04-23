package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
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
		policy      *RefreshTokenPolicy
		useObsolete bool
		error       string
	}{
		{
			name:   "Normal",
			policy: &RefreshTokenPolicy{rotateRefreshTokens: true},
			error:  ``,
		},
		{
			name: "Not expired because used",
			policy: &RefreshTokenPolicy{
				rotateRefreshTokens: false,
				validIfNotUsedFor:   time.Second * 60,
				now:                 func() time.Time { return t0.Add(time.Second * 25) },
			},
			error: ``,
		},
		{
			name: "Expired because not used",
			policy: &RefreshTokenPolicy{
				rotateRefreshTokens: false,
				validIfNotUsedFor:   time.Second * 60,
				now:                 func() time.Time { return t0.Add(time.Hour) },
			},
			error: `{"error":"invalid_request","error_description":"Refresh token expired."}`,
		},
		{
			name: "Absolutely expired",
			policy: &RefreshTokenPolicy{
				rotateRefreshTokens: true,
				absoluteLifetime:    time.Second * 60,
				now:                 func() time.Time { return t0.Add(time.Hour) },
			},
			error: `{"error":"invalid_request","error_description":"Refresh token expired."}`,
		},
		{
			name:        "Obsolete tokens are allowed",
			useObsolete: true,
			policy: &RefreshTokenPolicy{
				rotateRefreshTokens: true,
				reuseInterval:       time.Second * 30,
				now:                 func() time.Time { return t0.Add(time.Second * 25) },
			},
			error: ``,
		},
		{
			name:        "Obsolete tokens are not allowed",
			useObsolete: true,
			policy: &RefreshTokenPolicy{
				rotateRefreshTokens: true,
				now:                 func() time.Time { return t0.Add(time.Second * 25) },
			},
			error: `{"error":"invalid_request","error_description":"Refresh token is invalid or has already been claimed by another client."}`,
		},
		{
			name:        "Obsolete tokens are allowed but token is expired globally",
			useObsolete: true,
			policy: &RefreshTokenPolicy{
				rotateRefreshTokens: true,
				reuseInterval:       time.Second * 30,
				absoluteLifetime:    time.Second * 20,
				now:                 func() time.Time { return t0.Add(time.Second * 25) },
			},
			error: `{"error":"invalid_request","error_description":"Refresh token expired."}`,
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

			if tc.policy.rotateRefreshTokens == false {
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
		sessionConfig      *SessionConfig
		createUserIdentity bool
		wantAuthTime       bool
		wantHTTPError      bool
	}{
		{
			name: "sessions enabled with user identity",
			sessionConfig: &SessionConfig{
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
			sessionConfig: &SessionConfig{
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
			})
			defer httpServer.Close()

			s.sessionConfig = tc.sessionConfig

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
	return connector.Identity{}, errors.New("upstream: refresh token expired")
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
			})
			defer httpServer.Close()

			if tc.sessionsEnabled {
				s.sessionConfig = &SessionConfig{
					CookieName:       "dex_session",
					AbsoluteLifetime: 24 * time.Hour,
				}
			}

			mockRefreshTokenTestStorage(t, s.storage, false)

			// Replace the connector with one that always fails on Refresh.
			// When sessions are enabled this connector should never be called;
			// when sessions are disabled, the failure proves the error path works.
			s.mu.Lock()
			s.connectors["test"] = Connector{
				Connector: &failingRefreshConnector{
					identity: connector.Identity{
						UserID:   "0-385-28089-0",
						Username: "Kilgore Trout",
						Email:    "kilgore@kilgore.trout",
					},
				},
			}
			s.mu.Unlock()

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

func TestRefreshTokenPolicy(t *testing.T) {
	lastTime := time.Now()
	l := slog.New(slog.DiscardHandler)

	r, err := NewRefreshTokenPolicy(l, true, "1m", "1m", "1m")
	require.NoError(t, err)

	t.Run("Allowed", func(t *testing.T) {
		r.now = func() time.Time { return lastTime }
		require.Equal(t, true, r.AllowedToReuse(lastTime))
		require.Equal(t, false, r.ExpiredBecauseUnused(lastTime))
		require.Equal(t, false, r.CompletelyExpired(lastTime))
	})

	t.Run("Expired", func(t *testing.T) {
		r.now = func() time.Time { return lastTime.Add(2 * time.Minute) }
		require.Equal(t, false, r.AllowedToReuse(lastTime))
		require.Equal(t, true, r.ExpiredBecauseUnused(lastTime))
		require.Equal(t, true, r.CompletelyExpired(lastTime))
	})
}

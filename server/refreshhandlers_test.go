package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

func mockRefreshTokenTestStorage(t *testing.T, s storage.Storage, useObsolete bool) {
	c := storage.Client{
		ID:           "test",
		Secret:       "barfoo",
		RedirectURIs: []string{"foo://bar.com/", "https://auth.example.com"},
		Name:         "dex client",
		LogoURL:      "https://goo.gl/JIyzIC",
	}

	err := s.CreateClient(c)
	require.NoError(t, err)

	c1 := storage.Connector{
		ID:     "test",
		Type:   "mockCallback",
		Name:   "mockCallback",
		Config: nil,
	}

	err = s.CreateConnector(c1)
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

	err = s.CreateRefresh(refresh)
	require.NoError(t, err)

	offlineSessions := storage.OfflineSessions{
		UserID:        "1",
		ConnID:        "test",
		Refresh:       map[string]*storage.RefreshTokenRef{"test": {ID: "test", ClientID: "test"}},
		ConnectorData: nil,
	}

	err = s.CreateOfflineSessions(offlineSessions)
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
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Setup a dex server.
			httpServer, s := newTestServer(ctx, t, func(c *Config) {
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

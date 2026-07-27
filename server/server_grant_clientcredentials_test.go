package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

func TestHandleClientCredentials(t *testing.T) {
	tests := []struct {
		name                    string
		clientID                string
		clientSecret            string
		clientCredentialsClaims *storage.ClientCredentialsClaims
		scopes                  string
		wantCode                int
		wantAccessTok           bool
		wantIDToken             bool
		wantUsername            string
		wantGroups              []string
	}{
		{
			name:          "Basic grant, no scopes",
			clientID:      "test",
			clientSecret:  "barfoo",
			scopes:        "",
			wantCode:      200,
			wantAccessTok: true,
			wantIDToken:   false,
		},
		{
			name:          "With openid scope",
			clientID:      "test",
			clientSecret:  "barfoo",
			scopes:        "openid",
			wantCode:      200,
			wantAccessTok: true,
			wantIDToken:   true,
		},
		{
			name:          "With openid and profile scope includes username",
			clientID:      "test",
			clientSecret:  "barfoo",
			scopes:        "openid profile",
			wantCode:      200,
			wantAccessTok: true,
			wantIDToken:   true,
			wantUsername:  "Test Client",
		},
		{
			name:          "With openid email profile groups",
			clientID:      "test",
			clientSecret:  "barfoo",
			scopes:        "openid email profile groups",
			wantCode:      200,
			wantAccessTok: true,
			wantIDToken:   true,
			wantUsername:  "Test Client",
		},
		{
			name:         "With groups scope and clientCredentialsClaims groups populated",
			clientID:     "test",
			clientSecret: "barfoo",
			clientCredentialsClaims: &storage.ClientCredentialsClaims{
				Groups: []string{"admin-group", "dev-group"},
			},
			scopes:        "openid groups",
			wantCode:      200,
			wantAccessTok: true,
			wantIDToken:   true,
			wantGroups:    []string{"admin-group", "dev-group"},
		},
		{
			name:          "With groups scope but no clientCredentialsClaims configured",
			clientID:      "test",
			clientSecret:  "barfoo",
			scopes:        "openid groups",
			wantCode:      200,
			wantAccessTok: true,
			wantIDToken:   true,
		},
		{
			name:         "Invalid client secret",
			clientID:     "test",
			clientSecret: "wrong",
			scopes:       "",
			wantCode:     401,
		},
		{
			name:         "Unknown client",
			clientID:     "nonexistent",
			clientSecret: "secret",
			scopes:       "",
			wantCode:     401,
		},
		{
			name:         "offline_access scope rejected",
			clientID:     "test",
			clientSecret: "barfoo",
			scopes:       "openid offline_access",
			wantCode:     400,
		},
		{
			name:         "Unrecognized scope",
			clientID:     "test",
			clientSecret: "barfoo",
			scopes:       "openid bogus",
			wantCode:     400,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()

			httpServer, s := newTestServer(t, func(c *Config) {
				c.Now = time.Now
			})
			defer httpServer.Close()

			// Create a confidential client for testing.
			err := s.storage.CreateClient(ctx, storage.Client{
				ID:                      "test",
				Secret:                  "barfoo",
				RedirectURIs:            []string{"https://example.com/callback"},
				Name:                    "Test Client",
				ClientCredentialsClaims: tc.clientCredentialsClaims,
			})
			require.NoError(t, err)

			u, err := url.Parse(s.issuerURL.String())
			require.NoError(t, err)
			u.Path = path.Join(u.Path, "/token")

			v := url.Values{}
			v.Add("grant_type", "client_credentials")
			if tc.scopes != "" {
				v.Add("scope", tc.scopes)
			}

			req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(v.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.SetBasicAuth(tc.clientID, tc.clientSecret)

			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)

			require.Equal(t, tc.wantCode, rr.Code)

			if tc.wantCode == 200 {
				var resp struct {
					AccessToken  string `json:"access_token"`
					TokenType    string `json:"token_type"`
					ExpiresIn    int    `json:"expires_in"`
					IDToken      string `json:"id_token"`
					RefreshToken string `json:"refresh_token"`
				}
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)

				if tc.wantAccessTok {
					require.NotEmpty(t, resp.AccessToken)
					require.Equal(t, "bearer", resp.TokenType)
					require.Greater(t, resp.ExpiresIn, 0)
				}
				if tc.wantIDToken {
					require.NotEmpty(t, resp.IDToken)

					// Verify the ID token claims.
					provider, err := oidc.NewProvider(ctx, httpServer.URL)
					require.NoError(t, err)
					verifier := provider.Verifier(&oidc.Config{ClientID: tc.clientID})
					idToken, err := verifier.Verify(ctx, resp.IDToken)
					require.NoError(t, err)

					// Decode the subject to verify the connector ID.
					var sub internal.IDTokenSubject
					require.NoError(t, internal.Unmarshal(idToken.Subject, &sub))
					require.Equal(t, "", sub.ConnId)
					require.Equal(t, tc.clientID, sub.UserId)

					var claims struct {
						Name              string   `json:"name"`
						PreferredUsername string   `json:"preferred_username"`
						Groups            []string `json:"groups"`
					}
					require.NoError(t, idToken.Claims(&claims))

					if tc.wantUsername != "" {
						require.Equal(t, tc.wantUsername, claims.Name)
						require.Equal(t, tc.wantUsername, claims.PreferredUsername)
					} else {
						require.Empty(t, claims.Name)
						require.Empty(t, claims.PreferredUsername)
					}

					if tc.wantGroups != nil {
						require.Equal(t, tc.wantGroups, claims.Groups)
					} else {
						require.Empty(t, claims.Groups)
					}
				} else {
					require.Empty(t, resp.IDToken)
				}
				// client_credentials must never return a refresh token.
				require.Empty(t, resp.RefreshToken)
			}
		})
	}
}

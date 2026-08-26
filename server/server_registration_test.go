package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/registration"
	"github.com/dexidp/dex/storage"
)

const testInitialAccessToken = "initial-access-token"

func registerClient(t *testing.T, s *Server, req registration.Request) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(req)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, s.issuerURL.AbsURL("/register"), bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+testInitialAccessToken)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, r)
	return rr
}

// Registration decides which grants a client may register from the server's
// resolved grant types, so every response type the server is configured with
// must be registrable. An id_token response type needs the implicit grant, which
// the server has to enable on its own from the configured response types.
func TestDynamicRegistrationMatchesConfiguredResponseTypes(t *testing.T) {
	tests := []struct {
		name          string
		responseTypes []string
		request       registration.Request
	}{
		{
			name:          "id_token only",
			responseTypes: []string{oauth2.ResponseTypeIDToken},
			request: registration.Request{
				GrantTypes:    []string{oauth2.GrantTypeImplicit},
				ResponseTypes: []string{oauth2.ResponseTypeIDToken},
			},
		},
		{
			name:          "code and id_token",
			responseTypes: []string{oauth2.ResponseTypeCode, oauth2.ResponseTypeIDToken},
			request: registration.Request{
				GrantTypes:    []string{oauth2.GrantTypeAuthorizationCode, oauth2.GrantTypeImplicit},
				ResponseTypes: []string{oauth2.ResponseTypeCode, oauth2.ResponseTypeIDToken},
			},
		},
		{
			// Configured as separate values, requested as a combination.
			name:          "hybrid code id_token",
			responseTypes: []string{oauth2.ResponseTypeCode, oauth2.ResponseTypeIDToken},
			request: registration.Request{
				GrantTypes:    []string{oauth2.GrantTypeAuthorizationCode, oauth2.GrantTypeImplicit},
				ResponseTypes: []string{oauth2.ResponseTypeCodeIDToken},
			},
		},
		{
			name:          "code only",
			responseTypes: []string{oauth2.ResponseTypeCode},
			request:       registration.Request{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, s := newTestServer(t, func(c *Config) {
				c.SupportedResponseTypes = tc.responseTypes
				c.DynamicClientRegistration = &DynamicClientRegistrationConfig{InitialAccessToken: testInitialAccessToken}
			})

			tc.request.RedirectURIs = []string{"https://client.example.com/callback"}
			rr := registerClient(t, s, tc.request)
			require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

			var resp registration.Response
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
			client, err := s.storage.GetClient(t.Context(), resp.ClientID)
			require.NoError(t, err)
			require.True(t, client.DynamicallyRegistered)
		})
	}
}

// A client registered for authorization_code only must not be able to obtain
// tokens through the device flow by registering Dex's own device callback as
// its redirect URI.
func TestDynamicClientCannotUseDeviceFlow(t *testing.T) {
	t0 := time.Now()
	now := func() time.Time { return t0 }

	setup := func(t *testing.T) (*Server, storage.Client) {
		_, s := newTestServer(t, func(c *Config) {
			c.Issuer += "/non-root-path"
			c.Now = now
			c.DynamicClientRegistration = &DynamicClientRegistrationConfig{InitialAccessToken: testInitialAccessToken}
		})

		// The test issuer is plain HTTP on loopback, which only a public
		// client may register as a redirect URI.
		rr := registerClient(t, s, registration.Request{
			RedirectURIs:            []string{s.issuerURL.AbsURL(oauth2.DeviceCallbackURI)},
			TokenEndpointAuthMethod: "none",
		})
		require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
		var resp registration.Response
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		require.Equal(t, []string{oauth2.GrantTypeAuthorizationCode}, resp.GrantTypes)

		client, err := s.storage.GetClient(t.Context(), resp.ClientID)
		require.NoError(t, err)
		return s, client
	}

	t.Run("device authorization request is refused", func(t *testing.T) {
		s, client := setup(t)

		form := url.Values{"client_id": {client.ID}, "client_secret": {client.Secret}, "scope": {"openid"}}
		r := httptest.NewRequest(http.MethodPost, s.issuerURL.AbsURL("/device/code"), strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, r)

		require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
		require.Contains(t, rr.Body.String(), oauth2.UnauthorizedClient)
	})

	t.Run("authorization request to the device callback is refused", func(t *testing.T) {
		s, client := setup(t)

		q := url.Values{
			"client_id":     {client.ID},
			"redirect_uri":  {client.RedirectURIs[0]},
			"response_type": {"code"},
			"scope":         {"openid"},
			"state":         {"XXXX-XXXX"},
		}
		r := httptest.NewRequest(http.MethodGet, s.issuerURL.AbsURL("/auth/mock")+"?"+q.Encode(), nil)
		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, r)

		require.Equal(t, http.StatusSeeOther, rr.Code, rr.Body.String())
		location, err := url.Parse(rr.Header().Get("Location"))
		require.NoError(t, err)
		require.Equal(t, oauth2.UnauthorizedClient, location.Query().Get("error"))
	})

	// The callback is what mints the tokens, so it must hold even when a code
	// and a device request for the client already exist.
	t.Run("device callback does not mint tokens", func(t *testing.T) {
		s, client := setup(t)
		ctx := t.Context()

		require.NoError(t, s.storage.CreateAuthCode(ctx, storage.AuthCode{
			ID:          "somecode",
			ClientID:    client.ID,
			RedirectURI: client.RedirectURIs[0],
			Scopes:      []string{"openid"},
			ConnectorID: "mock",
			Expiry:      now().Add(5 * time.Minute),
		}))
		require.NoError(t, s.storage.CreateDeviceRequest(ctx, storage.DeviceRequest{
			UserCode:     "XXXX-XXXX",
			DeviceCode:   "devicecode",
			ClientID:     client.ID,
			ClientSecret: client.Secret,
			Scopes:       []string{"openid"},
			Expiry:       now().Add(5 * time.Minute),
		}))
		require.NoError(t, s.storage.CreateDeviceToken(ctx, storage.DeviceToken{
			DeviceCode: "devicecode",
			Status:     oauth2.DeviceTokenPending,
			Expiry:     now().Add(5 * time.Minute),
		}))

		q := url.Values{"state": {"XXXX-XXXX"}, "code": {"somecode"}}
		r := httptest.NewRequest(http.MethodGet, s.issuerURL.AbsURL(oauth2.DeviceCallbackURI)+"?"+q.Encode(), nil)
		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, r)

		require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
		require.Contains(t, rr.Body.String(), oauth2.UnauthorizedClient)

		token, err := s.storage.GetDeviceToken(ctx, "devicecode")
		require.NoError(t, err)
		require.Equal(t, oauth2.DeviceTokenPending, token.Status)
		require.Empty(t, token.Token)

		// The authorization code was not consumed on the client's behalf.
		_, err = s.storage.GetAuthCode(ctx, "somecode")
		require.NoError(t, err)
	})
}

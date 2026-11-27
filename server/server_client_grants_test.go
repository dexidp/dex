package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/storage"
)

func TestClientGrantRestrictions(t *testing.T) {
	for _, grantType := range []string{
		oauth2.GrantTypeAuthorizationCode,
		oauth2.GrantTypeRefreshToken,
		oauth2.GrantTypePassword,
		oauth2.GrantTypeTokenExchange,
		oauth2.GrantTypeClientCredentials,
		oauth2.GrantTypeDeviceCode,
	} {
		t.Run(grantType, func(t *testing.T) {
			httpServer, server := newTestServer(t, func(configuration *Config) {
				configuration.PasswordConnector = "mock"
			})
			t.Cleanup(httpServer.Close)
			require.NoError(t, server.storage.CreateClient(t.Context(), storage.Client{
				ID: "restricted", Secret: "test-secret", GrantTypes: []string{oauth2.GrantTypeImplicit},
			}))
			require.NoError(t, server.storage.CreateDeviceToken(t.Context(), storage.DeviceToken{
				DeviceCode: "device-code", ClientID: "restricted", Status: oauth2.DeviceTokenComplete,
				Token: `{"access_token":"must-not-be-returned"}`, Expiry: time.Now().Add(time.Minute),
			}))
			form := url.Values{
				"grant_type": {grantType}, "scope": {"openid"}, "device_code": {"device-code"},
			}
			request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, httpServer.URL+"/token", strings.NewReader(form.Encode()))
			require.NoError(t, err)
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.SetBasicAuth("restricted", "test-secret")
			response, err := http.DefaultClient.Do(request)
			require.NoError(t, err)
			t.Cleanup(func() { response.Body.Close() })
			require.Equal(t, http.StatusBadRequest, response.StatusCode)
			var refusal struct {
				Error string `json:"error"`
			}
			require.NoError(t, json.NewDecoder(response.Body).Decode(&refusal))
			require.Equal(t, oauth2.UnauthorizedClient, refusal.Error)
		})
	}
}

func TestClientCredentialsGrantPolicy(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		grantTypes   []string
		serverGrants []string
		secret       string
		wantStatus   int
		wantError    string
	}{
		{name: "legacy", secret: "test-secret", wantStatus: http.StatusOK},
		{name: "empty", grantTypes: []string{}, secret: "test-secret", wantStatus: http.StatusOK},
		{name: "allowed", grantTypes: []string{oauth2.GrantTypeClientCredentials}, secret: "test-secret", wantStatus: http.StatusOK},
		{name: "restricted", grantTypes: []string{oauth2.GrantTypeAuthorizationCode}, secret: "test-secret", wantStatus: http.StatusBadRequest, wantError: oauth2.UnauthorizedClient},
		{name: "server disabled", grantTypes: []string{oauth2.GrantTypeClientCredentials}, serverGrants: []string{oauth2.GrantTypeAuthorizationCode}, secret: "test-secret", wantStatus: http.StatusBadRequest, wantError: oauth2.UnsupportedGrantType},
		{name: "authenticate first", grantTypes: []string{oauth2.GrantTypeAuthorizationCode}, secret: "wrong-secret", wantStatus: http.StatusUnauthorized, wantError: oauth2.InvalidClient},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			httpServer, server := newTestServer(t, func(configuration *Config) {
				if testCase.serverGrants != nil {
					configuration.AllowedGrantTypes = testCase.serverGrants
				}
			})
			t.Cleanup(httpServer.Close)
			require.NoError(t, server.storage.CreateClient(t.Context(), storage.Client{
				ID: "application", Secret: "test-secret", GrantTypes: testCase.grantTypes,
			}))
			request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, httpServer.URL+"/token", strings.NewReader("grant_type=client_credentials&scope=openid"))
			require.NoError(t, err)
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.SetBasicAuth("application", testCase.secret)
			response, err := http.DefaultClient.Do(request)
			require.NoError(t, err)
			t.Cleanup(func() { response.Body.Close() })
			require.Equal(t, testCase.wantStatus, response.StatusCode)
			var result struct {
				Error       string `json:"error"`
				AccessToken string `json:"access_token"`
			}
			require.NoError(t, json.NewDecoder(response.Body).Decode(&result))
			require.Equal(t, testCase.wantError, result.Error)
			if testCase.wantError == "" {
				require.NotEmpty(t, result.AccessToken)
			} else {
				require.Empty(t, result.AccessToken)
			}
		})
	}
}

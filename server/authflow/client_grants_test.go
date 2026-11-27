package authflow

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/storage"
)

func TestAuthorizationClientGrantTypes(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		responseType string
		grantTypes   []string
		refused      bool
	}{
		{name: "legacy code", responseType: "code"},
		{name: "allowed code", responseType: "code", grantTypes: []string{"authorization_code"}},
		{name: "refused code", responseType: "code", grantTypes: []string{"client_credentials"}, refused: true},
		{name: "allowed implicit", responseType: "id_token token", grantTypes: []string{"implicit"}},
		{name: "refused implicit", responseType: "id_token token", grantTypes: []string{"authorization_code"}, refused: true},
		{name: "allowed hybrid", responseType: "code id_token", grantTypes: []string{"authorization_code", "implicit"}},
		{name: "refused hybrid", responseType: "code id_token", grantTypes: []string{"authorization_code"}, refused: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			httpServer, handler := newTestHandler(t, func(configuration *testFlowConfig) {
				configuration.SupportedResponseTypes = toResponseTypeSet([]string{"code", "id_token", "token"})
				configuration.Storage = storage.WithStaticClients(configuration.Storage, []storage.Client{{
					ID: "application", RedirectURIs: []string{"https://app.example/callback"}, GrantTypes: testCase.grantTypes,
				}})
			})
			t.Cleanup(httpServer.Close)
			query := url.Values{
				"client_id": {"application"}, "redirect_uri": {"https://app.example/callback"},
				"response_type": {testCase.responseType}, "scope": {"openid"}, "nonce": {"nonce"},
			}
			_, _, err := handler.parseAuthorizationRequest(httptest.NewRequest("GET", httpServer.URL+"/auth?"+query.Encode(), nil))
			if testCase.refused {
				var refusal *redirectedAuthErr
				require.ErrorAs(t, err, &refusal)
				require.Equal(t, oauth2.UnauthorizedClient, refusal.Type)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

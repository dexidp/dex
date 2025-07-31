package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandleDiscovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpServer, server := newTestServer(ctx, t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", "/.well-known/openid-configuration", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 got %d", rr.Code)
	}

	var res discovery
	err := json.NewDecoder(rr.Result().Body).Decode(&res)
	require.NoError(t, err)
	require.Equal(t, discovery{
		Issuer:         httpServer.URL,
		Auth:           fmt.Sprintf("%s/auth", httpServer.URL),
		Token:          fmt.Sprintf("%s/token", httpServer.URL),
		Keys:           fmt.Sprintf("%s/keys", httpServer.URL),
		UserInfo:       fmt.Sprintf("%s/userinfo", httpServer.URL),
		DeviceEndpoint: fmt.Sprintf("%s/device/code", httpServer.URL),
		Introspect:     fmt.Sprintf("%s/token/introspect", httpServer.URL),
		GrantTypes: []string{
			"authorization_code",
			"refresh_token",
			"urn:ietf:params:oauth:grant-type:device_code",
			"urn:ietf:params:oauth:grant-type:token-exchange",
		},
		ResponseTypes: []string{
			"code",
		},
		Subjects: []string{
			"public",
		},
		IDTokenAlgs: []string{
			"RS256",
		},
		CodeChallengeAlgs: []string{
			"S256",
			"plain",
		},
		Scopes: []string{
			"openid",
			"email",
			"groups",
			"profile",
			"offline_access",
		},
		AuthMethods: []string{
			"client_secret_basic",
			"client_secret_post",
		},
		Claims: []string{
			"iss",
			"sub",
			"aud",
			"iat",
			"exp",
			"email",
			"email_verified",
			"locale",
			"name",
			"preferred_username",
			"at_hash",
		},
	}, res)
}

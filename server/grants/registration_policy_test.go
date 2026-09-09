package grants

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

type registrationPolicyGrant struct{ grantType string }

func (g registrationPolicyGrant) GrantType() string      { return g.grantType }
func (registrationPolicyGrant) RequiresClientAuth() bool { return true }
func (registrationPolicyGrant) ScopePolicy() ScopePolicy { return ScopePolicy{} }
func (registrationPolicyGrant) ConnectorID(context.Context, *Request, storage.Client) (string, *oauth2.Error) {
	return "", nil
}

func (registrationPolicyGrant) Authorize(context.Context, *Request, storage.Client, connectors.Connector) (Responder, error) {
	panic("authorization must not run when registration policy rejects the request")
}

func TestDynamicClientGrantAndAuthenticationMethodAreEnforced(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := memory.New(logger)
	client := storage.Client{
		ID: "dynamic", Secret: "secret", DynamicallyRegistered: true,
		GrantTypes:              []string{oauth2.GrantTypeAuthorizationCode},
		TokenEndpointAuthMethod: "client_secret_post",
	}
	require.NoError(t, store.CreateClient(t.Context(), client))

	h := &Handler{Storage: store, Logger: logger, grants: map[string]Grant{
		oauth2.GrantTypeRefreshToken: registrationPolicyGrant{grantType: oauth2.GrantTypeRefreshToken},
	}}
	form := url.Values{
		"client_id": {client.ID}, "client_secret": {client.Secret},
		"grant_type": {oauth2.GrantTypeRefreshToken},
	}
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	require.True(t, h.dispatch(rr, req, oauth2.GrantTypeRefreshToken))
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), oauth2.UnauthorizedClient)

	client.GrantTypes = []string{oauth2.GrantTypeRefreshToken}
	client.TokenEndpointAuthMethod = "client_secret_basic"
	require.NoError(t, store.UpdateClient(t.Context(), client.ID, func(storage.Client) (storage.Client, error) { return client, nil }))
	req = httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr = httptest.NewRecorder()
	require.True(t, h.dispatch(rr, req, oauth2.GrantTypeRefreshToken))
	require.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Contains(t, rr.Body.String(), oauth2.InvalidClient)
}

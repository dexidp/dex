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

// A public client has no secret to present, so it uses the "none" method
// however it sends its client ID; many OAuth libraries put it in a Basic header
// with an empty password.
func TestClientAuthMethod(t *testing.T) {
	for _, test := range []struct {
		name      string
		basicUser string
		basicPass string
		useBasic  bool
		form      url.Values
		want      string
	}{
		{name: "basic with secret", useBasic: true, basicUser: "id", basicPass: "secret", want: "client_secret_basic"},
		{name: "basic without secret", useBasic: true, basicUser: "id", want: "none"},
		{name: "form with secret", form: url.Values{"client_id": {"id"}, "client_secret": {"secret"}}, want: "client_secret_post"},
		{name: "form without secret", form: url.Values{"client_id": {"id"}}, want: "none"},
		// The header wins over the form, as it does when resolving credentials.
		{name: "basic without secret ignores form secret", useBasic: true, basicUser: "id", form: url.Values{"client_secret": {"secret"}}, want: "none"},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(test.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if test.useBasic {
				req.SetBasicAuth(test.basicUser, test.basicPass)
			}
			parsed, oerr := parseRequest(req)
			require.Nil(t, oerr)
			require.Equal(t, test.want, parsed.ClientAuthMethod)
		})
	}
}

func TestPublicDynamicClientMayIdentifyWithBasicHeader(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := memory.New(logger)
	require.NoError(t, store.CreateClient(t.Context(), storage.Client{
		ID: "public", Public: true, DynamicallyRegistered: true,
		GrantTypes:              []string{oauth2.GrantTypeAuthorizationCode},
		TokenEndpointAuthMethod: "none",
	}))
	h := &Handler{Storage: store, Logger: logger}

	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("public", "")
	parsed, oerr := parseRequest(req)
	require.Nil(t, oerr)

	rr := httptest.NewRecorder()
	client, ok := h.authenticateClient(t.Context(), rr, parsed)
	require.True(t, ok, rr.Body.String())
	require.Equal(t, "public", client.ID)
}

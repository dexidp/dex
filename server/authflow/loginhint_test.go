package authflow

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/connector/mock"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/storage"
)

// An OIDC login_hint on the authorization request is forwarded to the password
// connector's login page and prefills the username field; without a hint the
// field stays empty (and keeps focus). The hint is never stored.
func TestPasswordLoginPrefillsLoginHint(t *testing.T) {
	ctx := t.Context()
	logger := newLogger(t)
	_, s := newTestHandler(t, func(c *testFlowConfig) {
		c.Connectors = connectors.NewCache(c.Storage, func(conn storage.Connector) (connector.Connector, error) {
			if conn.Type == "mockPassword" {
				return (&mock.PasswordConfig{Username: "kilgore", Password: "trout"}).Open(conn.ID, logger)
			}
			return mock.NewCallbackConnector(logger), nil
		})
	})
	require.NoError(t, s.Storage.CreateConnector(ctx, storage.Connector{ID: "mockPw", Type: "mockPassword", Name: "Mock password", ResourceVersion: "1"}))
	require.NoError(t, s.Storage.CreateClient(ctx, storage.Client{ID: "cli", Secret: "secret", Name: "cli", RedirectURIs: []string{"https://example.com/cb"}}))

	for _, tc := range []struct{ name, hint string }{
		{"hint is forwarded and prefilled", "bob@example.com"},
		{"no hint leaves the field empty", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := url.Values{"client_id": {"cli"}, "redirect_uri": {"https://example.com/cb"}, "response_type": {"code"}, "scope": {"openid"}}
			if tc.hint != "" {
				q.Set("login_hint", tc.hint)
			}
			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/auth/mockPw?"+q.Encode(), nil))
			require.Equal(t, http.StatusFound, rr.Code)
			loc, err := url.Parse(rr.Header().Get("Location"))
			require.NoError(t, err)
			require.Equal(t, tc.hint, loc.Query().Get("login_hint"), "the hint rides on the redirect to the login page")

			rr = httptest.NewRecorder()
			s.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, loc.String(), nil))
			require.Equal(t, http.StatusOK, rr.Code)
			if tc.hint != "" {
				require.Contains(t, rr.Body.String(), `value="`+tc.hint+`"`)
			} else {
				require.Regexp(t, `name="login"[^>]*autofocus`, rr.Body.String())
			}
		})
	}
}

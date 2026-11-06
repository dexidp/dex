package server

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

func TestHandleTokenExchange(t *testing.T) {
	tests := []struct {
		name               string
		scope              string
		requestedTokenType string
		subjectTokenType   string
		subjectToken       string

		expectedCode      int
		expectedTokenType string
	}{
		{
			"id-for-acccess",
			"openid",
			oauth2.TokenTypeAccess,
			oauth2.TokenTypeID,
			"foobar",
			http.StatusOK,
			oauth2.TokenTypeAccess,
		},
		{
			"id-for-id",
			"openid",
			oauth2.TokenTypeID,
			oauth2.TokenTypeID,
			"foobar",
			http.StatusOK,
			oauth2.TokenTypeID,
		},
		{
			"id-for-default",
			"openid",
			"",
			oauth2.TokenTypeID,
			"foobar",
			http.StatusOK,
			oauth2.TokenTypeAccess,
		},
		{
			"access-for-access",
			"openid",
			oauth2.TokenTypeAccess,
			oauth2.TokenTypeAccess,
			"foobar",
			http.StatusOK,
			oauth2.TokenTypeAccess,
		},
		{
			"missing-subject_token_type",
			"openid",
			oauth2.TokenTypeAccess,
			"",
			"foobar",
			http.StatusBadRequest,
			"",
		},
		{
			"missing-subject_token",
			"openid",
			oauth2.TokenTypeAccess,
			oauth2.TokenTypeAccess,
			"",
			http.StatusBadRequest,
			"",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			httpServer, s := newTestServer(t, func(c *Config) {
				c.Storage.CreateClient(ctx, storage.Client{
					ID:     "client_1",
					Secret: "secret_1",
				})
			})
			defer httpServer.Close()
			vals := make(url.Values)
			vals.Set("grant_type", oauth2.GrantTypeTokenExchange)
			setNonEmpty(vals, "connector_id", "mock")
			setNonEmpty(vals, "scope", tc.scope)
			setNonEmpty(vals, "requested_token_type", tc.requestedTokenType)
			setNonEmpty(vals, "subject_token_type", tc.subjectTokenType)
			setNonEmpty(vals, "subject_token", tc.subjectToken)
			setNonEmpty(vals, "client_id", "client_1")
			setNonEmpty(vals, "client_secret", "secret_1")

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, httpServer.URL+"/token", strings.NewReader(vals.Encode()))
			req.Header.Set("content-type", "application/x-www-form-urlencoded")

			s.ServeHTTP(rr, req)

			require.Equal(t, tc.expectedCode, rr.Code, rr.Body.String())
			require.Equal(t, "application/json", rr.Result().Header.Get("content-type"))
			if tc.expectedCode == http.StatusOK {
				var res tokens.Response
				err := json.NewDecoder(rr.Result().Body).Decode(&res)
				require.NoError(t, err)
				require.Equal(t, tc.expectedTokenType, res.IssuedTokenType)
			}
		})
	}
}

func TestHandleTokenExchangeLogsSuccess(t *testing.T) {
	ctx := t.Context()
	var logBuf bytes.Buffer
	httpServer, s := newTestServer(t, func(c *Config) {
		c.Logger = slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
		c.Storage.CreateClient(ctx, storage.Client{
			ID:     "client_1",
			Secret: "secret_1",
		})
	})
	defer httpServer.Close()

	vals := make(url.Values)
	vals.Set("grant_type", oauth2.GrantTypeTokenExchange)
	vals.Set("connector_id", "mock")
	vals.Set("scope", "openid")
	vals.Set("requested_token_type", oauth2.TokenTypeAccess)
	vals.Set("subject_token_type", oauth2.TokenTypeID)
	vals.Set("subject_token", "foobar")
	vals.Set("client_id", "client_1")
	vals.Set("client_secret", "secret_1")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, httpServer.URL+"/token", strings.NewReader(vals.Encode()))
	req.Header.Set("content-type", "application/x-www-form-urlencoded")

	s.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var found map[string]any
	for _, line := range strings.Split(strings.TrimSpace(logBuf.String()), "\n") {
		var entry map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		if entry["msg"] == "token exchange successful" {
			found = entry
			break
		}
	}
	require.NotNil(t, found, "expected \"token exchange successful\" log line, got: %s", logBuf.String())
	require.Equal(t, "INFO", found["level"])
	require.Equal(t, "mock", found["connector_id"])
	require.Equal(t, "client_1", found["client_id"])
	require.Equal(t, "0-385-28089-0", found["user_id"])
	require.Equal(t, "Kilgore Trout", found["username"])
	require.Equal(t, "kilgore@kilgore.trout", found["email"])
	require.Equal(t, []any{"authors"}, found["groups"])
	require.Equal(t, oauth2.TokenTypeID, found["subject_token_type"])
	require.Equal(t, oauth2.TokenTypeAccess, found["requested_token_type"])
}

func TestHandleTokenExchangeConnectorGrantTypeRestriction(t *testing.T) {
	ctx := t.Context()
	httpServer, s := newTestServer(t, func(c *Config) {
		c.Storage.CreateClient(ctx, storage.Client{
			ID:     "client_1",
			Secret: "secret_1",
		})
	})
	defer httpServer.Close()

	// Restrict mock connector to authorization_code only
	err := s.storage.UpdateConnector(ctx, "mock", func(c storage.Connector) (storage.Connector, error) {
		c.GrantTypes = []string{oauth2.GrantTypeAuthorizationCode}
		return c, nil
	})
	require.NoError(t, err)
	// Clear cached connector to pick up new grant types
	s.connectors.Close("mock")

	vals := make(url.Values)
	vals.Set("grant_type", oauth2.GrantTypeTokenExchange)
	vals.Set("connector_id", "mock")
	vals.Set("scope", "openid")
	vals.Set("requested_token_type", oauth2.TokenTypeAccess)
	vals.Set("subject_token_type", oauth2.TokenTypeID)
	vals.Set("subject_token", "foobar")
	vals.Set("client_id", "client_1")
	vals.Set("client_secret", "secret_1")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, httpServer.URL+"/token", strings.NewReader(vals.Encode()))
	req.Header.Set("content-type", "application/x-www-form-urlencoded")

	s.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
}

func TestHandleTokenExchangeAllowedConnectors(t *testing.T) {
	tests := []struct {
		name              string
		allowedConnectors []string
		expectedCode      int
	}{
		{
			name:              "connector in allowed list",
			allowedConnectors: []string{"mock"},
			expectedCode:      http.StatusOK,
		},
		{
			name:              "connector matches non-first entry in allowed list",
			allowedConnectors: []string{"other", "mock"},
			expectedCode:      http.StatusOK,
		},
		{
			name:              "connector not in allowed list",
			allowedConnectors: []string{"other"},
			expectedCode:      http.StatusBadRequest,
		},
		{
			name:              "empty allowed list permits any connector",
			allowedConnectors: nil,
			expectedCode:      http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			httpServer, s := newTestServer(t, func(c *Config) {
				c.Storage.CreateClient(ctx, storage.Client{
					ID:                "client_1",
					Secret:            "secret_1",
					AllowedConnectors: tc.allowedConnectors,
				})
			})
			defer httpServer.Close()

			vals := make(url.Values)
			vals.Set("grant_type", oauth2.GrantTypeTokenExchange)
			vals.Set("connector_id", "mock")
			vals.Set("scope", "openid")
			vals.Set("requested_token_type", oauth2.TokenTypeAccess)
			vals.Set("subject_token_type", oauth2.TokenTypeID)
			vals.Set("subject_token", "foobar")
			vals.Set("client_id", "client_1")
			vals.Set("client_secret", "secret_1")

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, httpServer.URL+"/token", strings.NewReader(vals.Encode()))
			req.Header.Set("content-type", "application/x-www-form-urlencoded")

			s.ServeHTTP(rr, req)

			require.Equal(t, tc.expectedCode, rr.Code, rr.Body.String())
			if tc.expectedCode == http.StatusBadRequest {
				require.Contains(t, rr.Body.String(), "Connector not allowed",
					"rejection must be for the connector policy, not an unrelated reason")
			}
		})
	}
}

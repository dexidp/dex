package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/introspection"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

func toJSON(a interface{}) string {
	b, err := json.Marshal(a)
	if err != nil {
		return ""
	}

	return string(b)
}

func mockTestStorage(t *testing.T, s storage.Storage) {
	ctx := t.Context()
	c := storage.Client{
		ID:           "test",
		Secret:       "barfoo",
		RedirectURIs: []string{"foo://bar.com/", "https://auth.example.com"},
		Name:         "dex client",
		LogoURL:      "https://goo.gl/JIyzIC",
	}

	err := s.CreateClient(ctx, c)
	require.NoError(t, err)

	c1 := storage.Connector{
		ID:   "test",
		Type: "mockPassword",
		Name: "mockPassword",
		Config: []byte(`{
"username": "test",
"password": "test"
}`),
	}

	err = s.CreateConnector(ctx, c1)
	require.NoError(t, err)

	err = s.CreateRefresh(ctx, storage.RefreshToken{
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
	})
	require.NoError(t, err)

	err = s.CreateRefresh(ctx, storage.RefreshToken{
		ID:            "expired",
		Token:         "bar",
		ObsoleteToken: "",
		Nonce:         "foo",
		ClientID:      "test",
		ConnectorID:   "test",
		Scopes:        []string{"openid", "email", "profile"},
		CreatedAt:     time.Now().AddDate(-1, 0, 0).UTC().Round(time.Millisecond),
		LastUsed:      time.Now().AddDate(-1, 0, 0).UTC().Round(time.Millisecond),
		Claims: storage.Claims{
			UserID:        "1",
			Username:      "jane",
			Email:         "jane.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
		ConnectorData: []byte(`{"some":"data"}`),
	})
	require.NoError(t, err)

	err = s.CreateOfflineSessions(ctx, storage.OfflineSessions{
		UserID: "1",
		ConnID: "test",
		Refresh: map[string]*storage.RefreshTokenRef{
			"test":    {ID: "test", ClientID: "test"},
			"expired": {ID: "expired", ClientID: "test"},
		},
		ConnectorData: nil,
	})
	require.NoError(t, err)
}

func getIntrospectionValue(issuerURL url.URL, issuedAt time.Time, expiry time.Time, tokenUse string) *introspection.Introspection {
	trueValue := true
	return &introspection.Introspection{
		Active:    true,
		ClientID:  "test",
		Subject:   "CgExEgR0ZXN0",
		Expiry:    expiry.Unix(),
		IssuedAt:  issuedAt.Unix(),
		NotBefore: issuedAt.Unix(),
		Audience: []string{
			"test",
		},
		Issuer:    issuerURL.String(),
		TokenType: "Bearer",
		TokenUse:  tokenUse,
		Extra: introspection.IntrospectionExtra{
			Email:         "jane.doe@example.com",
			EmailVerified: &trueValue,
			Groups: []string{
				"a",
				"b",
			},
			Name: "jane",
		},
	}
}

func TestHandleIntrospect(t *testing.T) {
	t0 := time.Now()

	ctx := t.Context()

	// Setup a dex server.
	now := func() time.Time { return t0 }

	refreshTokenPolicy := tokens.NewRefreshStrategy(true, 24*time.Hour, 0, 0, now)

	httpServer, s := newTestServer(t, func(c *Config) {
		c.Issuer += "/non-root-path"
		c.RefreshTokenPolicy = refreshTokenPolicy
		c.Now = now
	})
	defer httpServer.Close()

	mockTestStorage(t, s.storage)

	activeAccessToken, expiry, err := s.issuer.SignIDToken(ctx, tokens.Authorization{
		Client: storage.Client{ID: "test"},
		Claims: storage.Claims{
			UserID:        "1",
			Username:      "jane",
			Email:         "jane.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
		Scopes:      []string{"openid", "email", "profile", "groups"},
		Nonce:       "foo",
		ConnectorID: "test",
	}, "", "")
	require.NoError(t, err)

	activeRefreshToken, err := internal.Marshal(&internal.RefreshToken{RefreshId: "test", Token: "bar"})
	require.NoError(t, err)
	expiredRefreshToken, err := internal.Marshal(&internal.RefreshToken{RefreshId: "expired", Token: "bar"})
	require.NoError(t, err)

	inactiveResponse := "{\"active\":false}\n"
	badRequestResponse := `{"error":"invalid_request","error_description":"The POST body can not be empty."}`

	tests := []struct {
		testName           string
		token              string
		tokenType          string
		response           string
		responseStatusCode int
	}{
		// No token
		{
			testName:           "No token",
			response:           badRequestResponse,
			responseStatusCode: 400,
		},
		// Access token tests
		{
			testName:           "Access Token: active",
			token:              activeAccessToken,
			response:           toJSON(getIntrospectionValue(s.issuerURL.URL, t0, expiry, "access_token")),
			responseStatusCode: 200,
		},
		{
			testName:           "Access Token: wrong",
			token:              "fake-token",
			response:           inactiveResponse,
			responseStatusCode: 200,
		},
		// Refresh token tests
		{
			testName:           "Refresh Token: active",
			token:              activeRefreshToken,
			response:           toJSON(getIntrospectionValue(s.issuerURL.URL, t0, t0.Add(refreshTokenPolicy.AbsoluteLifetime()), "refresh_token")),
			responseStatusCode: 200,
		},
		{
			testName:           "Refresh Token: expired",
			token:              expiredRefreshToken,
			response:           inactiveResponse,
			responseStatusCode: 200,
		},
		{
			testName:           "Refresh Token: active => false (wrong)",
			token:              "fake-token",
			response:           inactiveResponse,
			responseStatusCode: 200,
		},
	}

	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
			data := url.Values{}
			if tc.token != "" {
				data.Set("token", tc.token)
			}
			if tc.tokenType != "" {
				data.Set("token_type_hint", tc.tokenType)
			}

			u, err := url.Parse(s.issuerURL.String())
			if err != nil {
				t.Fatalf("Could not parse issuer URL %v", err)
			}
			u.Path = path.Join(u.Path, "token", "introspect")

			req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(data.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)

			if rr.Code != tc.responseStatusCode {
				t.Errorf("%s: Unexpected Response Type.  Expected %v got %v", tc.testName, tc.responseStatusCode, rr.Code)
			}

			result, _ := io.ReadAll(rr.Body)
			if string(result) != tc.response {
				t.Errorf("%s: Unexpected Response.  Expected %q got %q", tc.testName, tc.response, result)
			}
		})
	}
}

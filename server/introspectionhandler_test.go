package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/internal"
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
	ctx := context.Background()
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

func getIntrospectionValue(issuerURL url.URL, issuedAt time.Time, expiry time.Time, tokenUse string) *Introspection {
	trueValue := true
	return &Introspection{
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
		Extra: IntrospectionExtra{
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

func TestGetTokenFromRequestSuccess(t *testing.T) {
	t0 := time.Now()

	now := func() time.Time { return t0 }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup a dex server.
	httpServer, s := newTestServer(ctx, t, func(c *Config) {
		c.Issuer += "/non-root-path"
		c.Now = now
	})
	defer httpServer.Close()

	tests := []struct {
		testName          string
		expectedToken     string
		expectedTokenType TokenTypeEnum
	}{
		// Access Token
		{
			testName:          "Access Token",
			expectedToken:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			expectedTokenType: AccessToken,
		},
		// Refresh Token
		{
			testName:          "Refresh token",
			expectedToken:     "CgR0ZXN0EgNiYXI",
			expectedTokenType: RefreshToken,
		},
		// Unknown token
		{
			testName:          "Unknown token",
			expectedToken:     "AaAaAaA",
			expectedTokenType: RefreshToken,
		},
	}

	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
			data := url.Values{}
			data.Set("token", tc.expectedToken)
			req := httptest.NewRequest(http.MethodPost, "https://test.tech/token/introspect", bytes.NewBufferString(data.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			token, tokenType, err := s.getTokenFromRequest(req)
			if err != nil {
				t.Fatalf("Error returned: %s", err.Error())
			}

			if token != tc.expectedToken {
				t.Fatalf("Wrong token returned.  Expected %v got %v", tc.expectedToken, token)
			}

			if tokenType != tc.expectedTokenType {
				t.Fatalf("Wrong token type returned.  Expected %v got %v", tc.expectedTokenType, tokenType)
			}
		})
	}
}

func TestGetTokenFromRequestFailure(t *testing.T) {
	t0 := time.Now()

	now := func() time.Time { return t0 }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup a dex server.
	httpServer, s := newTestServer(ctx, t, func(c *Config) {
		c.Issuer += "/non-root-path"
		c.Now = now
	})
	defer httpServer.Close()

	_, _, err := s.getTokenFromRequest(httptest.NewRequest(http.MethodGet, "https://test.tech/token/introspect", nil))
	require.ErrorIs(t, err, &introspectionError{
		typ:  errInvalidRequest,
		desc: "HTTP method is \"GET\", expected \"POST\".",
		code: http.StatusBadRequest,
	})

	_, _, err = s.getTokenFromRequest(httptest.NewRequest(http.MethodPost, "https://test.tech/token/introspect", nil))
	require.ErrorIs(t, err, &introspectionError{
		typ:  errInvalidRequest,
		desc: "The POST body can not be empty.",
		code: http.StatusBadRequest,
	})

	req := httptest.NewRequest(http.MethodPost, "https://test.tech/token/introspect", strings.NewReader("token_type_hint=access_token"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	_, _, err = s.getTokenFromRequest(req)
	require.ErrorIs(t, err, &introspectionError{
		typ:  errInvalidRequest,
		desc: "The POST body doesn't contain 'token' parameter.",
		code: http.StatusBadRequest,
	})
}

func TestHandleIntrospect(t *testing.T) {
	t0 := time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup a dex server.
	now := func() time.Time { return t0 }

	refreshTokenPolicy, err := NewRefreshTokenPolicy(logger, false, "", "24h", "")
	if err != nil {
		t.Fatalf("failed to prepare rotation policy: %v", err)
	}
	refreshTokenPolicy.now = now

	httpServer, s := newTestServer(ctx, t, func(c *Config) {
		c.Issuer += "/non-root-path"
		c.RefreshTokenPolicy = refreshTokenPolicy
		c.Now = now
	})
	defer httpServer.Close()

	mockTestStorage(t, s.storage)

	activeAccessToken, expiry, err := s.newIDToken("test", storage.Claims{
		UserID:        "1",
		Username:      "jane",
		Email:         "jane.doe@example.com",
		EmailVerified: true,
		Groups:        []string{"a", "b"},
	}, []string{"openid", "email", "profile", "groups"}, "foo", "", "", "test")
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
			response:           toJSON(getIntrospectionValue(s.issuerURL, time.Now(), expiry, "access_token")),
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
			response:           toJSON(getIntrospectionValue(s.issuerURL, time.Now(), time.Now().Add(s.refreshTokenPolicy.absoluteLifetime), "refresh_token")),
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

func TestIntrospectErrHelper(t *testing.T) {
	t0 := time.Now()

	now := func() time.Time { return t0 }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup a dex server.
	httpServer, s := newTestServer(ctx, t, func(c *Config) {
		c.Issuer += "/non-root-path"
		c.Now = now
	})
	defer httpServer.Close()

	tests := []struct {
		testName      string
		err           *introspectionError
		resStatusCode int
		resBody       string
	}{
		{
			testName:      "Inactive Token",
			err:           newIntrospectInactiveTokenError(),
			resStatusCode: http.StatusOK,
			resBody:       "{\"active\":false}\n",
		},
		{
			testName:      "Bad Request",
			err:           newIntrospectBadRequestError("This is a bad request"),
			resStatusCode: http.StatusBadRequest,
			resBody:       `{"error":"invalid_request","error_description":"This is a bad request"}`,
		},
		{
			testName:      "Internal Server Error",
			err:           newIntrospectInternalServerError(),
			resStatusCode: http.StatusInternalServerError,
			resBody:       `{"error":"server_error"}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
			w1 := httptest.NewRecorder()

			s.introspectErrHelper(w1, tc.err.typ, tc.err.desc, tc.err.code)

			res := w1.Result()
			require.Equal(t, tc.resStatusCode, res.StatusCode)
			require.Equal(t, "application/json", res.Header.Get("Content-Type"))

			data, err := io.ReadAll(res.Body)
			defer res.Body.Close()
			require.NoError(t, err)
			require.Equal(t, tc.resBody, string(data))
		})
	}
}

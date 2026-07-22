package introspection

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

const testIssuer = "https://test.tech/non-root-path"

func testHandler(t *testing.T) *Handler {
	t.Helper()
	return &Handler{
		Issuer: testIssuer,
		Logger: slog.New(slog.DiscardHandler),
	}
}

// testAccessToken signs a valid RS256 access token for token-type guessing.
func testAccessToken(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	sig, err := signer.NewMockSigner(key)
	require.NoError(t, err)

	issURL, err := url.Parse(testIssuer)
	require.NoError(t, err)

	issuer := tokens.NewIssuer(memory.New(logger), sig, *issURL, time.Hour, time.Now, logger)
	token, _, err := issuer.SignIDToken(ctx, tokens.Authorization{
		Client:      storage.Client{ID: "test"},
		Claims:      storage.Claims{UserID: "1", Username: "jane"},
		Scopes:      []string{"openid"},
		Nonce:       "nonce",
		ConnectorID: "test",
	}, "", "")
	require.NoError(t, err)
	return token
}

func TestGetTokenFromRequestSuccess(t *testing.T) {
	h := testHandler(t)
	accessToken := testAccessToken(t)

	tests := []struct {
		testName          string
		expectedToken     string
		expectedTokenType TokenTypeEnum
	}{
		{
			testName:          "Access Token",
			expectedToken:     accessToken,
			expectedTokenType: AccessToken,
		},
		{
			testName:          "Refresh token",
			expectedToken:     "CgR0ZXN0EgNiYXI",
			expectedTokenType: RefreshToken,
		},
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
			req := httptest.NewRequest(http.MethodPost, "https://test.tech/token/introspect", strings.NewReader(data.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			token, tokenType, err := h.getTokenFromRequest(req)
			require.NoError(t, err)
			require.Equal(t, tc.expectedToken, token)
			require.Equal(t, tc.expectedTokenType, tokenType)
		})
	}
}

func TestGetTokenFromRequestFailure(t *testing.T) {
	h := testHandler(t)

	// The method is now enforced at the router level (POST only), so
	// getTokenFromRequest no longer checks it; only body validation remains.
	_, _, err := h.getTokenFromRequest(httptest.NewRequest(http.MethodPost, "https://test.tech/token/introspect", nil))
	require.ErrorIs(t, err, &introspectionError{
		typ:  oauth2.InvalidRequest,
		desc: "The POST body can not be empty.",
		code: http.StatusBadRequest,
	})

	req := httptest.NewRequest(http.MethodPost, "https://test.tech/token/introspect", strings.NewReader("token_type_hint=access_token"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	_, _, err = h.getTokenFromRequest(req)
	require.ErrorIs(t, err, &introspectionError{
		typ:  oauth2.InvalidRequest,
		desc: "The POST body doesn't contain 'token' parameter.",
		code: http.StatusBadRequest,
	})
}

func TestIntrospectErrHelper(t *testing.T) {
	h := testHandler(t)

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

			h.introspectErrHelper(w1, tc.err.typ, tc.err.desc, tc.err.code)

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

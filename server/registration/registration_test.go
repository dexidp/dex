package registration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func testHandler() *Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &Handler{
		Storage:            memory.New(logger),
		Logger:             logger,
		InitialAccessToken: "initial-access-token",
		GrantTypes: []string{
			oauth2.GrantTypeAuthorizationCode,
			oauth2.GrantTypeRefreshToken,
		},
		ResponseTypes: map[string]bool{oauth2.ResponseTypeCode: true},
		Now:           func() time.Time { return time.Unix(1_700_000_000, 0) },
	}
}

func registrationRequest(t *testing.T, h *Handler, method, auth string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var requestBody []byte
	if raw, ok := body.([]byte); ok {
		requestBody = raw
	} else {
		var err error
		requestBody, err = json.Marshal(body)
		require.NoError(t, err)
	}
	req := httptest.NewRequest(method, "https://dex.example.com/register", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rr := httptest.NewRecorder()
	h.handle(rr, req)
	return rr
}

func TestRegisterConfidentialClient(t *testing.T) {
	h := testHandler()
	rr := registrationRequest(t, h, http.MethodPost, "bearer initial-access-token", Request{
		RedirectURIs: []string{"https://client.example.com/oauth2/callback"},
		ClientName:   "Example Client",
		Scope:        "openid profile",
	})
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	require.Equal(t, "no-store", rr.Header().Get("Cache-Control"))
	require.Equal(t, "no-cache", rr.Header().Get("Pragma"))

	var response Response
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
	require.NotEmpty(t, response.ClientID)
	require.NotEmpty(t, response.ClientSecret)
	require.Equal(t, int64(1_700_000_000), response.ClientIDIssuedAt)
	require.NotNil(t, response.ClientSecretExpiresAt)
	require.Zero(t, *response.ClientSecretExpiresAt)
	// RFC 7591 defaults to authorization_code, not authorization_code plus
	// refresh_token.
	require.Equal(t, []string{oauth2.GrantTypeAuthorizationCode}, response.GrantTypes)
	require.Equal(t, []string{oauth2.ResponseTypeCode}, response.ResponseTypes)

	client, err := h.Storage.GetClient(t.Context(), response.ClientID)
	require.NoError(t, err)
	require.Equal(t, storage.Client{
		ID:                      response.ClientID,
		Secret:                  response.ClientSecret,
		RedirectURIs:            []string{"https://client.example.com/oauth2/callback"},
		Name:                    "Example Client",
		DynamicallyRegistered:   true,
		GrantTypes:              []string{oauth2.GrantTypeAuthorizationCode},
		ResponseTypes:           []string{oauth2.ResponseTypeCode},
		AllowedScopes:           []string{"openid", "profile"},
		TokenEndpointAuthMethod: "client_secret_basic",
		RegistrationTime:        1_700_000_000,
		RegistrationTokenID:     "sha256:e1f064d6f227d4de",
	}, client)
}

func TestRegisterPublicClientOmitsSecretMetadata(t *testing.T) {
	h := testHandler()
	rr := registrationRequest(t, h, http.MethodPost, "Bearer initial-access-token", Request{
		RedirectURIs:            []string{"com.example.app:/callback"},
		TokenEndpointAuthMethod: "none",
	})
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	var raw map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&raw))
	require.NotContains(t, raw, "client_secret")
	require.NotContains(t, raw, "client_secret_expires_at")
	require.Equal(t, "openid", raw["scope"])

	client, err := h.Storage.GetClient(t.Context(), raw["client_id"].(string))
	require.NoError(t, err)
	require.True(t, client.Public)
}

func TestRegisterRequiresInitialAccessToken(t *testing.T) {
	h := testHandler()
	for _, auth := range []string{"", "Basic abc", "Bearer wrong"} {
		rr := registrationRequest(t, h, http.MethodPost, auth, Request{
			RedirectURIs: []string{"https://example.com/callback"},
		})
		require.Equal(t, http.StatusUnauthorized, rr.Code)
		require.Equal(t, `Bearer error="invalid_token"`, rr.Header().Get("WWW-Authenticate"))
	}
}

func TestRegisterRejectsInvalidMetadata(t *testing.T) {
	tests := []struct {
		name      string
		request   Request
		errorType string
	}{
		{"missing redirect URI", Request{}, errorInvalidRedirectURI},
		{"relative redirect URI", Request{RedirectURIs: []string{"/callback"}}, errorInvalidRedirectURI},
		{"non-loopback HTTP redirect URI", Request{RedirectURIs: []string{"http://example.com/callback"}}, errorInvalidRedirectURI},
		{"confidential loopback HTTP redirect URI", Request{RedirectURIs: []string{"http://127.0.0.1/callback"}}, errorInvalidRedirectURI},
		{"custom scheme on confidential client", Request{RedirectURIs: []string{"com.example.app:/callback"}}, errorInvalidRedirectURI},
		{"dangerous custom scheme", Request{RedirectURIs: []string{"javascript:/callback"}, TokenEndpointAuthMethod: "none"}, errorInvalidRedirectURI},
		{"fragment in redirect URI", Request{RedirectURIs: []string{"https://example.com/callback#fragment"}}, errorInvalidRedirectURI},
		{"duplicate redirect URI", Request{RedirectURIs: []string{"https://example.com/callback", "https://example.com/callback"}}, errorInvalidRedirectURI},
		{"unsupported auth method", Request{RedirectURIs: []string{"https://example.com/callback"}, TokenEndpointAuthMethod: "private_key_jwt"}, errorInvalidClientMetadata},
		{"unsupported grant", Request{RedirectURIs: []string{"https://example.com/callback"}, GrantTypes: []string{"password"}}, errorInvalidClientMetadata},
		{"unsupported response", Request{RedirectURIs: []string{"https://example.com/callback"}, ResponseTypes: []string{"token"}}, errorInvalidClientMetadata},
		{"inconsistent grant and response", Request{RedirectURIs: []string{"https://example.com/callback"}, GrantTypes: []string{oauth2.GrantTypeRefreshToken}, ResponseTypes: []string{oauth2.ResponseTypeCode}}, errorInvalidClientMetadata},
		{"cross-origin logo", Request{RedirectURIs: []string{"https://example.com/callback"}, LogoURI: "https://attacker.example/logo.svg"}, errorInvalidClientMetadata},
		{"duplicate scope", Request{RedirectURIs: []string{"https://example.com/callback"}, Scope: "openid openid"}, errorInvalidClientMetadata},
		{"unsupported scope", Request{RedirectURIs: []string{"https://example.com/callback"}, Scope: "openid unknown"}, errorInvalidClientMetadata},
		{"offline access without refresh grant", Request{RedirectURIs: []string{"https://example.com/callback"}, Scope: "openid offline_access"}, errorInvalidClientMetadata},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rr := registrationRequest(t, testHandler(), http.MethodPost, "Bearer initial-access-token", test.request)
			require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
			var response map[string]any
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
			require.Equal(t, test.errorType, response["error"])
		})
	}
}

func TestRegisterRejectsUnsupportedDynamicGrant(t *testing.T) {
	for _, grantType := range []string{
		oauth2.GrantTypeClientCredentials,
		oauth2.GrantTypeDeviceCode,
		oauth2.GrantTypePassword,
		oauth2.GrantTypeTokenExchange,
	} {
		t.Run(grantType, func(t *testing.T) {
			h := testHandler()
			h.GrantTypes = append(h.GrantTypes, grantType)
			rr := registrationRequest(t, h, http.MethodPost, "Bearer initial-access-token", Request{
				RedirectURIs: []string{"https://example.com/callback"},
				GrantTypes:   []string{grantType},
			})
			require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())

			var response map[string]any
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
			require.Equal(t, errorInvalidClientMetadata, response["error"])
			require.Equal(t, fmt.Sprintf("grant type %q is not supported for dynamic registration", grantType), response["error_description"])
		})
	}
}

func TestRegisterRejectsNonInteractiveGrantSet(t *testing.T) {
	rr := registrationRequest(t, testHandler(), http.MethodPost, "Bearer initial-access-token", Request{
		RedirectURIs: []string{"https://example.com/callback"},
		GrantTypes:   []string{oauth2.GrantTypeRefreshToken},
	})
	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	require.Contains(t, rr.Body.String(), "dynamic registration requires an authorization_code or implicit grant")
}

func TestRegisterRequiresOpenIDScope(t *testing.T) {
	rr := registrationRequest(t, testHandler(), http.MethodPost, "Bearer initial-access-token", Request{
		RedirectURIs: []string{"https://example.com/callback"},
		Scope:        "email",
	})
	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	require.Contains(t, rr.Body.String(), "openid scope is required")
}

func TestRegisterAcceptsCompoundResponseType(t *testing.T) {
	h := testHandler()
	h.GrantTypes = append(h.GrantTypes, oauth2.GrantTypeImplicit)
	h.ResponseTypes[oauth2.ResponseTypeCodeIDToken] = true

	rr := registrationRequest(t, h, http.MethodPost, "Bearer initial-access-token", Request{
		RedirectURIs:  []string{"https://example.com/callback"},
		GrantTypes:    []string{oauth2.GrantTypeAuthorizationCode, oauth2.GrantTypeImplicit},
		ResponseTypes: []string{"code id_token"},
	})
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
}

func TestRegisterRejectsInconsistentImplicitMetadata(t *testing.T) {
	h := testHandler()
	h.GrantTypes = append(h.GrantTypes, oauth2.GrantTypeImplicit)
	h.ResponseTypes[oauth2.ResponseTypeCodeIDToken] = true

	for _, req := range []Request{
		{
			RedirectURIs:  []string{"https://example.com/callback"},
			GrantTypes:    []string{oauth2.GrantTypeAuthorizationCode},
			ResponseTypes: []string{"code id_token"},
		},
		{
			RedirectURIs:  []string{"https://example.com/callback"},
			GrantTypes:    []string{oauth2.GrantTypeAuthorizationCode, oauth2.GrantTypeImplicit},
			ResponseTypes: []string{oauth2.ResponseTypeCode},
		},
	} {
		rr := registrationRequest(t, h, http.MethodPost, "Bearer initial-access-token", req)
		require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
		require.Contains(t, rr.Body.String(), errorInvalidClientMetadata)
	}
}

func TestRegisterRequestValidation(t *testing.T) {
	h := testHandler()

	rr := registrationRequest(t, h, http.MethodGet, "Bearer initial-access-token", Request{})
	require.Equal(t, http.StatusMethodNotAllowed, rr.Code)
	require.Equal(t, http.MethodPost, rr.Header().Get("Allow"))

	rr = registrationRequest(t, h, http.MethodPost, "Bearer initial-access-token", []byte(`{} {}`))
	require.Equal(t, http.StatusBadRequest, rr.Code)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer initial-access-token")
	rr = httptest.NewRecorder()
	h.handle(rr, req)
	require.Equal(t, http.StatusUnsupportedMediaType, rr.Code)

	oversized := `{"client_name":"` + strings.Repeat("x", maxRequestBodyBytes) + `"}`
	req = httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(oversized))
	req.Header.Set("Authorization", "Bearer initial-access-token")
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.handle(rr, req)
	require.Equal(t, http.StatusRequestEntityTooLarge, rr.Code)
}

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

func TestRegisterLogsRejections(t *testing.T) {
	var logs bytes.Buffer
	h := testHandler()
	h.Logger = slog.New(slog.NewTextHandler(&logs, nil))

	rr := registrationRequest(t, h, http.MethodPost, "Bearer initial-access-token", Request{
		RedirectURIs: []string{"http://client.example.com/callback"},
		ClientName:   "client",
	})
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, logs.String(), "level=WARN")
	require.Contains(t, logs.String(), `msg="rejected dynamic client registration"`)
	require.Contains(t, logs.String(), "error=invalid_redirect_uri")
	require.Contains(t, logs.String(), "http://client.example.com/callback")
	require.Contains(t, logs.String(), "client_name=client")

	for auth, want := range map[string]string{
		"":                                   "missing",
		"initial-access-token":               `"no scheme"`,
		"Basic initial-access-token":         `"non-bearer scheme"`,
		"Bearer bearer initial-access-token": `"bearer token contains whitespace"`,
		"Bearer wrong":                       `"bearer token mismatch"`,
	} {
		logs.Reset()
		rr := registrationRequest(t, h, http.MethodPost, auth, Request{
			RedirectURIs: []string{"https://client.example.com/callback"},
		})
		require.Equal(t, http.StatusUnauthorized, rr.Code)
		require.Contains(t, logs.String(), "authorization="+want, auth)
		require.NotContains(t, logs.String(), "initial-access-token", auth)
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
	h.ResponseTypes[oauth2.ResponseTypeIDToken] = true

	rr := registrationRequest(t, h, http.MethodPost, "Bearer initial-access-token", Request{
		RedirectURIs:  []string{"https://example.com/callback"},
		GrantTypes:    []string{oauth2.GrantTypeAuthorizationCode, oauth2.GrantTypeImplicit},
		ResponseTypes: []string{"id_token code"},
	})
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	// The combination is stored in canonical order, which is how the
	// authorization endpoint compares it.
	var response Response
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
	require.Equal(t, []string{"code id_token"}, response.ResponseTypes)
}

// The authorization endpoint accepts a combination when each of its values is
// enabled, so registration must not ask for the combination to be configured
// as a response type of its own.
func TestRegisterResponseTypeCombinations(t *testing.T) {
	tests := []struct {
		name          string
		enabled       []string
		grantTypes    []string
		responseTypes []string
		status        int
		description   string
	}{
		{
			name:          "hybrid from individually enabled values",
			enabled:       []string{"code", "id_token", "token"},
			grantTypes:    []string{oauth2.GrantTypeAuthorizationCode, oauth2.GrantTypeImplicit},
			responseTypes: []string{"code id_token token"},
			status:        http.StatusCreated,
		},
		{
			name:          "combination with a disabled value",
			enabled:       []string{"code", "id_token"},
			grantTypes:    []string{oauth2.GrantTypeAuthorizationCode, oauth2.GrantTypeImplicit},
			responseTypes: []string{"code token"},
			status:        http.StatusBadRequest,
			description:   `unsupported response type "code token"`,
		},
		{
			name:          "combination enabled only as a whole",
			enabled:       []string{"code id_token"},
			grantTypes:    []string{oauth2.GrantTypeAuthorizationCode, oauth2.GrantTypeImplicit},
			responseTypes: []string{"code id_token"},
			status:        http.StatusBadRequest,
			description:   `unsupported response type "code id_token"`,
		},
		{
			// The authorization endpoint never accepts token on its own, so
			// such a client could not be used.
			name:          "token on its own",
			enabled:       []string{"code", "id_token", "token"},
			grantTypes:    []string{oauth2.GrantTypeImplicit},
			responseTypes: []string{"token"},
			status:        http.StatusBadRequest,
			description:   `response type "token" must be combined with "code" or "id_token"`,
		},
		{
			name:          "repeated value",
			enabled:       []string{"code"},
			grantTypes:    []string{oauth2.GrantTypeAuthorizationCode},
			responseTypes: []string{"code code"},
			status:        http.StatusBadRequest,
			description:   `response type "code code" repeats a value`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := testHandler()
			h.GrantTypes = append(h.GrantTypes, oauth2.GrantTypeImplicit)
			h.ResponseTypes = map[string]bool{}
			for _, responseType := range test.enabled {
				h.ResponseTypes[responseType] = true
			}

			rr := registrationRequest(t, h, http.MethodPost, "Bearer initial-access-token", Request{
				RedirectURIs:  []string{"https://example.com/callback"},
				GrantTypes:    test.grantTypes,
				ResponseTypes: test.responseTypes,
			})
			require.Equal(t, test.status, rr.Code, rr.Body.String())
			if test.description != "" {
				var response map[string]any
				require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
				require.Equal(t, errorInvalidClientMetadata, response["error"])
				require.Equal(t, test.description, response["error_description"])
			}
		})
	}
}

func TestRegisterPrivateUseSchemeRedirectURI(t *testing.T) {
	for _, test := range []struct {
		redirectURI string
		authMethod  string
		status      int
	}{
		{"com.example.app:/callback", "none", http.StatusCreated},
		{"com.example.app://oauth", "none", http.StatusCreated},
		{"myapp://callback", "none", http.StatusCreated},
		{"myapp://callback/path?x=1", "none", http.StatusCreated},
		{"myapp:callback", "none", http.StatusBadRequest},
		{"myapp://", "none", http.StatusBadRequest},
		{"javascript://callback", "none", http.StatusBadRequest},
		{"myapp://callback", "client_secret_basic", http.StatusBadRequest},
	} {
		t.Run(test.redirectURI+" "+test.authMethod, func(t *testing.T) {
			rr := registrationRequest(t, testHandler(), http.MethodPost, "Bearer initial-access-token", Request{
				RedirectURIs:            []string{test.redirectURI},
				TokenEndpointAuthMethod: test.authMethod,
			})
			require.Equal(t, test.status, rr.Code, rr.Body.String())
			if test.status != http.StatusCreated {
				require.Contains(t, rr.Body.String(), errorInvalidRedirectURI)
			}
		})
	}
}

// A token read from a secret file or environment variable commonly ends in a
// newline, which a bearer token can never carry.
func TestRegisterIgnoresWhitespaceAroundConfiguredToken(t *testing.T) {
	h := testHandler()
	h.InitialAccessToken = "initial-access-token\n"

	request := Request{RedirectURIs: []string{"https://example.com/callback"}}
	rr := registrationRequest(t, h, http.MethodPost, "Bearer initial-access-token", request)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	// The recorded token identifier does not depend on the whitespace.
	var response Response
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
	client, err := h.Storage.GetClient(t.Context(), response.ClientID)
	require.NoError(t, err)
	require.Equal(t, testHandler().initialAccessTokenID(), client.RegistrationTokenID)

	rr = registrationRequest(t, h, http.MethodPost, "Bearer wrong", request)
	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRegisterReportsWrongTypedMetadata(t *testing.T) {
	for _, test := range []struct {
		name        string
		body        string
		errorType   string
		description string
	}{
		{"scope as an array", `{"redirect_uris":["https://example.com/callback"],"scope":["openid","email"]}`, errorInvalidClientMetadata, "scope has the wrong JSON type"},
		{"grant_types as a string", `{"redirect_uris":["https://example.com/callback"],"grant_types":"authorization_code"}`, errorInvalidClientMetadata, "grant_types has the wrong JSON type"},
		{"redirect_uris as a string", `{"redirect_uris":"https://example.com/callback"}`, errorInvalidClientMetadata, "redirect_uris has the wrong JSON type"},
		{"body is not an object", `["https://example.com/callback"]`, oauth2.InvalidRequest, "invalid JSON request body"},
		{"malformed JSON", `{"redirect_uris":`, oauth2.InvalidRequest, "invalid JSON request body"},
	} {
		t.Run(test.name, func(t *testing.T) {
			rr := registrationRequest(t, testHandler(), http.MethodPost, "Bearer initial-access-token", []byte(test.body))
			require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
			var response map[string]any
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
			require.Equal(t, test.errorType, response["error"])
			require.Equal(t, test.description, response["error_description"])
		})
	}
}

func TestRegisterRejectsInconsistentImplicitMetadata(t *testing.T) {
	h := testHandler()
	h.GrantTypes = append(h.GrantTypes, oauth2.GrantTypeImplicit)
	h.ResponseTypes[oauth2.ResponseTypeIDToken] = true

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

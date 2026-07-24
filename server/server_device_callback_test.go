package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/storage"
)

func TestDeviceCallback(t *testing.T) {
	t0 := time.Now()

	now := func() time.Time { return t0 }

	type formValues struct {
		state string
		code  string
		error string
	}

	// Base "Control" test values
	baseFormValues := formValues{
		state: "XXXX-XXXX",
		code:  "somecode",
	}
	baseAuthCode := storage.AuthCode{
		ID:            "somecode",
		ClientID:      "testclient",
		RedirectURI:   oauth2.DeviceCallbackURI,
		Nonce:         "",
		Scopes:        []string{"openid", "profile", "email"},
		ConnectorID:   "mock",
		ConnectorData: nil,
		Claims:        storage.Claims{},
		Expiry:        now().Add(5 * time.Minute),
	}
	baseDeviceRequest := storage.DeviceRequest{
		UserCode:     "XXXX-XXXX",
		DeviceCode:   "devicecode",
		ClientID:     "testclient",
		ClientSecret: "",
		Scopes:       []string{"openid", "profile", "email"},
		Expiry:       now().Add(5 * time.Minute),
	}
	baseDeviceToken := storage.DeviceToken{
		DeviceCode:          "devicecode",
		Status:              oauth2.DeviceTokenPending,
		Token:               "",
		Expiry:              now().Add(5 * time.Minute),
		LastRequestTime:     time.Time{},
		PollIntervalSeconds: 0,
	}

	tests := []struct {
		testName               string
		expectedResponseCode   int
		expectedServerResponse string
		values                 formValues
		testAuthCode           storage.AuthCode
		testDeviceRequest      storage.DeviceRequest
		testDeviceToken        storage.DeviceToken
	}{
		{
			testName: "Missing State",
			values: formValues{
				state: "",
				code:  "somecode",
				error: "",
			},
			expectedResponseCode: http.StatusBadRequest,
		},
		{
			testName: "Missing Code",
			values: formValues{
				state: "XXXX-XXXX",
				code:  "",
				error: "",
			},
			expectedResponseCode: http.StatusBadRequest,
		},
		{
			testName: "Error During Authorization",
			values: formValues{
				state: "XXXX-XXXX",
				code:  "somecode",
				error: "Error Condition",
			},
			expectedResponseCode: http.StatusBadRequest,
			// Note: Error details should NOT be displayed to user anymore.
			// Instead, a safe generic message is shown.
		},
		{
			testName: "Expired Auth Code",
			values:   baseFormValues,
			testAuthCode: storage.AuthCode{
				ID:            "somecode",
				ClientID:      "testclient",
				RedirectURI:   oauth2.DeviceCallbackURI,
				Nonce:         "",
				Scopes:        []string{"openid", "profile", "email"},
				ConnectorID:   "pic",
				ConnectorData: nil,
				Claims:        storage.Claims{},
				Expiry:        now().Add(-5 * time.Minute),
			},
			expectedResponseCode: http.StatusBadRequest,
		},
		{
			testName: "Invalid Auth Code",
			values:   baseFormValues,
			testAuthCode: storage.AuthCode{
				ID:            "somecode",
				ClientID:      "testclient",
				RedirectURI:   oauth2.DeviceCallbackURI,
				Nonce:         "",
				Scopes:        []string{"openid", "profile", "email"},
				ConnectorID:   "pic",
				ConnectorData: nil,
				Claims:        storage.Claims{},
				Expiry:        now().Add(5 * time.Minute),
			},
			expectedResponseCode: http.StatusBadRequest,
		},
		{
			testName:     "Expired Device Request",
			values:       baseFormValues,
			testAuthCode: baseAuthCode,
			testDeviceRequest: storage.DeviceRequest{
				UserCode:   "XXXX-XXXX",
				DeviceCode: "devicecode",
				ClientID:   "testclient",
				Scopes:     []string{"openid", "profile", "email"},
				Expiry:     now().Add(-5 * time.Minute),
			},
			expectedResponseCode: http.StatusBadRequest,
		},
		{
			testName:     "Non-Existent User Code",
			values:       baseFormValues,
			testAuthCode: baseAuthCode,
			testDeviceRequest: storage.DeviceRequest{
				UserCode:   "ZZZZ-ZZZZ",
				DeviceCode: "devicecode",
				Scopes:     []string{"openid", "profile", "email"},
				Expiry:     now().Add(5 * time.Minute),
			},
			expectedResponseCode: http.StatusBadRequest,
		},
		{
			// The device request names a different client than the auth code was
			// minted for: the client/redirect binding must reject it (cross-client
			// auth code injection guard) before any client lookup.
			testName:     "Cross-client auth code injection",
			values:       baseFormValues,
			testAuthCode: baseAuthCode,
			testDeviceRequest: storage.DeviceRequest{
				UserCode:   "XXXX-XXXX",
				DeviceCode: "devicecode",
				ClientID:   "otherclient",
				Scopes:     []string{"openid", "profile", "email"},
				Expiry:     now().Add(5 * time.Minute),
			},
			expectedResponseCode: http.StatusBadRequest,
		},
		{
			testName:     "Bad Device Request Secret",
			values:       baseFormValues,
			testAuthCode: baseAuthCode,
			testDeviceRequest: storage.DeviceRequest{
				UserCode:     "XXXX-XXXX",
				DeviceCode:   "devicecode",
				ClientID:     "testclient",
				ClientSecret: "foobar",
				Scopes:       []string{"openid", "profile", "email"},
				Expiry:       now().Add(5 * time.Minute),
			},
			expectedResponseCode: http.StatusUnauthorized,
		},
		{
			testName:          "Expired Device Token",
			values:            baseFormValues,
			testAuthCode:      baseAuthCode,
			testDeviceRequest: baseDeviceRequest,
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "devicecode",
				Status:              oauth2.DeviceTokenPending,
				Token:               "",
				Expiry:              now().Add(-5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
			},
			expectedResponseCode: http.StatusBadRequest,
		},
		{
			testName:          "Device Code Already Redeemed",
			values:            baseFormValues,
			testAuthCode:      baseAuthCode,
			testDeviceRequest: baseDeviceRequest,
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "devicecode",
				Status:              oauth2.DeviceTokenComplete,
				Token:               "",
				Expiry:              now().Add(5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
			},
			expectedResponseCode: http.StatusBadRequest,
		},
		{
			testName:             "Successful Exchange",
			values:               baseFormValues,
			testAuthCode:         baseAuthCode,
			testDeviceRequest:    baseDeviceRequest,
			testDeviceToken:      baseDeviceToken,
			expectedResponseCode: http.StatusOK,
		},
		{
			testName: "Prevent cross-site scripting",
			values: formValues{
				state: "XXXX-XXXX",
				code:  "somecode",
				error: "<script>console.log(window);</script>",
			},
			expectedResponseCode: http.StatusBadRequest,
			// Note: XSS data should NOT be displayed to user anymore.
			// Instead, a safe generic message is shown.
		},
	}
	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
			ctx := t.Context()

			// Setup a dex server.
			httpServer, s := newTestServer(t, func(c *Config) {
				c.Issuer = c.Issuer + "/non-root-path"
				c.Now = now
			})
			defer httpServer.Close()

			if err := s.storage.CreateAuthCode(ctx, tc.testAuthCode); err != nil {
				t.Fatalf("failed to create auth code: %v", err)
			}

			if err := s.storage.CreateDeviceRequest(ctx, tc.testDeviceRequest); err != nil {
				t.Fatalf("failed to create device request: %v", err)
			}

			if err := s.storage.CreateDeviceToken(ctx, tc.testDeviceToken); err != nil {
				t.Fatalf("failed to create device token: %v", err)
			}

			client := storage.Client{
				ID:           "testclient",
				Secret:       "",
				RedirectURIs: []string{oauth2.DeviceCallbackURI},
			}
			if err := s.storage.CreateClient(ctx, client); err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			u, err := url.Parse(s.issuerURL.String())
			if err != nil {
				t.Fatalf("Could not parse issuer URL %v", err)
			}
			u.Path = path.Join(u.Path, "device/callback")
			q := u.Query()
			q.Set("state", tc.values.state)
			q.Set("code", tc.values.code)
			q.Set("error", tc.values.error)
			u.RawQuery = q.Encode()
			req, _ := http.NewRequest("GET", u.String(), nil)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)
			if rr.Code != tc.expectedResponseCode {
				t.Errorf("%s: Unexpected Response Type.  Expected %v got %v", tc.testName, tc.expectedResponseCode, rr.Code)
			}

			if len(tc.expectedServerResponse) > 0 {
				result, _ := io.ReadAll(rr.Body)
				if string(result) != tc.expectedServerResponse {
					t.Errorf("%s: Unexpected Response.  Expected %q got %q", tc.testName, tc.expectedServerResponse, result)
				}
			}

			// Special check for error message safety tests
			if tc.testName == "Prevent cross-site scripting" || tc.testName == "Error During Authorization" {
				result, _ := io.ReadAll(rr.Body)
				responseBody := string(result)

				// Error details should NOT be present in the response (for security)
				if tc.testName == "Prevent cross-site scripting" {
					if strings.Contains(responseBody, "<script>") || strings.Contains(responseBody, "console.log(window)") {
						t.Errorf("%s: XSS script found in response, but should be blocked: %q", tc.testName, responseBody)
					}
				}
				if tc.testName == "Error During Authorization" {
					if strings.Contains(responseBody, "Error Condition") {
						t.Errorf("%s: Error details found in response, but should be hidden: %q", tc.testName, responseBody)
					}
				}

				// Safe message should be present
				if !strings.Contains(responseBody, "Authorization failed. Please try again.") {
					t.Errorf("%s: Safe error message not found in response: %q", tc.testName, responseBody)
				}
			}
		})
	}
}

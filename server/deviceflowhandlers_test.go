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

	"github.com/dexidp/dex/storage"
)

func TestDeviceVerificationURI(t *testing.T) {
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

	u, err := url.Parse(s.issuerURL.String())
	if err != nil {
		t.Fatalf("Could not parse issuer URL %v", err)
	}
	u.Path = path.Join(u.Path, "/device/auth/verify_code")

	uri := s.getDeviceVerificationURI()
	if uri != u.Path {
		t.Errorf("Invalid verification URI.  Expected %v got %v", u.Path, uri)
	}
}

func TestHandleDeviceCode(t *testing.T) {
	t0 := time.Now()

	now := func() time.Time { return t0 }

	tests := []struct {
		testName               string
		clientID               string
		codeChallengeMethod    string
		requestType            string
		scopes                 []string
		expectedResponseCode   int
		expectedContentType    string
		expectedServerResponse string
	}{
		{
			testName:             "New Code",
			clientID:             "test",
			requestType:          "POST",
			scopes:               []string{"openid", "profile", "email"},
			expectedResponseCode: http.StatusOK,
			expectedContentType:  "application/json",
		},
		{
			testName:             "Invalid request Type (GET)",
			clientID:             "test",
			requestType:          "GET",
			scopes:               []string{"openid", "profile", "email"},
			expectedResponseCode: http.StatusBadRequest,
			expectedContentType:  "application/json",
		},
		{
			testName:             "New Code with valid PKCE",
			clientID:             "test",
			requestType:          "POST",
			scopes:               []string{"openid", "profile", "email"},
			codeChallengeMethod:  "S256",
			expectedResponseCode: http.StatusOK,
			expectedContentType:  "application/json",
		},
		{
			testName:             "Invalid code challenge method",
			clientID:             "test",
			requestType:          "POST",
			codeChallengeMethod:  "invalid",
			scopes:               []string{"openid", "profile", "email"},
			expectedResponseCode: http.StatusBadRequest,
			expectedContentType:  "application/json",
		},
	}
	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Setup a dex server.
			httpServer, s := newTestServer(ctx, t, func(c *Config) {
				c.Issuer += "/non-root-path"
				c.Now = now
			})
			defer httpServer.Close()

			u, err := url.Parse(s.issuerURL.String())
			if err != nil {
				t.Fatalf("Could not parse issuer URL %v", err)
			}
			u.Path = path.Join(u.Path, "device/code")

			data := url.Values{}
			data.Set("client_id", tc.clientID)
			data.Set("code_challenge_method", tc.codeChallengeMethod)
			for _, scope := range tc.scopes {
				data.Add("scope", scope)
			}
			req, _ := http.NewRequest(tc.requestType, u.String(), bytes.NewBufferString(data.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)
			if rr.Code != tc.expectedResponseCode {
				t.Errorf("Unexpected Response Type.  Expected %v got %v", tc.expectedResponseCode, rr.Code)
			}

			if rr.Header().Get("content-type") != tc.expectedContentType {
				t.Errorf("Unexpected Response Content Type.  Expected %v got %v", tc.expectedContentType, rr.Header().Get("content-type"))
			}

			body, err := io.ReadAll(rr.Body)
			if err != nil {
				t.Errorf("Could read token response %v", err)
			}
			if tc.expectedResponseCode == http.StatusOK {
				var resp deviceCodeResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Errorf("Unexpected Device Code Response Format %v", string(body))
				}
			}
		})
	}
}

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
		RedirectURI:   deviceCallbackURI,
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
		Status:              deviceTokenPending,
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
			expectedResponseCode:   http.StatusBadRequest,
			expectedServerResponse: "Error Condition: \n",
		},
		{
			testName: "Expired Auth Code",
			values:   baseFormValues,
			testAuthCode: storage.AuthCode{
				ID:            "somecode",
				ClientID:      "testclient",
				RedirectURI:   deviceCallbackURI,
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
				RedirectURI:   deviceCallbackURI,
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
			testName:     "Bad Device Request Client",
			values:       baseFormValues,
			testAuthCode: baseAuthCode,
			testDeviceRequest: storage.DeviceRequest{
				UserCode:   "XXXX-XXXX",
				DeviceCode: "devicecode",
				Scopes:     []string{"openid", "profile", "email"},
				Expiry:     now().Add(5 * time.Minute),
			},
			expectedResponseCode: http.StatusUnauthorized,
		},
		{
			testName:     "Bad Device Request Secret",
			values:       baseFormValues,
			testAuthCode: baseAuthCode,
			testDeviceRequest: storage.DeviceRequest{
				UserCode:     "XXXX-XXXX",
				DeviceCode:   "devicecode",
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
				Status:              deviceTokenPending,
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
				Status:              deviceTokenComplete,
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
			expectedResponseCode:   http.StatusBadRequest,
			expectedServerResponse: "&lt;script&gt;console.log(window);&lt;/script&gt;: \n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Setup a dex server.
			httpServer, s := newTestServer(ctx, t, func(c *Config) {
				// c.Issuer = c.Issuer + "/non-root-path"
				c.Now = now
			})
			defer httpServer.Close()

			if err := s.storage.CreateAuthCode(tc.testAuthCode); err != nil {
				t.Fatalf("failed to create auth code: %v", err)
			}

			if err := s.storage.CreateDeviceRequest(tc.testDeviceRequest); err != nil {
				t.Fatalf("failed to create device request: %v", err)
			}

			if err := s.storage.CreateDeviceToken(tc.testDeviceToken); err != nil {
				t.Fatalf("failed to create device token: %v", err)
			}

			client := storage.Client{
				ID:           "testclient",
				Secret:       "",
				RedirectURIs: []string{deviceCallbackURI},
			}
			if err := s.storage.CreateClient(client); err != nil {
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
		})
	}
}

func TestDeviceTokenResponse(t *testing.T) {
	t0 := time.Now()

	now := func() time.Time { return t0 }

	// Base PKCE values
	// base64-urlencoded, sha256 digest of code_verifier
	codeChallenge := "L7ZqsT_zNwvrH8E7J0CqPHx1wgBaFiaE-fAZcKUUAbc"
	codeChallengeMethod := "S256"
	// "random" string between 43 & 128 ASCII characters
	codeVerifier := "66114650f56cc45dee7ee03c49f048ddf9aa53cbf5b09985832fa4f790ff2604"

	baseDeviceRequest := storage.DeviceRequest{
		UserCode:   "ABCD-WXYZ",
		DeviceCode: "foo",
		ClientID:   "testclient",
		Scopes:     []string{"openid", "profile", "offline_access"},
		Expiry:     now().Add(5 * time.Minute),
	}

	tests := []struct {
		testName               string
		testDeviceRequest      storage.DeviceRequest
		testDeviceToken        storage.DeviceToken
		testGrantType          string
		testDeviceCode         string
		testCodeVerifier       string
		expectedServerResponse string
		expectedResponseCode   int
	}{
		{
			testName:          "Valid but pending token",
			testDeviceRequest: baseDeviceRequest,
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "f00bar",
				Status:              deviceTokenPending,
				Token:               "",
				Expiry:              now().Add(5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
			},
			testDeviceCode:         "f00bar",
			expectedServerResponse: deviceTokenPending,
			expectedResponseCode:   http.StatusUnauthorized,
		},
		{
			testName:          "Invalid Grant Type",
			testDeviceRequest: baseDeviceRequest,
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "f00bar",
				Status:              deviceTokenPending,
				Token:               "",
				Expiry:              now().Add(5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
			},
			testDeviceCode:         "f00bar",
			testGrantType:          grantTypeAuthorizationCode,
			expectedServerResponse: errInvalidGrant,
			expectedResponseCode:   http.StatusBadRequest,
		},
		{
			testName:          "Test Slow Down State",
			testDeviceRequest: baseDeviceRequest,
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "f00bar",
				Status:              deviceTokenPending,
				Token:               "",
				Expiry:              now().Add(5 * time.Minute),
				LastRequestTime:     now(),
				PollIntervalSeconds: 10,
			},
			testDeviceCode:         "f00bar",
			expectedServerResponse: deviceTokenSlowDown,
			expectedResponseCode:   http.StatusBadRequest,
		},
		{
			testName:          "Test Expired Device Token",
			testDeviceRequest: baseDeviceRequest,
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "f00bar",
				Status:              deviceTokenPending,
				Token:               "",
				Expiry:              now().Add(-5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
			},
			testDeviceCode:         "f00bar",
			expectedServerResponse: deviceTokenExpired,
			expectedResponseCode:   http.StatusBadRequest,
		},
		{
			testName:          "Test Nonexistent Device Code",
			testDeviceRequest: baseDeviceRequest,
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "foo",
				Status:              deviceTokenPending,
				Token:               "",
				Expiry:              now().Add(-5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
			},
			testDeviceCode:         "bar",
			expectedServerResponse: errInvalidRequest,
			expectedResponseCode:   http.StatusBadRequest,
		},
		{
			testName:          "Empty Device Code in Request",
			testDeviceRequest: baseDeviceRequest,
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "bar",
				Status:              deviceTokenPending,
				Token:               "",
				Expiry:              now().Add(-5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
			},
			testDeviceCode:         "",
			expectedServerResponse: errInvalidRequest,
			expectedResponseCode:   http.StatusBadRequest,
		},
		{
			testName:          "Claim validated token from Device Code",
			testDeviceRequest: baseDeviceRequest,
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "foo",
				Status:              deviceTokenComplete,
				Token:               "{\"access_token\": \"foobar\"}",
				Expiry:              now().Add(5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
			},
			testDeviceCode:         "foo",
			expectedServerResponse: "{\"access_token\": \"foobar\"}",
			expectedResponseCode:   http.StatusOK,
		},
		{
			testName: "Successful Exchange with PKCE",
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "foo",
				Status:              deviceTokenComplete,
				Token:               "{\"access_token\": \"foobar\"}",
				Expiry:              now().Add(5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
				PKCE: storage.PKCE{
					CodeChallenge:       codeChallenge,
					CodeChallengeMethod: codeChallengeMethod,
				},
			},
			testDeviceCode:         "foo",
			testCodeVerifier:       codeVerifier,
			testDeviceRequest:      baseDeviceRequest,
			expectedServerResponse: "{\"access_token\": \"foobar\"}",
			expectedResponseCode:   http.StatusOK,
		},
		{
			testName: "Test Exchange started with PKCE but without verifier provided",
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "foo",
				Status:              deviceTokenComplete,
				Token:               "{\"access_token\": \"foobar\"}",
				Expiry:              now().Add(5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
				PKCE: storage.PKCE{
					CodeChallenge:       codeChallenge,
					CodeChallengeMethod: codeChallengeMethod,
				},
			},
			testDeviceCode:         "foo",
			testDeviceRequest:      baseDeviceRequest,
			expectedServerResponse: errInvalidGrant,
			expectedResponseCode:   http.StatusBadRequest,
		},
		{
			testName: "Test Exchange not started with PKCE but verifier provided",
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "foo",
				Status:              deviceTokenComplete,
				Token:               "{\"access_token\": \"foobar\"}",
				Expiry:              now().Add(5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
			},
			testDeviceCode:         "foo",
			testCodeVerifier:       codeVerifier,
			testDeviceRequest:      baseDeviceRequest,
			expectedServerResponse: errInvalidRequest,
			expectedResponseCode:   http.StatusBadRequest,
		},
		{
			testName: "Test with PKCE but incorrect verifier provided",
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "foo",
				Status:              deviceTokenComplete,
				Token:               "{\"access_token\": \"foobar\"}",
				Expiry:              now().Add(5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
				PKCE: storage.PKCE{
					CodeChallenge:       codeChallenge,
					CodeChallengeMethod: codeChallengeMethod,
				},
			},
			testDeviceCode:         "foo",
			testCodeVerifier:       "invalid",
			testDeviceRequest:      baseDeviceRequest,
			expectedServerResponse: errInvalidGrant,
			expectedResponseCode:   http.StatusBadRequest,
		},
		{
			testName: "Test with PKCE but incorrect challenge provided",
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "foo",
				Status:              deviceTokenComplete,
				Token:               "{\"access_token\": \"foobar\"}",
				Expiry:              now().Add(5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
				PKCE: storage.PKCE{
					CodeChallenge:       "invalid",
					CodeChallengeMethod: codeChallengeMethod,
				},
			},
			testDeviceCode:         "foo",
			testCodeVerifier:       codeVerifier,
			testDeviceRequest:      baseDeviceRequest,
			expectedServerResponse: errInvalidGrant,
			expectedResponseCode:   http.StatusBadRequest,
		},
	}
	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Setup a dex server.
			httpServer, s := newTestServer(ctx, t, func(c *Config) {
				c.Issuer += "/non-root-path"
				c.Now = now
			})
			defer httpServer.Close()

			if err := s.storage.CreateDeviceRequest(tc.testDeviceRequest); err != nil {
				t.Fatalf("Failed to store device token %v", err)
			}

			if err := s.storage.CreateDeviceToken(tc.testDeviceToken); err != nil {
				t.Fatalf("Failed to store device token %v", err)
			}

			u, err := url.Parse(s.issuerURL.String())
			if err != nil {
				t.Fatalf("Could not parse issuer URL %v", err)
			}
			u.Path = path.Join(u.Path, "device/token")

			data := url.Values{}
			grantType := grantTypeDeviceCode
			if tc.testGrantType != "" {
				grantType = tc.testGrantType
			}
			data.Set("grant_type", grantType)
			data.Set("device_code", tc.testDeviceCode)
			if tc.testCodeVerifier != "" {
				data.Set("code_verifier", tc.testCodeVerifier)
			}
			req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(data.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)
			if rr.Code != tc.expectedResponseCode {
				t.Errorf("Unexpected Response Type.  Expected %v got %v", tc.expectedResponseCode, rr.Code)
			}

			body, err := io.ReadAll(rr.Body)
			if err != nil {
				t.Errorf("Could read token response %v", err)
			}
			if tc.expectedResponseCode == http.StatusBadRequest || tc.expectedResponseCode == http.StatusUnauthorized {
				expectJSONErrorResponse(tc.testName, body, tc.expectedServerResponse, t)
			} else if string(body) != tc.expectedServerResponse {
				t.Errorf("Unexpected Server Response.  Expected %v got %v", tc.expectedServerResponse, string(body))
			}
		})
	}
}

func expectJSONErrorResponse(testCase string, body []byte, expectedError string, t *testing.T) {
	jsonMap := make(map[string]interface{})
	err := json.Unmarshal(body, &jsonMap)
	if err != nil {
		t.Errorf("Unexpected error unmarshalling response: %v", err)
	}
	if jsonMap["error"] != expectedError {
		t.Errorf("Test Case %s expected error %v, received %v", testCase, expectedError, jsonMap["error"])
	}
}

func TestVerifyCodeResponse(t *testing.T) {
	t0 := time.Now()

	now := func() time.Time { return t0 }

	tests := []struct {
		testName             string
		testDeviceRequest    storage.DeviceRequest
		userCode             string
		expectedResponseCode int
		expectedRedirectPath string
	}{
		{
			testName: "Unknown user code",
			testDeviceRequest: storage.DeviceRequest{
				UserCode:   "ABCD-WXYZ",
				DeviceCode: "f00bar",
				ClientID:   "testclient",
				Scopes:     []string{"openid", "profile", "offline_access"},
				Expiry:     now().Add(5 * time.Minute),
			},
			userCode:             "CODE-TEST",
			expectedResponseCode: http.StatusBadRequest,
			expectedRedirectPath: "",
		},
		{
			testName: "Expired user code",
			testDeviceRequest: storage.DeviceRequest{
				UserCode:   "ABCD-WXYZ",
				DeviceCode: "f00bar",
				ClientID:   "testclient",
				Scopes:     []string{"openid", "profile", "offline_access"},
				Expiry:     now().Add(-5 * time.Minute),
			},
			userCode:             "ABCD-WXYZ",
			expectedResponseCode: http.StatusBadRequest,
			expectedRedirectPath: "",
		},
		{
			testName: "No user code",
			testDeviceRequest: storage.DeviceRequest{
				UserCode:   "ABCD-WXYZ",
				DeviceCode: "f00bar",
				ClientID:   "testclient",
				Scopes:     []string{"openid", "profile", "offline_access"},
				Expiry:     now().Add(-5 * time.Minute),
			},
			userCode:             "",
			expectedResponseCode: http.StatusBadRequest,
			expectedRedirectPath: "",
		},
		{
			testName: "Valid user code, expect redirect to auth endpoint",
			testDeviceRequest: storage.DeviceRequest{
				UserCode:   "ABCD-WXYZ",
				DeviceCode: "f00bar",
				ClientID:   "testclient",
				Scopes:     []string{"openid", "profile", "offline_access"},
				Expiry:     now().Add(5 * time.Minute),
			},
			userCode:             "ABCD-WXYZ",
			expectedResponseCode: http.StatusFound,
			expectedRedirectPath: "/auth",
		},
	}
	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Setup a dex server.
			httpServer, s := newTestServer(ctx, t, func(c *Config) {
				c.Issuer += "/non-root-path"
				c.Now = now
			})
			defer httpServer.Close()

			if err := s.storage.CreateDeviceRequest(tc.testDeviceRequest); err != nil {
				t.Fatalf("Failed to store device token %v", err)
			}

			u, err := url.Parse(s.issuerURL.String())
			if err != nil {
				t.Fatalf("Could not parse issuer URL %v", err)
			}

			u.Path = path.Join(u.Path, "device/auth/verify_code")
			data := url.Values{}
			data.Set("user_code", tc.userCode)
			req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(data.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)
			if rr.Code != tc.expectedResponseCode {
				t.Errorf("Unexpected Response Type.  Expected %v got %v", tc.expectedResponseCode, rr.Code)
			}

			u, err = url.Parse(s.issuerURL.String())
			if err != nil {
				t.Errorf("Could not parse issuer URL %v", err)
			}
			u.Path = path.Join(u.Path, tc.expectedRedirectPath)

			location := rr.Header().Get("Location")
			if rr.Code == http.StatusFound && !strings.HasPrefix(location, u.Path) {
				t.Errorf("Invalid Redirect.  Expected %v got %v", u.Path, location)
			}
		})
	}
}

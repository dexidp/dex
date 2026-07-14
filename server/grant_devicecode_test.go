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

	"github.com/dexidp/dex/storage"
)

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
			expectedResponseCode:   http.StatusBadRequest,
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
			ctx := t.Context()

			// Setup a dex server.
			httpServer, s := newTestServer(t, func(c *Config) {
				c.Issuer += "/non-root-path"
				c.Now = now
			})
			defer httpServer.Close()

			if err := s.storage.CreateDeviceRequest(ctx, tc.testDeviceRequest); err != nil {
				t.Fatalf("Failed to store device token %v", err)
			}

			if err := s.storage.CreateDeviceToken(ctx, tc.testDeviceToken); err != nil {
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
	jsonMap := make(map[string]any)
	err := json.Unmarshal(body, &jsonMap)
	if err != nil {
		t.Errorf("Unexpected error unmarshalling response: %v", err)
	}
	if jsonMap["error"] != expectedError {
		t.Errorf("Test Case %s expected error %v, received %v", testCase, expectedError, jsonMap["error"])
	}
}

package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/dexidp/dex/server/oauth2"
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

	// Older Dex versions stored the token response indented, and poll still has to decode it.
	storedTokenResponse := "{\n  \"access_token\": \"foobar\",\n  \"token_type\": \"bearer\",\n  \"expires_in\": 300,\n  \"id_token\": \"idtoken\"\n}"

	tests := []struct {
		testName               string
		testDeviceRequest      storage.DeviceRequest
		testDeviceToken        storage.DeviceToken
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
				Status:              oauth2.DeviceTokenPending,
				Token:               "",
				Expiry:              now().Add(5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
			},
			testDeviceCode:         "f00bar",
			expectedServerResponse: oauth2.DeviceTokenPending,
			expectedResponseCode:   http.StatusBadRequest,
		},
		{
			testName:          "Test Slow Down State",
			testDeviceRequest: baseDeviceRequest,
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "f00bar",
				Status:              oauth2.DeviceTokenPending,
				Token:               "",
				Expiry:              now().Add(5 * time.Minute),
				LastRequestTime:     now(),
				PollIntervalSeconds: 10,
			},
			testDeviceCode:         "f00bar",
			expectedServerResponse: oauth2.DeviceTokenSlowDown,
			expectedResponseCode:   http.StatusBadRequest,
		},
		{
			testName:          "Test Expired Device Token",
			testDeviceRequest: baseDeviceRequest,
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "f00bar",
				Status:              oauth2.DeviceTokenPending,
				Token:               "",
				Expiry:              now().Add(-5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
			},
			testDeviceCode:         "f00bar",
			expectedServerResponse: oauth2.DeviceTokenExpired,
			expectedResponseCode:   http.StatusBadRequest,
		},
		{
			testName:          "Test Nonexistent Device Code",
			testDeviceRequest: baseDeviceRequest,
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "foo",
				Status:              oauth2.DeviceTokenPending,
				Token:               "",
				Expiry:              now().Add(-5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
			},
			testDeviceCode:         "bar",
			expectedServerResponse: oauth2.InvalidGrant,
			expectedResponseCode:   http.StatusBadRequest,
		},
		{
			testName:          "Empty Device Code in Request",
			testDeviceRequest: baseDeviceRequest,
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "bar",
				Status:              oauth2.DeviceTokenPending,
				Token:               "",
				Expiry:              now().Add(-5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
			},
			testDeviceCode:         "",
			expectedServerResponse: oauth2.InvalidRequest,
			expectedResponseCode:   http.StatusBadRequest,
		},
		{
			testName:          "Claim validated token from Device Code",
			testDeviceRequest: baseDeviceRequest,
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "foo",
				Status:              oauth2.DeviceTokenComplete,
				Token:               storedTokenResponse,
				Expiry:              now().Add(5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
			},
			testDeviceCode:         "foo",
			expectedServerResponse: `{"access_token":"foobar","token_type":"bearer","expires_in":300,"id_token":"idtoken"}`,
			expectedResponseCode:   http.StatusOK,
		},
		{
			testName: "Successful Exchange with PKCE",
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "foo",
				Status:              oauth2.DeviceTokenComplete,
				Token:               storedTokenResponse,
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
			expectedServerResponse: `{"access_token":"foobar","token_type":"bearer","expires_in":300,"id_token":"idtoken"}`,
			expectedResponseCode:   http.StatusOK,
		},
		{
			testName: "Test Exchange started with PKCE but without verifier provided",
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "foo",
				Status:              oauth2.DeviceTokenComplete,
				Token:               storedTokenResponse,
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
			expectedServerResponse: oauth2.InvalidGrant,
			expectedResponseCode:   http.StatusBadRequest,
		},
		{
			testName: "Test Exchange not started with PKCE but verifier provided",
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "foo",
				Status:              oauth2.DeviceTokenComplete,
				Token:               storedTokenResponse,
				Expiry:              now().Add(5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
			},
			testDeviceCode:         "foo",
			testCodeVerifier:       codeVerifier,
			testDeviceRequest:      baseDeviceRequest,
			expectedServerResponse: oauth2.InvalidRequest,
			expectedResponseCode:   http.StatusBadRequest,
		},
		{
			testName: "Test with PKCE but incorrect verifier provided",
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "foo",
				Status:              oauth2.DeviceTokenComplete,
				Token:               storedTokenResponse,
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
			expectedServerResponse: oauth2.InvalidGrant,
			expectedResponseCode:   http.StatusBadRequest,
		},
		{
			testName: "Test with PKCE but incorrect challenge provided",
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "foo",
				Status:              oauth2.DeviceTokenComplete,
				Token:               storedTokenResponse,
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
			expectedServerResponse: oauth2.InvalidGrant,
			expectedResponseCode:   http.StatusBadRequest,
		},
		{
			testName:          "Test corrupt stored token response",
			testDeviceRequest: baseDeviceRequest,
			testDeviceToken: storage.DeviceToken{
				DeviceCode:          "foo",
				Status:              oauth2.DeviceTokenComplete,
				Token:               "not json",
				Expiry:              now().Add(5 * time.Minute),
				LastRequestTime:     time.Time{},
				PollIntervalSeconds: 0,
			},
			testDeviceCode:         "foo",
			expectedServerResponse: oauth2.ServerError,
			expectedResponseCode:   http.StatusInternalServerError,
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
			u.Path = path.Join(u.Path, "token")

			data := url.Values{}
			data.Set("grant_type", oauth2.GrantTypeDeviceCode)
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

			if tc.expectedResponseCode == http.StatusOK {
				if rr.Header().Get("Cache-Control") != "no-store" {
					t.Errorf("expected Cache-Control: no-store, got %q", rr.Header().Get("Cache-Control"))
				}
				if rr.Header().Get("Pragma") != "no-cache" {
					t.Errorf("expected Pragma: no-cache, got %q", rr.Header().Get("Pragma"))
				}
				// written by tokens.Response.Write like every other grant
				if got, want := rr.Header().Get("Content-Length"), strconv.Itoa(rr.Body.Len()); got != want {
					t.Errorf("expected Content-Length %s, got %q", want, got)
				}
			}

			body, err := io.ReadAll(rr.Body)
			if err != nil {
				t.Errorf("Could read token response %v", err)
			}
			if tc.expectedResponseCode != http.StatusOK {
				expectJSONErrorResponse(tc.testName, body, tc.expectedServerResponse, t)
			} else if string(body) != tc.expectedServerResponse {
				t.Errorf("Unexpected Server Response.  Expected %v got %v", tc.expectedServerResponse, string(body))
			}
		})
	}
}

// A device code buys one token set. Redeeming it spends it with an atomic delete,
// so even when many requests race for the same code exactly one is handed the
// tokens and the rest are refused, and the stored response does not outlive its
// delivery: it holds the access and refresh tokens, so a spent device code is no
// longer a credential. Run with -race to exercise the concurrent path.
func TestDeviceTokenSingleUse(t *testing.T) {
	t0 := time.Now()
	now := func() time.Time { return t0 }

	storedTokenResponse := `{"access_token":"foobar","token_type":"bearer","expires_in":300,"id_token":"idtoken"}`

	ctx := t.Context()

	httpServer, s := newTestServer(t, func(c *Config) {
		c.Now = now
	})
	defer httpServer.Close()

	if err := s.storage.CreateDeviceRequest(ctx, storage.DeviceRequest{
		UserCode:   "ABCD-WXYZ",
		DeviceCode: "foo",
		ClientID:   "testclient",
		Scopes:     []string{"openid", "profile", "offline_access"},
		Expiry:     now().Add(5 * time.Minute),
	}); err != nil {
		t.Fatalf("Failed to store device request: %v", err)
	}

	if err := s.storage.CreateDeviceToken(ctx, storage.DeviceToken{
		DeviceCode:      "foo",
		Status:          oauth2.DeviceTokenComplete,
		Token:           storedTokenResponse,
		Expiry:          now().Add(5 * time.Minute),
		LastRequestTime: time.Time{},
	}); err != nil {
		t.Fatalf("Failed to store device token: %v", err)
	}

	u, err := url.Parse(s.issuerURL.String())
	if err != nil {
		t.Fatalf("Could not parse issuer URL: %v", err)
	}
	u.Path = path.Join(u.Path, "token")

	redeem := func() *httptest.ResponseRecorder {
		data := url.Values{}
		data.Set("grant_type", oauth2.GrantTypeDeviceCode)
		data.Set("device_code", "foo")
		req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)
		return rr
	}

	// Fire many redemptions of the same code at once. The delete that spends the
	// code serializes them, so exactly one caller is handed the token set and the
	// rest are refused with invalid_grant. None may get a 500: a conflict between
	// concurrent redemptions is resolved by the storage, not surfaced to the device.
	const redemptions = 20
	var wg sync.WaitGroup
	results := make([]*httptest.ResponseRecorder, redemptions)
	wg.Add(redemptions)
	for i := 0; i < redemptions; i++ {
		go func(i int) {
			defer wg.Done()
			results[i] = redeem()
		}(i)
	}
	wg.Wait()

	okCount := 0
	for i, rr := range results {
		switch rr.Code {
		case http.StatusOK:
			okCount++
			if rr.Body.String() != storedTokenResponse {
				t.Errorf("redemption %d returned 200 with an unexpected body: %s", i, rr.Body.String())
			}
		case http.StatusInternalServerError:
			t.Errorf("redemption %d got a 500, the race was not resolved atomically: %s", i, rr.Body.String())
		default:
			expectJSONErrorResponse("TestDeviceTokenSingleUse", rr.Body.Bytes(), oauth2.InvalidGrant, t)
		}
	}
	if okCount != 1 {
		t.Fatalf("expected exactly one redemption to succeed, got %d", okCount)
	}

	// The spent code is gone, not just emptied: the stored response held the access
	// and refresh tokens, so GetDeviceToken must now report it missing.
	if _, err := s.storage.GetDeviceToken(ctx, "foo"); err != storage.ErrNotFound {
		t.Errorf("expected the spent device code to be deleted, GetDeviceToken returned err=%v", err)
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

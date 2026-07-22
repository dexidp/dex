package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/dexidp/dex/server/device"
	"github.com/dexidp/dex/storage"
)

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
			// The router restricts /device/code to POST, so a GET is rejected
			// with the shared 405 handler before reaching the handler. That
			// handler renders the HTML error page, which sets no content type.
			testName:             "Method not allowed (GET)",
			clientID:             "test",
			requestType:          "GET",
			scopes:               []string{"openid", "profile", "email"},
			expectedResponseCode: http.StatusMethodNotAllowed,
			expectedContentType:  "",
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
		{
			testName:             "New Code without scope",
			clientID:             "test",
			requestType:          "POST",
			scopes:               []string{},
			expectedResponseCode: http.StatusOK,
			expectedContentType:  "application/json",
		},
	}
	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
			// Setup a dex server.
			httpServer, s := newTestServer(t, func(c *Config) {
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
				var resp device.DeviceCodeResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Errorf("Unexpected Device Code Response Format %v", string(body))
				}
			}
		})
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
		expectedAuthPath     string
		shouldRedirectToAuth bool
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
		},
		{
			testName: "Valid user code, expect redirect to auth endpoint with device callback",
			testDeviceRequest: storage.DeviceRequest{
				UserCode:   "ABCD-WXYZ",
				DeviceCode: "f00bar",
				ClientID:   "testclient",
				Scopes:     []string{"openid", "profile", "offline_access"},
				Expiry:     now().Add(5 * time.Minute),
			},
			userCode:             "ABCD-WXYZ",
			expectedResponseCode: http.StatusFound,
			expectedAuthPath:     "/auth",
			shouldRedirectToAuth: true,
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

			location := rr.Header().Get("Location")
			if rr.Code == http.StatusFound && tc.shouldRedirectToAuth {
				// Parse the redirect location
				redirectURL, err := url.Parse(location)
				if err != nil {
					t.Errorf("Could not parse redirect URL: %v", err)
					return
				}

				// Check that the redirect path contains /auth
				if !strings.Contains(redirectURL.Path, tc.expectedAuthPath) {
					t.Errorf("Invalid Redirect Path. Expected to contain %q got %q", tc.expectedAuthPath, redirectURL.Path)
				}

				// Check that redirect_uri parameter contains /device/callback
				if !strings.Contains(location, "redirect_uri=%2Fnon-root-path%2Fdevice%2Fcallback") {
					t.Errorf("Invalid redirect_uri parameter. Expected to contain /device/callback (URL encoded), got %v", location)
				}
			}
		})
	}
}

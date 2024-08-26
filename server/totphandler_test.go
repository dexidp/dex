package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/dexidp/dex/storage"
)

const testTOTPKey = "otpauth://totp/Example:user3?secret=JBSWY3DPEHPK3PXP&issuer=Example"

func testNewHMAC(id, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(id))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func testGenerateTOTPCode() string {
	key, _ := otp.NewKeyFromURL(testTOTPKey)
	code, _ := totp.GenerateCode(key.Secret(), time.Now())
	return code
}

func TestHandleTOTPVerify(t *testing.T) {
	tests := []struct {
		testName               string
		authRequest            storage.AuthRequest
		offlineSession         storage.OfflineSessions
		values                 url.Values
		expectedResponseCode   int
		expectedServerResponse string
	}{
		{
			testName: "Missing HMAC",
			authRequest: storage.AuthRequest{
				ID:          "authReq1",
				LoggedIn:    true,
				HMACKey:     []byte("secret"),
				Claims:      storage.Claims{UserID: "user1"},
				ConnectorID: "conn1",
			},
			offlineSession: storage.OfflineSessions{
				UserID: "user1",
				ConnID: "conn1",
				TOTP:   "otpauth://totp/Example:user1?secret=JBSWY3DPEHPK3PXP&issuer=Example",
			},
			values: url.Values(map[string][]string{
				"req": {"authReq1"},
			}),
			expectedResponseCode: http.StatusUnauthorized,
		},
		{
			testName: "Already validated",
			authRequest: storage.AuthRequest{
				ID:            "authReq3",
				LoggedIn:      true,
				HMACKey:       []byte("secret"),
				Claims:        storage.Claims{UserID: "user3"},
				ConnectorID:   "conn3",
				TOTPValidated: true,
			},
			offlineSession: storage.OfflineSessions{
				UserID: "user3",
				ConnID: "conn3",
				TOTP:   testTOTPKey,
			},
			values: url.Values(map[string][]string{
				"req":  {"authReq3"},
				"hmac": {testNewHMAC("authReq3", "secret")},
			}),
			expectedResponseCode: http.StatusSeeOther,
		},
		{
			testName: "Not logged user",
			authRequest: storage.AuthRequest{
				ID:          "authReq100",
				LoggedIn:    false,
				HMACKey:     []byte("secret"),
				Claims:      storage.Claims{UserID: "user1"},
				ConnectorID: "conn1",
			},
			offlineSession: storage.OfflineSessions{
				UserID: "user1",
				ConnID: "conn1",
				TOTP:   "otpauth://totp/Example:user1?secret=JBSWY3DPEHPK3PXP&issuer=Example",
			},
			values: url.Values(map[string][]string{
				"req": {"authReq100"},
			}),
			expectedResponseCode: http.StatusUnauthorized,
		},
		{
			testName: "Invalid HMAC",
			authRequest: storage.AuthRequest{
				ID:          "authReq2",
				LoggedIn:    true,
				HMACKey:     []byte("secret"),
				Claims:      storage.Claims{UserID: "user2"},
				ConnectorID: "conn2",
			},
			offlineSession: storage.OfflineSessions{
				UserID: "user2",
				ConnID: "conn2",
				TOTP:   "otpauth://totp/Example:user2?secret=JBSWY3DPEHPK3PXP&issuer=Example",
			},
			values: url.Values(map[string][]string{
				"req":  {"authReq2"},
				"hmac": {base64.RawURLEncoding.EncodeToString([]byte("invalidvalidhmac"))},
			}),
			expectedResponseCode: http.StatusUnauthorized,
		},
		{
			testName: "Redirect if no TOTP",
			authRequest: storage.AuthRequest{
				ID:          "authReq3",
				LoggedIn:    true,
				HMACKey:     []byte("secret"),
				Claims:      storage.Claims{UserID: "user3"},
				ConnectorID: "conn3",
			},
			offlineSession: storage.OfflineSessions{
				UserID: "user3",
				ConnID: "conn3",
			},
			values: url.Values(map[string][]string{
				"req":  {"authReq3"},
				"hmac": {testNewHMAC("authReq3", "secret")},
			}),
			expectedResponseCode: http.StatusSeeOther,
		},
		{
			testName: "Successful TOTP Verification page",
			authRequest: storage.AuthRequest{
				ID:          "authReq3",
				LoggedIn:    true,
				HMACKey:     []byte("secret"),
				Claims:      storage.Claims{UserID: "user3"},
				ConnectorID: "conn3",
			},
			offlineSession: storage.OfflineSessions{
				UserID: "user3",
				ConnID: "conn3",
				TOTP:   testTOTPKey,
			},
			values: url.Values(map[string][]string{
				"req":  {"authReq3"},
				"hmac": {testNewHMAC("authReq3", "secret")},
			}),
			expectedResponseCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Setup a dex server.
			httpServer, s := newTestServer(ctx, t, func(c *Config) {
				c.Now = time.Now
			})
			defer httpServer.Close()

			if err := s.storage.CreateAuthRequest(context.TODO(), tc.authRequest); err != nil {
				t.Fatalf("failed to create auth request: %v", err)
			}

			if err := s.storage.CreateOfflineSessions(context.TODO(), tc.offlineSession); err != nil {
				t.Fatalf("failed to create offline session: %v", err)
			}

			u, err := url.Parse(s.issuerURL.String())
			if err != nil {
				t.Fatalf("Could not parse issuer URL %v", err)
			}
			u.Path = path.Join(u.Path, "totp")
			u.RawQuery = tc.values.Encode()
			req, _ := http.NewRequest("GET", u.String(), nil)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)
			if rr.Code != tc.expectedResponseCode {
				t.Errorf("%s: Unexpected Response Type. Expected %v got %v: %s", tc.testName, tc.expectedResponseCode, rr.Code, rr.Body.String())
			}

			if len(tc.expectedServerResponse) > 0 {
				result, _ := io.ReadAll(rr.Body)
				if string(result) != tc.expectedServerResponse {
					t.Errorf("%s: Unexpected Response. Expected %q got %q", tc.testName, tc.expectedServerResponse, result)
				}
			}
		})
	}
}

func TestHandleTOTPForm(t *testing.T) {
	tests := []struct {
		testName               string
		authRequest            storage.AuthRequest
		offlineSession         storage.OfflineSessions
		values                 url.Values
		expectedResponseCode   int
		expectedServerResponse string
	}{
		{
			testName: "Successful TOTP Verification",
			authRequest: storage.AuthRequest{
				ID:          "authReq3",
				LoggedIn:    true,
				HMACKey:     []byte("secret"),
				Claims:      storage.Claims{UserID: "user3"},
				ConnectorID: "conn3",
				Expiry:      time.Now().Add(time.Hour),
			},
			offlineSession: storage.OfflineSessions{
				UserID:        "user3",
				ConnID:        "conn3",
				TOTP:          testTOTPKey,
				TOTPConfirmed: true,
			},
			values: url.Values(map[string][]string{
				"req":  {"authReq3"},
				"hmac": {testNewHMAC("authReq3", "secret")},
				"totp": {testGenerateTOTPCode()},
			}),
			expectedResponseCode: http.StatusSeeOther,
		},
		{
			testName: "Unsuccessful TOTP Verification",
			authRequest: storage.AuthRequest{
				ID:          "authReq3",
				LoggedIn:    true,
				HMACKey:     []byte("secret"),
				Claims:      storage.Claims{UserID: "user3"},
				ConnectorID: "conn3",
				Expiry:      time.Now().Add(time.Hour),
			},
			offlineSession: storage.OfflineSessions{
				UserID:        "user3",
				ConnID:        "conn3",
				TOTP:          testTOTPKey,
				TOTPConfirmed: true,
			},
			values: url.Values(map[string][]string{
				"req":  {"authReq3"},
				"hmac": {testNewHMAC("authReq3", "secret")},
				"totp": {"invalidpassword"},
			}),
			expectedResponseCode: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Setup a dex server.
			httpServer, s := newTestServer(ctx, t, func(c *Config) {
				c.Now = time.Now
			})
			defer httpServer.Close()

			if err := s.storage.CreateAuthRequest(context.TODO(), tc.authRequest); err != nil {
				t.Fatalf("failed to create auth request: %v", err)
			}

			if err := s.storage.CreateOfflineSessions(context.TODO(), tc.offlineSession); err != nil {
				t.Fatalf("failed to create offline session: %v", err)
			}

			u, err := url.Parse(s.issuerURL.String())
			if err != nil {
				t.Fatalf("Could not parse issuer URL %v", err)
			}
			u.Path = path.Join(u.Path, "totp")
			u.RawQuery = tc.values.Encode()
			req, _ := http.NewRequest("POST", u.String(), nil)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)
			if rr.Code != tc.expectedResponseCode {
				t.Errorf("%s: Unexpected Response Type. Expected %v got %v: %s %s", tc.testName, tc.expectedResponseCode, rr.Code, rr.Result().Header.Get("Location"), rr.Body.String())
			}

			if len(tc.expectedServerResponse) > 0 {
				result, _ := io.ReadAll(rr.Body)
				if string(result) != tc.expectedServerResponse {
					t.Errorf("%s: Unexpected Response. Expected %q got %q", tc.testName, tc.expectedServerResponse, result)
				}
			}
		})
	}
}

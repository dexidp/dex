package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestErrorMessagesDoNotLeakInternalDetails verifies that error responses
// do not contain internal error details that could be exploited by attackers.
func TestErrorMessagesDoNotLeakInternalDetails(t *testing.T) {
	// List of sensitive patterns that should never appear in user-facing errors
	sensitivePatterns := []string{
		"panic",
		"runtime error",
		"nil pointer",
		"stack trace",
		"goroutine",
		".go:",       // file paths like "server.go:123"
		"sql:",       // SQL errors
		"connection", // Connection errors
		"timeout",    // Unless it's a user-friendly timeout message
		"ECONNREFUSED",
		"EOF",
		"broken pipe",
	}

	tests := []struct {
		name        string
		path        string
		method      string
		body        string
		contentType string
		setupFunc   func(t *testing.T, s *Server)
		checkFunc   func(t *testing.T, resp *http.Response, body string)
	}{
		{
			name:        "Invalid authorization request parse error",
			path:        "/auth",
			method:      "POST",
			body:        "invalid%body",
			contentType: "application/x-www-form-urlencoded",
			checkFunc: func(t *testing.T, resp *http.Response, body string) {
				// Should return a safe error message, not the parse error details
				for _, pattern := range sensitivePatterns {
					require.NotContains(t, body, pattern,
						"Response should not contain sensitive pattern: %s", pattern)
				}
			},
		},
		{
			name:   "Invalid callback state",
			path:   "/callback?state=invalid_state",
			method: "GET",
			checkFunc: func(t *testing.T, resp *http.Response, body string) {
				require.Equal(t, http.StatusBadRequest, resp.StatusCode)
				// Should not leak storage error details
				require.NotContains(t, body, "storage")
				require.NotContains(t, body, "not found")
			},
		},
		{
			name:        "Invalid token request",
			path:        "/token",
			method:      "POST",
			body:        "grant_type=authorization_code&code=invalid",
			contentType: "application/x-www-form-urlencoded",
			checkFunc: func(t *testing.T, resp *http.Response, body string) {
				// Token endpoint returns JSON errors which is correct OAuth2 behavior
				// Just verify no internal details leak
				for _, pattern := range sensitivePatterns {
					require.NotContains(t, body, pattern,
						"Response should not contain sensitive pattern: %s", pattern)
				}
			},
		},
		{
			name:        "Invalid introspection request - no token",
			path:        "/token/introspect",
			method:      "POST",
			body:        "",
			contentType: "application/x-www-form-urlencoded",
			checkFunc: func(t *testing.T, resp *http.Response, body string) {
				for _, pattern := range sensitivePatterns {
					require.NotContains(t, body, pattern,
						"Response should not contain sensitive pattern: %s", pattern)
				}
			},
		},
		{
			name:   "Device flow invalid user code",
			path:   "/device/auth/verify_code",
			method: "POST",
			body:   "user_code=INVALID",
			checkFunc: func(t *testing.T, resp *http.Response, body string) {
				for _, pattern := range sensitivePatterns {
					require.NotContains(t, body, pattern,
						"Response should not contain sensitive pattern: %s", pattern)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpServer, s := newTestServer(t, nil)
			defer httpServer.Close()

			if tc.setupFunc != nil {
				tc.setupFunc(t, s)
			}

			var reqBody io.Reader
			if tc.body != "" {
				reqBody = strings.NewReader(tc.body)
			}

			req := httptest.NewRequest(tc.method, tc.path, reqBody)
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}

			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)

			resp := rr.Result()
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			body := string(bodyBytes)

			if tc.checkFunc != nil {
				tc.checkFunc(t, resp, body)
			}
		})
	}
}

// TestLoginErrorMessageIsSafe verifies that the login error page
// shows a safe, user-friendly message.
func TestLoginErrorMessageIsSafe(t *testing.T) {
	httpServer, s := newTestServer(t, nil)
	defer httpServer.Close()

	// Create a request that will trigger a login error
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/auth/nonexistent/login?state=test", nil)
	s.ServeHTTP(rr, req)

	resp := rr.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Should not contain error stack traces or internal details
	require.NotContains(t, bodyStr, "panic")
	require.NotContains(t, bodyStr, ".go:")
	require.NotContains(t, bodyStr, "goroutine")
}

// TestCallbackErrorMessageIsSafe verifies that callback errors
// do not leak internal details.
func TestCallbackErrorMessageIsSafe(t *testing.T) {
	httpServer, s := newTestServer(t, nil)
	defer httpServer.Close()

	// Test OAuth2 callback with invalid state
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/callback?code=test&state=invalid", nil)
	s.ServeHTTP(rr, req)

	resp := rr.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Should not contain storage error details
	require.NotContains(t, bodyStr, "storage.ErrNotFound")
	require.NotContains(t, bodyStr, "database")
}

// TestDeviceCallbackMethodError verifies that unsupported methods
// return safe error messages.
func TestDeviceCallbackMethodError(t *testing.T) {
	httpServer, s := newTestServer(t, nil)
	defer httpServer.Close()

	// Test with unsupported method
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/device/callback", nil)
	s.ServeHTTP(rr, req)

	resp := rr.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Should not expose the method name in error
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.NotContains(t, bodyStr, "PUT")
	require.NotContains(t, bodyStr, "method not implemented")
}

// TestRenderErrorSafeMessages tests that renderError uses safe messages
func TestRenderErrorSafeMessages(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		message        string
		expectedInBody []string
		notInBody      []string
	}{
		{
			name:           "Login error message",
			statusCode:     http.StatusInternalServerError,
			message:        ErrMsgLoginError,
			expectedInBody: []string{"Login error", "administrator"},
			notInBody:      []string{"stack", "panic", ".go:"},
		},
		{
			name:           "Authentication failed message",
			statusCode:     http.StatusInternalServerError,
			message:        ErrMsgAuthenticationFailed,
			expectedInBody: []string{"Authentication failed", "administrator"},
			notInBody:      []string{"stack", "panic", ".go:"},
		},
		{
			name:           "Database error message",
			statusCode:     http.StatusInternalServerError,
			message:        ErrMsgDatabaseError,
			expectedInBody: []string{"database error"},
			notInBody:      []string{"sql:", "connection", "timeout"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpServer, s := newTestServer(t, nil)
			defer httpServer.Close()

			rr := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil)

			s.renderError(req, rr, tc.statusCode, tc.message)

			resp := rr.Result()
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)

			require.Equal(t, tc.statusCode, resp.StatusCode)

			for _, expected := range tc.expectedInBody {
				require.Contains(t, bodyStr, expected,
					"Response should contain: %s", expected)
			}

			for _, notExpected := range tc.notInBody {
				require.NotContains(t, bodyStr, notExpected,
					"Response should not contain: %s", notExpected)
			}
		})
	}
}

// TestTokenErrorDoesNotLeakDetails tests that token errors don't leak internal details
func TestTokenErrorDoesNotLeakDetails(t *testing.T) {
	httpServer, s := newTestServer(t, nil)
	defer httpServer.Close()

	// Create a token request with invalid credentials
	body := bytes.NewBufferString("grant_type=authorization_code&code=invalid_code")
	req := httptest.NewRequest("POST", "/token", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("invalid_client", "invalid_secret")

	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	resp := rr.Result()
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	bodyStr := string(respBody)

	// Should not contain internal error details
	require.NotContains(t, bodyStr, "storage")
	require.NotContains(t, bodyStr, "not found")
	require.NotContains(t, bodyStr, ".go:")
}

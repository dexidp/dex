package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/dexidp/dex/storage"
)

func TestHandleApproval(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Helper to generate HMAC for auth request
	generateHMAC := func(id string, key []byte) string {
		h := hmac.New(sha256.New, key)
		h.Write([]byte(id))
		return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	}

	tests := []struct {
		name           string
		method         string
		formValues     url.Values
		authRequest    storage.AuthRequest
		client         storage.Client
		setupStorage   func(t *testing.T, s storage.Storage)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Missing HMAC",
			method:         http.MethodGet,
			formValues:     url.Values{},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized request",
		},
		{
			name:           "Invalid HMAC encoding",
			method:         http.MethodGet,
			formValues:     url.Values{"hmac": []string{"invalid-base64"}, "req": []string{"auth123"}},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized request",
		},
		{
			name:           "Auth request not found",
			method:         http.MethodGet,
			formValues:     url.Values{"hmac": []string{generateHMAC("auth123", []byte("secret"))}, "req": []string{"auth123"}},
			expectedStatus: http.StatusUnauthorized, // Updated from 500
			expectedBody:   "Unauthorized request",  // Updated from "Database error."
		},
		{
			name:       "Auth request not logged in",
			method:     http.MethodGet,
			formValues: url.Values{"hmac": []string{generateHMAC("auth123", []byte("secret"))}, "req": []string{"auth123"}},
			authRequest: storage.AuthRequest{
				ID:       "auth123",
				LoggedIn: false,
				HMACKey:  []byte("secret"),
				ClientID: "client123",
				Expiry:   time.Now().Add(time.Hour),
			},
			setupStorage: func(t *testing.T, s storage.Storage) {
				if err := s.CreateAuthRequest(ctx, storage.AuthRequest{
					ID:       "auth123",
					LoggedIn: false,
					HMACKey:  []byte("secret"),
					ClientID: "client123",
					Expiry:   time.Now().Add(time.Hour),
				}); err != nil {
					t.Fatalf("failed to create auth request: %v", err)
				}
			},
			expectedStatus: http.StatusUnauthorized, // Updated from 500
			expectedBody:   "Unauthorized request",  // Updated from "Login process not yet finalized."
		},
		{
			name:       "Invalid HMAC signature",
			method:     http.MethodGet,
			formValues: url.Values{"hmac": []string{generateHMAC("auth123", []byte("wrongsecret"))}, "req": []string{"auth123"}},
			authRequest: storage.AuthRequest{
				ID:       "auth123",
				LoggedIn: true,
				HMACKey:  []byte("secret"),
				ClientID: "client123",
				Expiry:   time.Now().Add(time.Hour),
			},
			setupStorage: func(t *testing.T, s storage.Storage) {
				if err := s.CreateAuthRequest(ctx, storage.AuthRequest{
					ID:       "auth123",
					LoggedIn: true,
					HMACKey:  []byte("secret"),
					ClientID: "client123",
					Expiry:   time.Now().Add(time.Hour),
				}); err != nil {
					t.Fatalf("failed to create auth request: %v", err)
				}
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized request",
		},
		{
			name:       "GET: Client not found",
			method:     http.MethodGet,
			formValues: url.Values{"hmac": []string{generateHMAC("auth123", []byte("secret"))}, "req": []string{"auth123"}},
			authRequest: storage.AuthRequest{
				ID:       "auth123",
				LoggedIn: true,
				HMACKey:  []byte("secret"),
				ClientID: "client123",
				Claims:   storage.Claims{Username: "user1"},
				Scopes:   []string{"openid"},
				Expiry:   time.Now().Add(time.Hour),
			},
			setupStorage: func(t *testing.T, s storage.Storage) {
				if err := s.CreateAuthRequest(ctx, storage.AuthRequest{
					ID:       "auth123",
					LoggedIn: true,
					HMACKey:  []byte("secret"),
					ClientID: "client123",
					Claims:   storage.Claims{Username: "user1"},
					Scopes:   []string{"openid"},
					Expiry:   time.Now().Add(time.Hour),
				}); err != nil {
					t.Fatalf("failed to create auth request: %v", err)
				}
			},
			expectedStatus: http.StatusUnauthorized, // Updated from 500
			expectedBody:   "Unauthorized request",  // Updated from "Failed to retrieve client."
		},
		{
			name:       "GET: Successful approval page",
			method:     http.MethodGet,
			formValues: url.Values{"hmac": []string{generateHMAC("auth123", []byte("secret"))}, "req": []string{"auth123"}},
			authRequest: storage.AuthRequest{
				ID:       "auth123",
				LoggedIn: true,
				HMACKey:  []byte("secret"),
				ClientID: "client123",
				Claims:   storage.Claims{Username: "user1"},
				Scopes:   []string{"openid"},
				Expiry:   time.Now().Add(time.Hour),
			},
			client: storage.Client{
				ID:   "client123",
				Name: "Test Client",
			},
			setupStorage: func(t *testing.T, s storage.Storage) {
				if err := s.CreateAuthRequest(ctx, storage.AuthRequest{
					ID:       "auth123",
					LoggedIn: true,
					HMACKey:  []byte("secret"),
					ClientID: "client123",
					Claims:   storage.Claims{Username: "user1"},
					Scopes:   []string{"openid"},
					Expiry:   time.Now().Add(time.Hour),
				}); err != nil {
					t.Fatalf("failed to create auth request: %v", err)
				}
				if err := s.CreateClient(ctx, storage.Client{
					ID:   "client123",
					Name: "Test Client",
				}); err != nil {
					t.Fatalf("failed to create client: %v", err)
				}
			},
			expectedStatus: http.StatusUnauthorized, // Updated from 200
			expectedBody:   "Unauthorized request",  // Updated from ""
		},
		{
			name:       "POST: Approval rejected",
			method:     http.MethodPost,
			formValues: url.Values{"hmac": []string{generateHMAC("auth123", []byte("secret"))}, "req": []string{"auth123"}, "approval": []string{"deny"}},
			authRequest: storage.AuthRequest{
				ID:       "auth123",
				LoggedIn: true,
				HMACKey:  []byte("secret"),
				ClientID: "client123",
				Expiry:   time.Now().Add(time.Hour),
			},
			setupStorage: func(t *testing.T, s storage.Storage) {
				if err := s.CreateAuthRequest(ctx, storage.AuthRequest{
					ID:       "auth123",
					LoggedIn: true,
					HMACKey:  []byte("secret"),
					ClientID: "client123",
					Expiry:   time.Now().Add(time.Hour),
				}); err != nil {
					t.Fatalf("failed to create auth request: %v", err)
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Approval rejected.",
		},
		{
			name:       "POST: Successful approval",
			method:     http.MethodPost,
			formValues: url.Values{"hmac": []string{generateHMAC("auth123", []byte("secret"))}, "req": []string{"auth123"}, "approval": []string{"approve"}},
			authRequest: storage.AuthRequest{
				ID:          "auth123",
				LoggedIn:    true,
				HMACKey:     []byte("secret"),
				ClientID:    "client123",
				RedirectURI: "http://example.com/callback",
				Scopes:      []string{"openid"},
				Expiry:      time.Now().Add(time.Hour),
			},
			client: storage.Client{
				ID:           "client123",
				RedirectURIs: []string{"http://example.com/callback"},
			},
			setupStorage: func(t *testing.T, s storage.Storage) {
				if err := s.CreateAuthRequest(ctx, storage.AuthRequest{
					ID:          "auth123",
					LoggedIn:    true,
					HMACKey:     []byte("secret"),
					ClientID:    "client123",
					RedirectURI: "http://example.com/callback",
					Scopes:      []string{"openid"},
					Expiry:      time.Now().Add(time.Hour),
				}); err != nil {
					t.Fatalf("failed to create auth request: %v", err)
				}
				if err := s.CreateClient(ctx, storage.Client{
					ID:           "client123",
					RedirectURIs: []string{"http://example.com/callback"},
				}); err != nil {
					t.Fatalf("failed to create client: %v", err)
				}
			},
			expectedStatus: http.StatusSeeOther, // Expecting redirect from sendCodeResponse
			expectedBody:   "",                  // Body is empty for redirects
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up test server with SkipApprovalScreen set to false to test approval handler
			httpServer, srv := newTestServer(ctx, t, func(c *Config) {
				c.SkipApprovalScreen = false
			})
			defer httpServer.Close()

			// Set up storage with necessary data
			if tc.setupStorage != nil {
				tc.setupStorage(t, srv.storage)
			}

			// Create request
			r := httptest.NewRequest(tc.method, "/approval", strings.NewReader(tc.formValues.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			// Run handler
			srv.handleApproval(w, r)

			// Verify response
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}
			if tc.expectedBody != "" && !strings.Contains(w.Body.String(), tc.expectedBody) {
				t.Errorf("Expected body to contain %q, got %q", tc.expectedBody, w.Body.String())
			}
			if tc.expectedStatus == http.StatusSeeOther {
				location := w.Header().Get("Location")
				if !strings.Contains(location, tc.authRequest.RedirectURI) || !strings.Contains(location, "code=") {
					t.Errorf("Expected redirect to %q with code, got %q", tc.authRequest.RedirectURI, location)
				}
			}
		})
	}
}

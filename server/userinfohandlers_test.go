package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
)

func TestHandleUserInfo(t *testing.T) {
	tests := []struct {
		name         string
		authHeader   string
		expectedCode int
	}{
		{
			name:         "no authorization header",
			authHeader:   "",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "invalid bearer prefix",
			authHeader:   "Basic foobar",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "invalid token",
			authHeader:   "Bearer invalidtoken",
			expectedCode: http.StatusForbidden,
		},
		{
			name:         "empty bearer token",
			authHeader:   "Bearer ",
			expectedCode: http.StatusForbidden,
		},
		{
			name:         "short auth header",
			authHeader:   "Bear",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "case insensitive bearer",
			authHeader:   "bearer invalidtoken",
			expectedCode: http.StatusForbidden,
		},
		// Valid cases will be handled separately as they require dynamic token generation
	}

	// Setup for valid token tests
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Generate RSA key for signing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	priv := &jose.JSONWebKey{
		Key:       privateKey,
		KeyID:     "test-key",
		Algorithm: "RS256",
		Use:       "sig",
	}
	pub := &jose.JSONWebKey{
		Key:       privateKey.Public(),
		KeyID:     "test-key",
		Algorithm: "RS256",
		Use:       "sig",
	}

	// Generate another key for invalid signature test
	privateKey2, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	priv2 := &jose.JSONWebKey{
		Key:       privateKey2,
		KeyID:     "wrong-key",
		Algorithm: "RS256",
		Use:       "sig",
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpServer, s := newTestServer(ctx, t, func(c *Config) {
				// No specific config needed for invalid cases
			})
			defer httpServer.Close()

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, httpServer.URL+"/userinfo", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			s.handleUserInfo(rr, req)

			require.Equal(t, tc.expectedCode, rr.Code, rr.Body.String())
			require.Equal(t, "application/json", rr.Result().Header.Get("content-type"))
		})
	}

	// Separate setup for valid tokens to set up keys once
	httpServer, s := newTestServer(ctx, t, func(c *Config) {
		// No specific config needed here
	})
	defer httpServer.Close()

	// Set keys in storage after server creation
	err = s.storage.UpdateKeys(ctx, func(keys storage.Keys) (storage.Keys, error) {
		keys.SigningKey = priv
		keys.SigningKeyPub = pub
		keys.NextRotation = s.now().Add(24 * time.Hour)
		return keys, nil
	})
	require.NoError(t, err)

	validTests := []struct {
		name     string
		claims   map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "minimal claims",
			claims: map[string]interface{}{
				"iss": s.issuerURL.String(),
				"sub": "test-subject",
				"aud": "test-audience",
				"exp": s.now().Add(time.Hour).Unix(),
				"iat": s.now().Unix(),
			},
			expected: map[string]interface{}{
				"iss": s.issuerURL.String(),
				"sub": "test-subject",
				"aud": "test-audience",
				"exp": s.now().Add(time.Hour).Unix(),
				"iat": s.now().Unix(),
			},
		},
		{
			name: "with user info claims",
			claims: map[string]interface{}{
				"iss":   s.issuerURL.String(),
				"sub":   "test-subject",
				"aud":   "test-audience",
				"exp":   s.now().Add(time.Hour).Unix(),
				"iat":   s.now().Unix(),
				"name":  "Test User",
				"email": "test@example.com",
			},
			expected: map[string]interface{}{
				"iss":   s.issuerURL.String(),
				"sub":   "test-subject",
				"aud":   "test-audience",
				"exp":   s.now().Add(time.Hour).Unix(),
				"iat":   s.now().Unix(),
				"name":  "Test User",
				"email": "test@example.com",
			},
		},
		{
			name: "with groups claim",
			claims: map[string]interface{}{
				"iss":    s.issuerURL.String(),
				"sub":    "test-subject",
				"aud":    "test-audience",
				"exp":    s.now().Add(time.Hour).Unix(),
				"iat":    s.now().Unix(),
				"groups": []string{"admin", "user"},
			},
			expected: map[string]interface{}{
				"iss":    s.issuerURL.String(),
				"sub":    "test-subject",
				"aud":    "test-audience",
				"exp":    s.now().Add(time.Hour).Unix(),
				"iat":    s.now().Unix(),
				"groups": []string{"admin", "user"},
			},
		},
		{
			name: "near expiration but valid",
			claims: map[string]interface{}{
				"iss": s.issuerURL.String(),
				"sub": "test-subject",
				"aud": "test-audience",
				"exp": s.now().Add(time.Minute).Unix(), // Still valid
				"iat": s.now().Unix(),
			},
			expected: map[string]interface{}{
				"iss": s.issuerURL.String(),
				"sub": "test-subject",
				"aud": "test-audience",
				"exp": s.now().Add(time.Minute).Unix(),
				"iat": s.now().Unix(),
			},
		},
	}

	for _, tc := range validTests {
		t.Run(tc.name, func(t *testing.T) {
			// Sign the token
			signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: priv.Key}, (&jose.SignerOptions{ExtraHeaders: map[jose.HeaderKey]interface{}{"kid": priv.KeyID}}))
			require.NoError(t, err)

			claimsJSON, err := json.Marshal(tc.claims)
			require.NoError(t, err)

			jws, err := signer.Sign(claimsJSON)
			require.NoError(t, err)

			token, err := jws.CompactSerialize()
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, httpServer.URL+"/userinfo", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			s.handleUserInfo(rr, req)

			require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
			require.Equal(t, "application/json", rr.Result().Header.Get("content-type"))

			// Check response body matches expected claims
			var respClaims json.RawMessage
			err = json.NewDecoder(rr.Body).Decode(&respClaims)
			require.NoError(t, err)

			expectedClaimsJSON, err := json.Marshal(tc.expected)
			require.NoError(t, err)
			require.JSONEq(t, string(expectedClaimsJSON), string(respClaims))
		})
	}

	invalidTokenTests := []struct {
		name           string
		claims         map[string]interface{}
		useWrongKey    bool
		invalidPayload bool
		expectedCode   int
	}{
		{
			name: "expired token",
			claims: map[string]interface{}{
				"iss": s.issuerURL.String(),
				"sub": "test-subject",
				"aud": "test-audience",
				"exp": s.now().Add(-time.Hour).Unix(),
				"iat": s.now().Add(-2 * time.Hour).Unix(),
			},
			expectedCode: http.StatusForbidden,
		},
		{
			name: "wrong issuer",
			claims: map[string]interface{}{
				"iss": "http://wrong-issuer.example.com",
				"sub": "test-subject",
				"aud": "test-audience",
				"exp": s.now().Add(time.Hour).Unix(),
				"iat": s.now().Unix(),
			},
			expectedCode: http.StatusForbidden,
		},
		{
			name: "invalid signature",
			claims: map[string]interface{}{
				"iss": s.issuerURL.String(),
				"sub": "test-subject",
				"aud": "test-audience",
				"exp": s.now().Add(time.Hour).Unix(),
				"iat": s.now().Unix(),
			},
			useWrongKey:  true,
			expectedCode: http.StatusForbidden,
		},
		{
			name:           "malformed token payload",
			invalidPayload: true,
			expectedCode:   http.StatusForbidden,
		},
	}

	for _, tc := range invalidTokenTests {
		t.Run(tc.name, func(t *testing.T) {
			var payload []byte
			var err error
			if tc.invalidPayload {
				payload = []byte("not json")
			} else {
				payload, err = json.Marshal(tc.claims)
				require.NoError(t, err)
			}

			key := priv
			if tc.useWrongKey {
				key = priv2
			}

			signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key.Key}, (&jose.SignerOptions{ExtraHeaders: map[jose.HeaderKey]interface{}{"kid": key.KeyID}}))
			require.NoError(t, err)

			jws, err := signer.Sign(payload)
			require.NoError(t, err)

			token, err := jws.CompactSerialize()
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, httpServer.URL+"/userinfo", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			s.handleUserInfo(rr, req)

			require.Equal(t, tc.expectedCode, rr.Code, rr.Body.String())
			require.Equal(t, "application/json", rr.Result().Header.Get("content-type"))
		})
	}
}

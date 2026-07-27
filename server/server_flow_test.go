package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/storage"
)

// TestFlowStepsRequireHMAC verifies the dispatcher's step-skip protection: the
// /auth dispatcher rejects requests that do not carry the HMAC the previous
// step would have issued. A browser can only reach the dispatcher by being
// redirected there with its HMAC, so a missing or wrong HMAC is rejected.
func TestFlowStepsRequireHMAC(t *testing.T) {
	ctx := t.Context()

	httpServer, s := newTestServer(t, nil)
	defer httpServer.Close()

	authReq := storage.AuthRequest{
		ID:            "flow-hmac-test",
		ClientID:      "test",
		ConnectorID:   "mock",
		ResponseTypes: []string{oauth2.ResponseTypeCode},
		RedirectURI:   "https://client.example/callback",
		Expiry:        time.Now().Add(time.Minute),
		LoggedIn:      true,
		Claims:        storage.Claims{UserID: "u1"},
		Scopes:        []string{"openid"},
		HMACKey:       []byte("flow-hmac-test-key"),
	}
	require.NoError(t, s.storage.CreateAuthRequest(ctx, authReq))

	// The dispatcher accepts only the "continue" and "approved" verifiers; a value
	// minted for a different step (or none) is rejected.
	wrongStepMAC := internal.ComputeHMAC(authReq.HMACKey, authReq.ID, "mfa")

	for _, tc := range []struct {
		name string
		path string
	}{
		{"dispatcher without hmac", "/auth?req=" + authReq.ID},
		{"dispatcher with wrong-step hmac", "/auth?req=" + authReq.ID + "&hmac=" + wrongStepMAC},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, tc.path, nil))
			require.Equal(t, http.StatusUnauthorized, rr.Code)
		})
	}

	// With no MFA configured and the "approved" verifier resolving consent, the
	// dispatcher issues the code to the client.
	rr := httptest.NewRecorder()
	approvedMAC := internal.ComputeHMAC(authReq.HMACKey, authReq.ID, "approved")
	s.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/auth?req="+authReq.ID+"&hmac="+approvedMAC, nil))
	require.Equal(t, http.StatusSeeOther, rr.Code)
	require.Contains(t, rr.Header().Get("Location"), "https://client.example/callback")
}

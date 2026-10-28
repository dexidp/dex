package device

import (
	"io"
	"log/slog"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/storage/memory"
)

func TestGetDeviceVerificationURI(t *testing.T) {
	u, err := url.Parse("https://dex.example.com/non-root-path")
	require.NoError(t, err)

	h := &Handler{IssuerURL: oauth2.IssuerURL{URL: *u}}
	require.Equal(t, "/non-root-path/device/auth/verify_code", h.getDeviceVerificationURI())
}

// TestCreateDeviceAuthorizationUsesInjectedClock pins the device request and
// token expiry to the handler's clock. Both were minted from a mix of
// time.Now() and h.Now(), which made the two expiries drift apart under a
// fixed test clock.
func TestCreateDeviceAuthorizationUsesInjectedClock(t *testing.T) {
	u, err := url.Parse("https://dex.example.com")
	require.NoError(t, err)

	fixed := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	h := &Handler{
		IssuerURL:        oauth2.IssuerURL{URL: *u},
		Storage:          memory.New(logger),
		Logger:           logger,
		Now:              func() time.Time { return fixed },
		RequestsValidFor: 5 * time.Minute,
	}

	resp, ferr := h.createDeviceAuthorization(t.Context(), deviceCodeRequest{
		clientID: "test",
		scopes:   []string{"openid"},
	})
	require.Nil(t, ferr)

	want := fixed.Add(5 * time.Minute)

	req, err := h.Storage.GetDeviceRequest(t.Context(), resp.UserCode)
	require.NoError(t, err)
	require.WithinDuration(t, want, req.Expiry, 0)

	token, err := h.Storage.GetDeviceToken(t.Context(), resp.DeviceCode)
	require.NoError(t, err)
	require.WithinDuration(t, want, token.Expiry, 0)
	require.WithinDuration(t, fixed, token.LastRequestTime, 0)
}

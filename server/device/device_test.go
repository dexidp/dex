package device

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetDeviceVerificationURI(t *testing.T) {
	u, err := url.Parse("https://dex.example.com/non-root-path")
	require.NoError(t, err)

	h := &Handler{IssuerURL: *u}
	require.Equal(t, "/non-root-path/device/auth/verify_code", h.getDeviceVerificationURI())
}

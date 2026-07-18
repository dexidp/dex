package authflow

import (
	"net/url"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

// buildMFAURL builds the HMAC-protected URL of the MFA gate — the step the login
// handler hands off to once the user is authenticated.
func (h *Handler) buildMFAURL(authReq storage.AuthRequest) string {
	v := url.Values{}
	v.Set("req", authReq.ID)
	v.Set("hmac", internal.ComputeHMAC(authReq.HMACKey, authReq.ID, "mfa"))
	return h.AbsPath("/mfa/start") + "?" + v.Encode()
}

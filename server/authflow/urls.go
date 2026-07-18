package authflow

import (
	"net/url"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

// buildContinueURL builds the HMAC-protected URL that returns to the /auth
// dispatcher, used once login completes so the dispatcher can pick the next step.
func (h *Handler) buildContinueURL(authReq storage.AuthRequest) string {
	v := url.Values{}
	v.Set("req", authReq.ID)
	v.Set("hmac", internal.ComputeHMAC(authReq.HMACKey, authReq.ID, "continue"))
	return h.absPath("/auth") + "?" + v.Encode()
}

// buildApprovalURL builds the HMAC-protected URL of the consent screen, where the
// dispatcher sends the user when consent is required.
func (h *Handler) buildApprovalURL(authReq storage.AuthRequest) string {
	v := url.Values{}
	v.Set("req", authReq.ID)
	v.Set("hmac", internal.ComputeHMAC(authReq.HMACKey, authReq.ID, "approval"))
	return h.absPath("/approval") + "?" + v.Encode()
}

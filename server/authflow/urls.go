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
	return h.IssuerURL.AbsPath("/auth") + "?" + v.Encode()
}

// buildMFAURL builds the HMAC-protected URL of the MFA entry, where the
// dispatcher sends the user when the client requires MFA. MFA resolves the
// effective chain and picks the factor; the dispatcher only decides that MFA
// applies.
func (h *Handler) buildMFAURL(authReq storage.AuthRequest) string {
	v := url.Values{}
	v.Set("req", authReq.ID)
	v.Set("hmac", internal.ComputeHMAC(authReq.HMACKey, authReq.ID, "mfa"))
	return h.IssuerURL.AbsPath("/mfa") + "?" + v.Encode()
}

// buildApprovalURL builds the HMAC-protected URL of the consent screen, where the
// dispatcher sends the user when consent is required.
func (h *Handler) buildApprovalURL(authReq storage.AuthRequest) string {
	v := url.Values{}
	v.Set("req", authReq.ID)
	v.Set("hmac", internal.ComputeHMAC(authReq.HMACKey, authReq.ID, "approval"))
	return h.IssuerURL.AbsPath("/approval") + "?" + v.Encode()
}

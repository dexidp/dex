package authflow

import (
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

// buildContinueURL builds the HMAC-protected URL that returns to the /auth
// dispatcher, used once login completes so the dispatcher can pick the next step.
func (h *Handler) buildContinueURL(authReq storage.AuthRequest) string {
	return internal.StepURL(h.IssuerURL.AbsPath("/auth"), authReq, internal.StepContinue, nil)
}

// buildMFAURL builds the HMAC-protected URL of the MFA entry, where the
// dispatcher sends the user when the client requires MFA. MFA resolves the
// effective chain and picks the factor; the dispatcher only decides that MFA
// applies.
func (h *Handler) buildMFAURL(authReq storage.AuthRequest) string {
	return internal.StepURL(h.IssuerURL.AbsPath("/mfa"), authReq, internal.StepMFA, nil)
}

// buildApprovalURL builds the HMAC-protected URL of the consent screen, where the
// dispatcher sends the user when consent is required.
func (h *Handler) buildApprovalURL(authReq storage.AuthRequest) string {
	return internal.StepURL(h.IssuerURL.AbsPath("/approval"), authReq, internal.StepApproval, nil)
}

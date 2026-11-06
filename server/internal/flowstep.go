package internal

import (
	"maps"
	"net/url"

	"github.com/dexidp/dex/storage"
)

// Step names one hop of the interactive login flow. Every redirect between the
// /auth dispatcher and a step (MFA, consent) carries an HMAC over the auth
// request ID and the step it is good for, so a verifier minted for one hop
// cannot be replayed against another.
//
// The labels are namespaced. Authenticator IDs come from the operator's config
// and are signed the same way, so without the prefixes an authenticator named
// "approved" would hand the user a valid consent verifier and let them skip the
// approval screen.
type Step string

const (
	// StepContinue returns to the /auth dispatcher so it can pick the next step.
	// Login and every completed MFA factor use it.
	StepContinue Step = "step:continue"
	// StepMFA enters the MFA handler, which resolves the chain and picks a factor.
	StepMFA Step = "step:mfa"
	// StepApproval enters the consent screen.
	StepApproval Step = "step:approval"
	// StepApproved returns to the dispatcher from the consent screen and proves
	// the user just approved, resolving consent for this request.
	StepApproved Step = "step:approved"
)

// Authenticator returns the step of one MFA factor. The id comes from the
// server's MFA config, so it is namespaced away from the flow's own steps.
func Authenticator(id string) Step { return Step("authenticator:" + id) }

// SignStep returns the verifier proving a redirect to step belongs to authReq.
func SignStep(authReq storage.AuthRequest, step Step) string {
	return ComputeHMAC(authReq.HMACKey, authReq.ID, string(step))
}

// VerifyStep reports whether mac is the verifier for step on authReq.
func VerifyStep(authReq storage.AuthRequest, mac string, step Step) bool {
	return VerifyHMAC(authReq.HMACKey, mac, authReq.ID, string(step))
}

// StepURL builds the URL of a flow step under base, carrying the auth request
// ID and its verifier. extra adds step-specific query parameters and may be nil.
func StepURL(base string, authReq storage.AuthRequest, step Step, extra url.Values) string {
	v := url.Values{}
	maps.Copy(v, extra)
	v.Set("req", authReq.ID)
	v.Set("hmac", SignStep(authReq, step))
	return base + "?" + v.Encode()
}

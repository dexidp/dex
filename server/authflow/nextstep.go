package authflow

import (
	"context"
	"net/http"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/storage"
)

// advance runs the auth request's next step: it redirects to the next MFA factor
// or the consent screen, or issues the authorization code. It is the single
// place the post-identity transition lives — connector login, session login,
// approval and MFA completion all funnel through it instead of each deciding
// where to go and building the URL themselves.
func (h *Handler) advance(w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest) {
	step, err := h.nextAuthStep(r.Context(), &authReq)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to determine next auth step", "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, ErrMsgInternalServerError)
		return
	}
	// prompt=none forbids any user-facing interaction: if the flow can't complete
	// silently, report interaction_required instead of showing MFA or consent.
	if prompt, _ := oauth2.ParsePrompt(authReq.Prompt); prompt.None() {
		if _, ok := step.(issueStep); !ok {
			h.redirectWithError(w, r, &authReq, oauth2.InteractionRequired, "User interaction required")
			return
		}
	}

	switch step := step.(type) {
	case mfaStep:
		http.Redirect(w, r, h.mfa.BuildRedirectURL(authReq, step.authenticator), http.StatusSeeOther)
	case approvalStep:
		http.Redirect(w, r, h.BuildApprovalURL(authReq), http.StatusSeeOther)
	case issueStep:
		h.sendCodeResponse(w, r, authReq)
	}
}

// authStep is the next thing a logged-in authorization request needs before the
// code can be issued. It is data, not behavior — the whole "what's next"
// decision lives in nextAuthStep (one ordered, testable place, in the spirit of
// zitadel's nextSteps), and the handlers dispatch on the concrete type.
type authStep interface{ authStep() }

// mfaStep means the user must satisfy the given MFA authenticator next.
type mfaStep struct{ authenticator string }

// approvalStep means the consent screen must be shown.
type approvalStep struct{}

// issueStep means every gate is satisfied and the authorization code (or token)
// can be issued.
type issueStep struct{}

func (mfaStep) authStep()      {}
func (approvalStep) authStep() {}
func (issueStep) authStep()    {}

// nextAuthStep decides what a logged-in auth request needs next: an MFA factor,
// the consent screen, or issuing the code. The ordering — MFA chain, then
// consent — lives here so the login, session-login and approval handlers share
// exactly one decision instead of each re-deriving it.
func (h *Handler) nextAuthStep(ctx context.Context, authReq *storage.AuthRequest) (authStep, error) {
	chain, err := h.mfa.ChainForClient(ctx, authReq.ClientID, authReq.ConnectorID)
	if err != nil {
		return nil, err
	}
	if len(chain) > 0 && !authReq.MFAValidated {
		return mfaStep{authenticator: chain[0]}, nil
	}
	if h.consentSatisfied(ctx, authReq) {
		return issueStep{}, nil
	}
	return approvalStep{}, nil
}

// consentSatisfied reports whether the approval screen can be skipped: the client
// did not force it, and either approval is disabled server-wide or the user has
// already consented to the requested scopes for this client.
func (h *Handler) consentSatisfied(ctx context.Context, authReq *storage.AuthRequest) bool {
	if authReq.ForceApprovalPrompt {
		return false
	}
	if h.skipApproval {
		return true
	}
	ui, err := h.storage.GetUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID)
	return err == nil && scopesCoveredByConsent(ui.Consents[authReq.ClientID], authReq.Scopes)
}

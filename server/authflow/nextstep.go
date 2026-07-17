package authflow

import (
	"context"
	"net/http"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/storage"
)

// authStep is one outcome of the post-login decision. Each step knows how to run
// itself against the flow, so advance just runs whatever nextAuthStep returns
// (in the spirit of zitadel's NextSteps) instead of re-deriving the transition.
type authStep interface {
	// run carries the flow forward: redirect to the next MFA factor or the
	// consent screen, or issue the authorization response.
	run(h *Handler, w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest)
}

// mfaStep sends the user to the next MFA authenticator in the chain.
type mfaStep struct{ authenticator string }

func (s mfaStep) run(h *Handler, w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest) {
	http.Redirect(w, r, h.mfa.BuildRedirectURL(authReq, s.authenticator), http.StatusSeeOther)
}

// approvalStep shows the consent screen.
type approvalStep struct{}

func (approvalStep) run(h *Handler, w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest) {
	http.Redirect(w, r, h.BuildApprovalURL(authReq), http.StatusSeeOther)
}

// issueStep issues the authorization code (or token) response. It is the only
// step that completes without user interaction.
type issueStep struct{}

func (issueStep) run(h *Handler, w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest) {
	h.sendCodeResponse(w, r, authReq)
}

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

// advance runs the auth request's next step. It is the single place the
// post-identity transition lives — connector login, session login, approval and
// MFA completion all funnel through it instead of each deciding where to go.
func (h *Handler) advance(w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest) {
	step, err := h.nextAuthStep(r.Context(), &authReq)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to determine next auth step", "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, ErrMsgInternalServerError)
		return
	}

	// prompt=none forbids any user-facing interaction: only a silent issue is
	// allowed; MFA or consent means interaction_required.
	if prompt, _ := oauth2.ParsePrompt(authReq.Prompt); prompt.None() {
		if _, ok := step.(issueStep); !ok {
			h.redirectWithError(w, r, &authReq, oauth2.InteractionRequired, "User interaction required")
			return
		}
	}

	step.run(h, w, r, authReq)
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

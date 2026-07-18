package authflow

// continue.go is the flow dispatcher: the central point the authorization flow
// converges on. Login and every interactive step (MFA factor, consent) redirect
// back here; the dispatcher inspects the auth request and decides the next step
// — another MFA factor, the consent screen, or issuing the response — in one
// place. This mirrors the authorize endpoint in hydra and the finalize step in
// zitadel: the steps hold no reference to one another, only to this hub.

import (
	"net/http"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/storage"
)

func (h *Handler) handleContinue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mac := r.FormValue("hmac")
	if mac == "" {
		h.RenderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}
	authReq, err := h.storage.GetAuthRequest(ctx, r.FormValue("req"))
	if err != nil {
		if err == storage.ErrNotFound {
			h.RenderError(r, w, http.StatusBadRequest, "User session error.")
			return
		}
		h.logger.ErrorContext(ctx, "failed to get auth request", "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}
	if !authReq.LoggedIn {
		h.logger.ErrorContext(ctx, "flow dispatcher reached for auth request without an identity")
		h.RenderError(r, w, http.StatusInternalServerError, "Login process not yet finalized.")
		return
	}
	if !internal.VerifyHMAC(authReq.HMACKey, mac, authReq.ID, "continue") {
		h.RenderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}

	h.dispatch(w, r, authReq)
}

// dispatch runs the auth request's next step. It is the single place the
// post-identity decision lives: MFA chain, then consent, then issuance. Each
// user-facing step is forbidden when prompt=none, which only allows a silent
// issue.
func (h *Handler) dispatch(w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest) {
	ctx := r.Context()
	prompt, _ := oauth2.ParsePrompt(authReq.Prompt)

	chain, err := h.mfa.ChainForClient(ctx, authReq.ClientID, authReq.ConnectorID)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to determine MFA chain", "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, ErrMsgInternalServerError)
		return
	}
	if len(chain) > 0 && !authReq.MFAValidated {
		if prompt.None() {
			h.RedirectAuthError(w, r, authReq, oauth2.InteractionRequired, "User interaction required")
			return
		}
		http.Redirect(w, r, h.mfa.BuildRedirectURL(authReq, chain[0]), http.StatusSeeOther)
		return
	}

	if !h.consent.Satisfied(ctx, &authReq) {
		if prompt.None() {
			h.RedirectAuthError(w, r, authReq, oauth2.InteractionRequired, "User interaction required")
			return
		}
		http.Redirect(w, r, h.BuildApprovalURL(authReq), http.StatusSeeOther)
		return
	}

	// Fully authorized — issue the response inline.
	h.issue.WriteResponse(w, r, authReq)
}

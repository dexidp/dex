package authflow

// dispatch.go is the /auth flow dispatcher. After login and after every step,
// the browser re-enters /auth carrying an HMAC verifier; the dispatcher inspects
// the auth request and decides the next step — an MFA factor, the consent screen,
// or issuing the response. This mirrors hydra's authorize strategy, which routes
// by which verifier (login/consent) is present. The steps only ever redirect
// back here, never to one another.

import (
	"context"
	"net/http"

	"github.com/dexidp/dex/server/consent"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/storage"
)

func (h *Handler) handleContinue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mac := r.FormValue("hmac")
	if mac == "" {
		h.renderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}
	authReq, err := h.Storage.GetAuthRequest(ctx, r.FormValue("req"))
	if err != nil {
		if err == storage.ErrNotFound {
			h.renderError(r, w, http.StatusBadRequest, "User session error.")
			return
		}
		h.Logger.ErrorContext(ctx, "failed to get auth request", "err", err)
		h.renderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}
	if !authReq.LoggedIn {
		h.Logger.ErrorContext(ctx, "flow dispatcher reached for auth request without an identity")
		h.renderError(r, w, http.StatusInternalServerError, "Login process not yet finalized.")
		return
	}

	// A step returns with the "continue" verifier (login, MFA) or the "approved"
	// verifier (consent). The latter proves the user just approved, so consent is
	// resolved for this request even under prompt=consent / ForceApprovalPrompt.
	consentApproved := internal.VerifyHMAC(authReq.HMACKey, mac, authReq.ID, "approved")
	if !consentApproved && !internal.VerifyHMAC(authReq.HMACKey, mac, authReq.ID, "continue") {
		h.renderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}

	h.dispatch(w, r, authReq, consentApproved)
}

// dispatch runs the auth request's next step. It is the single place the
// post-identity decision lives: MFA, then consent, then issuance. Each
// user-facing step is forbidden under prompt=none, which allows only a silent
// issue.
func (h *Handler) dispatch(w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest, consentApproved bool) {
	ctx := r.Context()
	prompt, _ := oauth2.ParsePrompt(authReq.Prompt)

	// MFA: if the client requires it and it is not yet satisfied, hand off to the
	// MFA entry, which resolves the effective chain and picks the factor. The
	// dispatcher only decides that MFA applies — the requested chain is client
	// state (client.MFAChain, else the server default), not a query into MFA.
	if h.MFAEnabled && !authReq.MFAValidated {
		required, err := h.mfaRequired(ctx, authReq.ClientID)
		if err != nil {
			h.Logger.ErrorContext(ctx, "failed to determine MFA requirement", "err", err)
			h.renderError(r, w, http.StatusInternalServerError, ErrMsgInternalServerError)
			return
		}
		if required {
			if prompt.None() {
				h.redirectWithError(w, r, &authReq, oauth2.InteractionRequired, "User interaction required")
				return
			}
			http.Redirect(w, r, h.buildMFAURL(authReq), http.StatusSeeOther)
			return
		}
	}

	// Consent: the "approved" verifier resolves it for this request; otherwise ask
	// whether it can be skipped from persisted state.
	if !consentApproved && !consent.Satisfied(ctx, h.Storage, h.SkipApproval, &authReq) {
		if prompt.None() {
			h.redirectWithError(w, r, &authReq, oauth2.InteractionRequired, "User interaction required")
			return
		}
		http.Redirect(w, r, h.buildApprovalURL(authReq), http.StatusSeeOther)
		return
	}

	// Fully authorized — issue the response.
	h.writeResponse(w, r, authReq)
}

// mfaRequired reports whether the client requests any MFA — its own chain, or the
// server default when the client sets none. This is only the dispatcher's cheap
// gate; the MFA entry does the precise, provider-aware resolution and, when
// nothing applies, records MFA as satisfied so control does not return here.
func (h *Handler) mfaRequired(ctx context.Context, clientID string) (bool, error) {
	client, err := h.Storage.GetClient(ctx, clientID)
	if err != nil {
		return false, err
	}
	chain := client.MFAChain
	if chain == nil {
		chain = h.DefaultMFAChain
	}
	return len(chain) > 0, nil
}

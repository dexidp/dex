package authflow

// approval.go handles the consent/approval step and builds the authorization-code
// or token response back to the client.

import (
	"net/http"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

func (h *Handler) handleApproval(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	macEncoded := r.FormValue("hmac")
	if macEncoded == "" {
		h.RenderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}
	authReq, err := h.storage.GetAuthRequest(ctx, r.FormValue("req"))
	if err != nil {
		if err == storage.ErrNotFound {
			h.RenderError(r, w, http.StatusBadRequest, "User session error.")
			return
		}
		h.logger.ErrorContext(r.Context(), "failed to get auth request", "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}
	if !authReq.LoggedIn {
		h.logger.ErrorContext(r.Context(), "auth request does not have an identity for approval")
		h.RenderError(r, w, http.StatusInternalServerError, "Login process not yet finalized.")
		return
	}

	if !internal.VerifyHMAC(authReq.HMACKey, macEncoded, authReq.ID, "") {
		h.RenderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}

	step, err := h.nextAuthStep(ctx, &authReq)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to determine next auth step", "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}
	// MFA may have been enabled mid-flow: send the user through it before consent.
	if _, ok := step.(mfaStep); ok {
		step.run(h, w, r, authReq)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Consent already satisfied → issue the code directly; otherwise the user
		// lands here after MFA and shouldn't see the consent screen again.
		if _, ok := step.(issueStep); ok {
			step.run(h, w, r, authReq)
			return
		}

		client, err := h.storage.GetClient(ctx, authReq.ClientID)
		if err != nil {
			h.logger.ErrorContext(r.Context(), "Failed to get client", "client_id", authReq.ClientID, "err", err)
			h.RenderError(r, w, http.StatusInternalServerError, "Failed to retrieve client.")
			return
		}
		if err := h.templates.Approval(r, w, authReq.ID, authReq.Claims.Username, client.Name, authReq.Scopes); err != nil {
			h.logger.ErrorContext(r.Context(), "server template error", "err", err)
		}
	case http.MethodPost:
		if r.FormValue("approval") != "approve" {
			h.RenderError(r, w, http.StatusInternalServerError, "Approval rejected.")
			return
		}
		// Persist user-approved scopes as consent for this client.
		if h.sessions.Enabled() {
			if err := h.storage.UpdateUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
				if old.Consents == nil {
					old.Consents = make(map[string][]string)
				}
				old.Consents[authReq.ClientID] = authReq.Scopes
				return old, nil
			}); err != nil {
				h.logger.ErrorContext(ctx, "failed to update user identity consents", "err", err)
			}
		}
		h.sendCodeResponse(w, r, authReq)
	}
}

func scopesCoveredByConsent(approved, requested []string) bool {
	approvedSet := make(map[string]struct{}, len(approved))
	for _, s := range approved {
		approvedSet[s] = struct{}{}
	}

	for _, scope := range requested {
		if scope == tokens.ScopeOpenID {
			continue
		}
		if _, ok := approvedSet[scope]; !ok {
			return false
		}
	}

	return true
}

// tokenErrHelper writes a JSON OAuth2 error response.

package consent

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/render"
	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/session"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// Handler owns the consent step. Reached from the MFA gate, it decides for
// itself whether consent is needed (Satisfied): if so it shows the approval
// screen and records the granted scopes, otherwise it hands off to issuance.
// Either way it moves the flow on by redirect and holds no reference to the
// other flow steps.
type Handler struct {
	*render.UI

	Storage      storage.Storage
	Templates    *templates.Templates
	Logger       *slog.Logger
	Sessions     *session.Manager
	SkipApproval bool
}

// Mount registers the consent endpoint.
func (h *Handler) Mount(mux router.Mux) {
	mux.HandleFunc("/approval", h.handleApproval)
}

// buildIssueURL builds the HMAC-protected URL that re-enters the authorize
// endpoint to issue the response — the step consent hands off to.
func (h *Handler) buildIssueURL(authReq storage.AuthRequest) string {
	v := url.Values{}
	v.Set("req", authReq.ID)
	v.Set("hmac", internal.ComputeHMAC(authReq.HMACKey, authReq.ID, "issue"))
	return h.AbsPath("/auth") + "?" + v.Encode()
}

// Satisfied reports whether the approval screen can be skipped: the client did
// not force it, and either approval is disabled server-wide or the user has
// already consented to the requested scopes for this client.
func (h *Handler) Satisfied(ctx context.Context, authReq *storage.AuthRequest) bool {
	if authReq.ForceApprovalPrompt {
		return false
	}
	if h.SkipApproval {
		return true
	}
	ui, err := h.Storage.GetUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID)
	return err == nil && scopesCoveredByConsent(ui.Consents[authReq.ClientID], authReq.Scopes)
}

func (h *Handler) handleApproval(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	macEncoded := r.FormValue("hmac")
	if macEncoded == "" {
		h.RenderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}
	authReq, err := h.Storage.GetAuthRequest(ctx, r.FormValue("req"))
	if err != nil {
		if err == storage.ErrNotFound {
			h.RenderError(r, w, http.StatusBadRequest, "User session error.")
			return
		}
		h.Logger.ErrorContext(ctx, "failed to get auth request", "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}
	if !authReq.LoggedIn {
		h.Logger.ErrorContext(ctx, "auth request does not have an identity for approval")
		h.RenderError(r, w, http.StatusInternalServerError, "Login process not yet finalized.")
		return
	}

	if !internal.VerifyHMAC(authReq.HMACKey, macEncoded, authReq.ID, "") {
		h.RenderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Consent already covered — go straight to issuance.
		if h.Satisfied(ctx, &authReq) {
			http.Redirect(w, r, h.buildIssueURL(authReq), http.StatusSeeOther)
			return
		}
		// Consent is required, but prompt=none forbids showing the screen.
		if prompt, _ := oauth2.ParsePrompt(authReq.Prompt); prompt.None() {
			h.RedirectAuthError(w, r, authReq, oauth2.InteractionRequired, "User interaction required")
			return
		}

		client, err := h.Storage.GetClient(ctx, authReq.ClientID)
		if err != nil {
			h.Logger.ErrorContext(ctx, "Failed to get client", "client_id", authReq.ClientID, "err", err)
			h.RenderError(r, w, http.StatusInternalServerError, "Failed to retrieve client.")
			return
		}
		if err := h.Templates.Approval(r, w, authReq.ID, authReq.Claims.Username, client.Name, authReq.Scopes); err != nil {
			h.Logger.ErrorContext(ctx, "server template error", "err", err)
		}
	case http.MethodPost:
		if r.FormValue("approval") != "approve" {
			h.RenderError(r, w, http.StatusInternalServerError, "Approval rejected.")
			return
		}
		// Persist user-approved scopes as consent for this client, then hand off to
		// issuance.
		if h.Sessions.Enabled() {
			if err := h.Storage.UpdateUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
				if old.Consents == nil {
					old.Consents = make(map[string][]string)
				}
				old.Consents[authReq.ClientID] = authReq.Scopes
				return old, nil
			}); err != nil {
				h.Logger.ErrorContext(ctx, "failed to update user identity consents", "err", err)
			}
		}
		http.Redirect(w, r, h.buildIssueURL(authReq), http.StatusSeeOther)
	}
}

// scopesCoveredByConsent checks whether the approved scopes cover all requested
// scopes. The openid scope is excluded from the comparison as it is a technical
// scope that does not require user consent.
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

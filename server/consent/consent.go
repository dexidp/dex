package consent

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"path"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/session"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// Handler owns the consent step. The /auth dispatcher decides whether consent is
// needed (via Satisfied) and, if so, routes to the approval screen here; on
// approve it records the granted scopes and returns to the dispatcher with the
// "approved" verifier. It holds no reference to the other flow steps.
type Handler struct {
	Storage      storage.Storage
	Templates    *templates.Templates
	Logger       *slog.Logger
	IssuerURL    url.URL
	Sessions     *session.Manager
	SkipApproval bool
}

// renderError renders a user-facing HTML error page.
func (h *Handler) renderError(r *http.Request, w http.ResponseWriter, status int, description string) {
	if err := h.Templates.Err(r, w, status, description); err != nil {
		h.Logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

// absPath returns the issuer path joined with the given path items.
func (h *Handler) absPath(pathItems ...string) string {
	return path.Join(append([]string{h.IssuerURL.Path}, pathItems...)...)
}

// Mount registers the consent endpoint.
func (h *Handler) Mount(mux router.Mux) {
	mux.HandleFunc("/approval", h.handleApproval)
}

// buildApprovedURL builds the HMAC-protected URL that returns to the authorize
// dispatcher (/auth) with the "approved" verifier, so the dispatcher knows the
// user consented and can issue.
func (h *Handler) buildApprovedURL(authReq storage.AuthRequest) string {
	v := url.Values{}
	v.Set("req", authReq.ID)
	v.Set("hmac", internal.ComputeHMAC(authReq.HMACKey, authReq.ID, "approved"))
	return h.absPath("/auth") + "?" + v.Encode()
}

// Satisfied reports whether the approval screen can be skipped: the client did
// not force it, and either approval is disabled server-wide or the user has
// already consented to the requested scopes for this client. It is a package
// function so the /auth dispatcher can decide consent from state without holding
// the consent Handler.
func Satisfied(ctx context.Context, store storage.Storage, skipApproval bool, authReq *storage.AuthRequest) bool {
	if authReq.ForceApprovalPrompt {
		return false
	}
	if skipApproval {
		return true
	}
	ui, err := store.GetUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID)
	return err == nil && scopesCoveredByConsent(ui.Consents[authReq.ClientID], authReq.Scopes)
}

func (h *Handler) handleApproval(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	macEncoded := r.FormValue("hmac")
	if macEncoded == "" {
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
		h.Logger.ErrorContext(ctx, "auth request does not have an identity for approval")
		h.renderError(r, w, http.StatusInternalServerError, "Login process not yet finalized.")
		return
	}

	if !internal.VerifyHMAC(authReq.HMACKey, macEncoded, authReq.ID, "approval") {
		h.renderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}

	switch r.Method {
	case http.MethodGet:
		// The dispatcher routes here only when consent is required, so just show
		// the approval screen.
		client, err := h.Storage.GetClient(ctx, authReq.ClientID)
		if err != nil {
			h.Logger.ErrorContext(ctx, "Failed to get client", "client_id", authReq.ClientID, "err", err)
			h.renderError(r, w, http.StatusInternalServerError, "Failed to retrieve client.")
			return
		}
		if err := h.Templates.Approval(r, w, authReq.ID, authReq.Claims.Username, client.Name, authReq.Scopes); err != nil {
			h.Logger.ErrorContext(ctx, "server template error", "err", err)
		}
	case http.MethodPost:
		if r.FormValue("approval") != "approve" {
			h.renderError(r, w, http.StatusInternalServerError, "Approval rejected.")
			return
		}
		// Persist the approved scopes so a future request skips consent, then return
		// to the dispatcher with the "approved" verifier.
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
		http.Redirect(w, r, h.buildApprovedURL(authReq), http.StatusSeeOther)
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

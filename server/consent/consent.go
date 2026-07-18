package consent

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/render"
	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/session"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// Manager owns the consent step of the login chain. It is reached from the MFA
// gate once the user is authenticated: it either skips consent or shows the
// approval screen, records the granted scopes, and hands off to the issue step
// by redirect. It holds no reference to the other flow steps.
type Manager struct {
	*render.UI

	Storage      storage.Storage
	Templates    *templates.Templates
	Logger       *slog.Logger
	Sessions     *session.Manager
	SkipApproval bool
}

// Mount registers the consent endpoint.
func (m *Manager) Mount(mux router.Mux) {
	mux.HandleFunc("/approval", m.handleApproval)
}

// Satisfied reports whether the approval screen can be skipped: the client did
// not force it, and either approval is disabled server-wide or the user has
// already consented to the requested scopes for this client.
func (m *Manager) Satisfied(ctx context.Context, authReq *storage.AuthRequest) bool {
	if authReq.ForceApprovalPrompt {
		return false
	}
	if m.SkipApproval {
		return true
	}
	ui, err := m.Storage.GetUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID)
	return err == nil && scopesCoveredByConsent(ui.Consents[authReq.ClientID], authReq.Scopes)
}

func (m *Manager) handleApproval(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	macEncoded := r.FormValue("hmac")
	if macEncoded == "" {
		m.RenderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}
	authReq, err := m.Storage.GetAuthRequest(ctx, r.FormValue("req"))
	if err != nil {
		if err == storage.ErrNotFound {
			m.RenderError(r, w, http.StatusBadRequest, "User session error.")
			return
		}
		m.Logger.ErrorContext(ctx, "failed to get auth request", "err", err)
		m.RenderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}
	if !authReq.LoggedIn {
		m.Logger.ErrorContext(ctx, "auth request does not have an identity for approval")
		m.RenderError(r, w, http.StatusInternalServerError, "Login process not yet finalized.")
		return
	}

	if !internal.VerifyHMAC(authReq.HMACKey, macEncoded, authReq.ID, "") {
		m.RenderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Consent already covered — hand off to issuance without showing the screen.
		if m.Satisfied(ctx, &authReq) {
			http.Redirect(w, r, m.BuildIssueURL(authReq), http.StatusSeeOther)
			return
		}
		// Consent is required, but prompt=none forbids showing the screen.
		if prompt, _ := oauth2.ParsePrompt(authReq.Prompt); prompt.None() {
			m.RedirectAuthError(w, r, authReq, oauth2.InteractionRequired, "User interaction required")
			return
		}

		client, err := m.Storage.GetClient(ctx, authReq.ClientID)
		if err != nil {
			m.Logger.ErrorContext(ctx, "Failed to get client", "client_id", authReq.ClientID, "err", err)
			m.RenderError(r, w, http.StatusInternalServerError, "Failed to retrieve client.")
			return
		}
		if err := m.Templates.Approval(r, w, authReq.ID, authReq.Claims.Username, client.Name, authReq.Scopes); err != nil {
			m.Logger.ErrorContext(ctx, "server template error", "err", err)
		}
	case http.MethodPost:
		if r.FormValue("approval") != "approve" {
			m.RenderError(r, w, http.StatusInternalServerError, "Approval rejected.")
			return
		}
		// Persist user-approved scopes as consent for this client.
		if m.Sessions.Enabled() {
			if err := m.Storage.UpdateUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
				if old.Consents == nil {
					old.Consents = make(map[string][]string)
				}
				old.Consents[authReq.ClientID] = authReq.Scopes
				return old, nil
			}); err != nil {
				m.Logger.ErrorContext(ctx, "failed to update user identity consents", "err", err)
			}
		}
		http.Redirect(w, r, m.BuildIssueURL(authReq), http.StatusSeeOther)
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

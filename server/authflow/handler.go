package authflow

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/consent"
	"github.com/dexidp/dex/server/mfa"
	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/session"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// Handler serves the interactive login flow (connector selection, connector and
// password login, the callback) and the /auth dispatcher that decides each next
// step and issues the response. The /auth endpoint is the flow dispatcher: it
// starts login, then on each return decides the next step (MFA factor, consent
// screen) or issues. It queries MFA and Consent for those decisions — like
// hydra's authorize strategy holding its managers — but the steps only redirect
// back to /auth, never to one another.
type Handler struct {
	IssuerURL              url.URL
	Connectors             *connectors.Cache
	Storage                storage.Storage
	Templates              *templates.Templates
	Signer                 signer.Signer
	Now                    func() time.Time
	Logger                 *slog.Logger
	AlwaysShowLogin        bool
	SupportedResponseTypes map[string]bool
	PKCE                   PKCEConfig
	AuthRequestsValidFor   time.Duration

	// Sessions owns the session cookie, SSO lookup and auth-session CRUD.
	Sessions *session.Manager
	// Issuer mints tokens for the authorization response (see response.go).
	Issuer *tokens.Issuer
	// MFA and Consent answer the dispatcher's "is this step needed" decisions.
	MFA     *mfa.Handler
	Consent *consent.Handler
}

// Mount registers the login routes. The /auth endpoint is both the entry
// (login) and the exit (issuance, see response.go). The mfa, consent and logout
// steps are mounted separately by the server.
func (h *Handler) Mount(m router.Mux) {
	m.HandleFunc("/auth", h.handleAuthorization)
	m.HandleFunc("/auth/{connector}", h.handleConnectorLogin)
	m.HandleFunc("/auth/{connector}/login", h.handlePasswordLogin)
	m.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Strip the X-Remote-* headers to prevent security issues on
		// misconfigured authproxy connector setups.
		for key := range r.Header {
			if strings.HasPrefix(strings.ToLower(key), "x-remote-") {
				r.Header.Del(key)
			}
		}
		h.handleConnectorCallback(w, r)
	})
	// For easier connector-specific web server configuration, e.g. for the
	// "authproxy" connector.
	m.HandleFunc("/callback/{connector}", h.handleConnectorCallback)
}

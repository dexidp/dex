package authflow

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/oauth2"
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
// starts login, then on each return decides the next step (MFA, consent) or
// issues. It decides those from persisted state and config alone — it holds no
// reference to the MFA or consent handlers; each step only redirects back to
// /auth, never to another step.
type Handler struct {
	IssuerURL              oauth2.IssuerURL
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

	// MFAEnabled reports whether any authenticator is configured; DefaultMFAChain
	// is the chain applied to clients that set none. Together they let the
	// dispatcher gate MFA without the MFA handler — see mfaRequired.
	MFAEnabled      bool
	DefaultMFAChain []string
	// SkipApproval disables the consent screen server-wide (see consent.Satisfied).
	SkipApproval bool
}

// Mount registers the login routes. The /auth endpoint is both the entry
// (login) and the exit (issuance, see response.go). The mfa, consent and logout
// steps are mounted separately by the server.
func (h *Handler) Mount(m router.Mux) {
	m.HandleFunc("/auth", h.handleAuthorization)
	m.HandleFunc("/auth/{connector}", h.handleConnectorLogin)
	m.HandleFunc("/auth/{connector}/login", h.handlePasswordLogin)
	// The bare /callback serves OAuth/OIDC redirects, where X-Remote-* never
	// belongs, so strip it: a client must not spoof the authproxy connector here.
	// The /callback/{connector} route is authproxy's own and passes them through.
	m.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		stripRemoteHeaders(r)
		h.handleConnectorCallback(w, r)
	})
	m.HandleFunc("/callback/{connector}", h.handleConnectorCallback)
}

// stripRemoteHeaders drops the X-Remote-* request headers the authproxy
// connector trusts, so they cannot be forged on a route that does not set them.
func stripRemoteHeaders(r *http.Request) {
	for key := range r.Header {
		if strings.HasPrefix(strings.ToLower(key), "x-remote-") {
			r.Header.Del(key)
		}
	}
}

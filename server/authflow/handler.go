package authflow

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dexidp/dex/server/authflow/mfa"
	"github.com/dexidp/dex/server/authflow/session"
	"github.com/dexidp/dex/server/authflow/web"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// Config holds everything the interactive auth flow depends on. It is the narrow
// contract between the top-level Server and this package: NewHandler copies these
// into a Handler, and the flow never reaches back into the Server.
type Config struct {
	IssuerURL              url.URL
	Connectors             *connectors.Cache
	Storage                storage.Storage
	Templates              *templates.Templates
	Signer                 signer.Signer
	Issuer                 *tokens.Issuer
	Now                    func() time.Time
	Logger                 *slog.Logger
	SkipApproval           bool
	AlwaysShowLogin        bool
	SupportedResponseTypes map[string]bool
	PKCE                   PKCEConfig
	AuthRequestsValidFor   time.Duration
	SessionConfig          *session.Config
	MFAProviders           map[string]mfa.Provider
	DefaultMFAChain        []string
}

// Handler serves the interactive authorization flow. It embeds web (shared
// browser rendering / URL building) and delegates the session and MFA domains
// to their own components; what remains on the Handler is HTTP orchestration.
type Handler struct {
	*web.UI

	connectors             *connectors.Cache
	storage                storage.Storage
	templates              *templates.Templates
	logger                 *slog.Logger
	issuer                 *tokens.Issuer
	signer                 signer.Signer
	issuerURL              url.URL
	pkce                   PKCEConfig
	supportedResponseTypes map[string]bool
	now                    func() time.Time
	skipApproval           bool
	alwaysShowLogin        bool
	authRequestsValidFor   time.Duration

	// sessions owns the session cookie, SSO lookup and auth-session CRUD.
	sessions *session.Manager
	// mfa owns the authenticator chain and the TOTP/WebAuthn endpoints.
	mfa *mfa.Manager
}

// NewHandler builds the interactive auth-flow handler from its configuration.
func NewHandler(c Config) *Handler {
	ui := &web.UI{Templates: c.Templates, IssuerURL: c.IssuerURL, Logger: c.Logger}
	sessions := &session.Manager{Storage: c.Storage, Config: c.SessionConfig, Now: c.Now, Logger: c.Logger, IssuerURL: c.IssuerURL}
	return &Handler{
		UI:                     ui,
		connectors:             c.Connectors,
		storage:                c.Storage,
		templates:              c.Templates,
		logger:                 c.Logger,
		issuer:                 c.Issuer,
		signer:                 c.Signer,
		issuerURL:              c.IssuerURL,
		pkce:                   c.PKCE,
		supportedResponseTypes: c.SupportedResponseTypes,
		now:                    c.Now,
		skipApproval:           c.SkipApproval,
		alwaysShowLogin:        c.AlwaysShowLogin,
		authRequestsValidFor:   c.AuthRequestsValidFor,
		sessions:               sessions,
		mfa:                    &mfa.Manager{UI: ui, Storage: c.Storage, Templates: c.Templates, Logger: c.Logger, MFAProviders: c.MFAProviders, DefaultMFAChain: c.DefaultMFAChain, Now: c.Now, Connectors: c.Connectors},
	}
}

// Mount registers the interactive auth-flow routes. The logout and MFA endpoints
// require sessions (they are only wired when a session config is present).
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
	m.HandleFunc("/approval", h.handleApproval)

	if !h.sessions.Enabled() {
		return
	}
	// Logout requires an active session.
	m.HandleFunc("/logout", h.handleLogout)
	m.HandleFunc("/logout/callback", h.handleLogoutCallback)
}

// MFA returns the MFA component so the server can mount its endpoints directly.
func (h *Handler) MFA() *mfa.Manager { return h.mfa }

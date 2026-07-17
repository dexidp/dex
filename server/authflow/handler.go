// Package authflow implements dex's interactive browser-facing authorization
// flow: the /auth authorization endpoint, connector and password login, the
// session (SSO) shortcut, MFA (TOTP and WebAuthn), the consent/approval screen,
// and RP-initiated logout.
//
// The flow is a state machine over a storage.AuthRequest. Two abstractions keep
// it honest:
//
//   - nextAuthStep (nextstep.go) is the single, data-oriented decision for what
//     a logged-in request needs next — an MFA factor, consent, or issuing the
//     code — in the spirit of zitadel's nextSteps. The handlers dispatch on its
//     typed result; they don't re-derive the decision.
//   - responseTypeHandler (approval.go) issues the authorization response, one
//     self-selecting handler per OAuth2 response_type, in the spirit of fosite's
//     AuthorizeEndpointHandler.
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
	issuerURL              url.URL
	templates              *templates.Templates
	logger                 *slog.Logger
	signer                 signer.Signer
	issuer                 *tokens.Issuer
	now                    func() time.Time
	skipApproval           bool
	alwaysShowLogin        bool
	supportedResponseTypes map[string]bool
	pkce                   PKCEConfig
	authRequestsValidFor   time.Duration

	// sessions owns the session cookie, SSO lookup and auth-session CRUD.
	sessions *session.Manager
	// mfa owns the authenticator chain and the TOTP/WebAuthn endpoints.
	mfa *mfa.Manager
}

// NewHandler builds the interactive auth-flow handler from its configuration.
func NewHandler(c Config) *Handler {
	ui := web.New(c.Templates, c.IssuerURL, c.Logger)
	return &Handler{
		UI:                     ui,
		connectors:             c.Connectors,
		storage:                c.Storage,
		issuerURL:              c.IssuerURL,
		templates:              c.Templates,
		logger:                 c.Logger,
		signer:                 c.Signer,
		issuer:                 c.Issuer,
		now:                    c.Now,
		skipApproval:           c.SkipApproval,
		alwaysShowLogin:        c.AlwaysShowLogin,
		supportedResponseTypes: c.SupportedResponseTypes,
		pkce:                   c.PKCE,
		authRequestsValidFor:   c.AuthRequestsValidFor,
		sessions:               session.New(c.Storage, c.SessionConfig, c.Now, c.Logger, c.IssuerURL),
		mfa:                    mfa.New(ui, c.Storage, c.Templates, c.Logger, c.MFAProviders, c.DefaultMFAChain, c.Now, c.Connectors),
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
	// The following endpoints require DEX_SESSIONS_ENABLED=true.
	m.HandleFunc("/logout", h.handleLogout)
	m.HandleFunc("/logout/callback", h.handleLogoutCallback)
	m.HandleFunc("/mfa/totp", h.mfa.HandleTOTP)
	m.HandleFunc("/mfa/webauthn", h.mfa.HandleWebAuthn)
	m.HandleFunc("/mfa/webauthn/register/begin", h.mfa.HandleWebAuthnRegisterBegin)
	m.HandleFunc("/mfa/webauthn/register/finish", h.mfa.HandleWebAuthnRegisterFinish)
	m.HandleFunc("/mfa/webauthn/login/begin", h.mfa.HandleWebAuthnLoginBegin)
	m.HandleFunc("/mfa/webauthn/login/finish", h.mfa.HandleWebAuthnLoginFinish)
}

// PKCEConfig holds PKCE (Proof Key for Code Exchange) settings.
type PKCEConfig struct {
	// If true, PKCE is required for all authorization code flows.
	Enforce bool
	// Supported code challenge methods. Defaults to ["S256", "plain"].
	CodeChallengeMethodsSupported []string
}

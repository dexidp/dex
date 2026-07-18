package authflow

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/render"
	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/session"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/storage"
)

// Config is the login flow's configuration. The login handler only authenticates
// the user and then redirects into the MFA gate; the remaining steps (mfa,
// consent, issue) are separate components the server mounts alongside it. Only
// the shared browser rendering (UI) and the session manager are injected here.
type Config struct {
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

	UI       *render.UI
	Sessions *session.Manager
}

// Handler serves the interactive login flow: connector selection, connector and
// password login, and the callback. Once the user is authenticated it redirects
// into the MFA gate; it holds no reference to the later flow steps.
type Handler struct {
	*render.UI

	connectors             *connectors.Cache
	storage                storage.Storage
	templates              *templates.Templates
	logger                 *slog.Logger
	signer                 signer.Signer
	issuerURL              url.URL
	pkce                   PKCEConfig
	supportedResponseTypes map[string]bool
	now                    func() time.Time
	alwaysShowLogin        bool
	authRequestsValidFor   time.Duration

	// sessions owns the session cookie, SSO lookup and auth-session CRUD.
	sessions *session.Manager
}

// NewHandler builds the login-flow handler from its configuration.
func NewHandler(c Config) *Handler {
	return &Handler{
		UI:                     c.UI,
		connectors:             c.Connectors,
		storage:                c.Storage,
		templates:              c.Templates,
		logger:                 c.Logger,
		signer:                 c.Signer,
		issuerURL:              c.IssuerURL,
		pkce:                   c.PKCE,
		supportedResponseTypes: c.SupportedResponseTypes,
		now:                    c.Now,
		alwaysShowLogin:        c.AlwaysShowLogin,
		authRequestsValidFor:   c.AuthRequestsValidFor,
		sessions:               c.Sessions,
	}
}

// Mount registers the login routes. The shared components (consent, mfa, logout)
// are mounted separately by the server, not through this handler.
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

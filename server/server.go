package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"sort"
	"sync/atomic"
	"time"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/dexidp/dex/pkg/featureflags"
	"github.com/dexidp/dex/server/authflow"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/consent"
	"github.com/dexidp/dex/server/device"
	"github.com/dexidp/dex/server/discovery"
	"github.com/dexidp/dex/server/grants"
	"github.com/dexidp/dex/server/home"
	"github.com/dexidp/dex/server/introspection"
	"github.com/dexidp/dex/server/logout"
	"github.com/dexidp/dex/server/mfa"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/session"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/server/userinfo"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/web"
)

// Config holds the server's configuration options.
//
// Multiple servers using the same storage are expected to be configured identically.
type Config struct {
	Issuer string

	// The backing persistence layer.
	Storage storage.Storage

	AllowedGrantTypes []string

	// Valid values are "code" to enable the code flow and "token" to enable the implicit
	// flow. If no response types are supplied this value defaults to "code".
	SupportedResponseTypes []string

	// Headers is a map of headers to be added to the all responses.
	Headers http.Header

	// Header to extract real ip from.
	RealIPHeader       string
	TrustedRealIPCIDRs []netip.Prefix

	// List of allowed origins for CORS requests on discovery, token and keys endpoint.
	// If none are indicated, CORS requests are disabled. Passing in "*" will allow any
	// domain.
	AllowedOrigins []string

	// List of allowed headers for CORS requests on discovery, token, and keys endpoint.
	AllowedHeaders []string

	// If enabled, the server won't prompt the user to approve authorization requests.
	// Logging in implies approval.
	SkipApprovalScreen bool

	// If enabled, the connectors selection page will always be shown even if there's only one
	AlwaysShowLoginScreen bool

	IDTokensValidFor       time.Duration // Defaults to 24 hours
	AuthRequestsValidFor   time.Duration // Defaults to 24 hours
	DeviceRequestsValidFor time.Duration // Defaults to 5 minutes

	// Refresh token expiration settings
	RefreshTokenPolicy *tokens.RefreshStrategy

	// If set, the server will use this connector to handle password grants
	PasswordConnector string

	// PKCE configuration
	PKCE authflow.PKCEConfig

	GCFrequency time.Duration // Defaults to 5 minutes

	// If specified, the server will use this function for determining time.
	Now func() time.Time

	Web WebConfig

	Logger *slog.Logger

	// Signer is used to sign tokens.
	Signer signer.Signer

	PrometheusRegistry *prometheus.Registry

	HealthChecker gosundheit.Health

	// If enabled, the server will continue starting even if some connectors fail to initialize.
	// This allows the server to operate with a subset of connectors if some are misconfigured.
	ContinueOnConnectorFailure bool

	// SessionConfig holds session settings. Nil when sessions are disabled.
	SessionConfig *session.Config

	// MFAProviders maps authenticator IDs to their provider implementations.
	MFAProviders map[string]mfa.Provider

	// DefaultMFAChain is applied to clients that don't specify their own mfaChain.
	DefaultMFAChain []string
}

// WebConfig holds the server's frontend templates and asset configuration.
type WebConfig struct {
	// A file path to static web assets.
	//
	// It is expected to contain the following directories:
	//
	//   * static - Static static served at "( issuer URL )/static".
	//   * templates - HTML templates controlled by dex.
	//   * themes/(theme) - Static static served at "( issuer URL )/theme".
	Dir string

	// Alternative way to programmatically configure static web assets.
	// If Dir is specified, WebFS is ignored.
	// It's expected to contain the same files and directories as mentioned above.
	//
	// Note: this is experimental. Might get removed without notice!
	WebFS fs.FS

	// Defaults to "( issuer URL )/theme/logo.png"
	LogoURL string

	// Defaults to "dex"
	Issuer string

	// Defaults to "light"
	Theme string

	// Map of extra values passed into the templates
	Extra map[string]string
}

func value(val, defaultValue time.Duration) time.Duration {
	if val == 0 {
		return defaultValue
	}
	return val
}

// Server is the top level object.
type Server struct {
	issuerURL oauth2.IssuerURL

	// In-memory cache of opened connectors.
	connectors *connectors.Cache

	storage storage.Storage

	mux http.Handler

	templates *templates.Templates

	logger *slog.Logger

	// issuer turns an Authorization into a TokenSet.
	issuer *tokens.Issuer

	// discovery is built once from config and shared by the mounted HTTP handler
	// and the gRPC API's Discovery accessor.
	discovery *discovery.Handler

	sessionConfig *session.Config
}

// Connectors is the server's connector cache. The gRPC API needs it to
// invalidate the cache on connector CRUD.
func (s *Server) Connectors() *connectors.Cache { return s.connectors }

// Discovery is the handler that builds the OIDC discovery document. The gRPC
// API serves the same handler that is mounted for HTTP, so both return an
// identical document.
func (s *Server) Discovery() *discovery.Handler { return s.discovery }

// NewServer constructs a server from the provided config.
func NewServer(ctx context.Context, c Config) (*Server, error) {
	return newServer(ctx, c)
}

func newServer(ctx context.Context, c Config) (*Server, error) {
	issuerURL, err := url.Parse(c.Issuer)
	if err != nil {
		return nil, fmt.Errorf("server: can't parse issuer URL")
	}

	if c.Storage == nil {
		return nil, errors.New("server: storage cannot be nil")
	}

	if len(c.SupportedResponseTypes) == 0 {
		c.SupportedResponseTypes = []string{oauth2.ResponseTypeCode}
	}
	if len(c.AllowedHeaders) == 0 {
		c.AllowedHeaders = []string{"Authorization"}
	}

	supportedChallengeMethods := map[string]bool{
		oauth2.PKCEMethodS256:  true,
		oauth2.PKCEMethodPlain: true,
	}
	if len(c.PKCE.CodeChallengeMethodsSupported) == 0 {
		c.PKCE.CodeChallengeMethodsSupported = []string{oauth2.PKCEMethodS256, oauth2.PKCEMethodPlain}
	}
	for _, m := range c.PKCE.CodeChallengeMethodsSupported {
		if !supportedChallengeMethods[m] {
			return nil, fmt.Errorf("unsupported PKCE challenge method %q", m)
		}
	}

	allSupportedGrants := map[string]bool{
		oauth2.GrantTypeAuthorizationCode: true,
		oauth2.GrantTypeRefreshToken:      true,
		oauth2.GrantTypeDeviceCode:        true,
		oauth2.GrantTypeTokenExchange:     true,
	}
	supportedRes := make(map[string]bool)

	for _, respType := range c.SupportedResponseTypes {
		switch respType {
		case oauth2.ResponseTypeCode, oauth2.ResponseTypeIDToken, oauth2.ResponseTypeCodeIDToken:
			// continue
		case oauth2.ResponseTypeToken, oauth2.ResponseTypeCodeToken, oauth2.ResponseTypeIDTokenToken, oauth2.ResponseTypeCodeIDTokenToken:
			// response_type=token is an implicit flow, let's add it to the discovery info
			// https://datatracker.ietf.org/doc/html/rfc6749#section-4.2.1
			allSupportedGrants[oauth2.GrantTypeImplicit] = true
		default:
			return nil, fmt.Errorf("unsupported response_type %q", respType)
		}
		supportedRes[respType] = true
	}

	if c.PasswordConnector != "" {
		allSupportedGrants[oauth2.GrantTypePassword] = true
	}

	allSupportedGrants[oauth2.GrantTypeClientCredentials] = true

	var supportedGrants []string
	if len(c.AllowedGrantTypes) > 0 {
		for _, grant := range c.AllowedGrantTypes {
			if allSupportedGrants[grant] {
				supportedGrants = append(supportedGrants, grant)
			}
		}
	} else {
		for grant := range allSupportedGrants {
			supportedGrants = append(supportedGrants, grant)
		}
	}
	sort.Strings(supportedGrants)

	webFS := web.FS()
	if c.Web.Dir != "" {
		webFS = os.DirFS(c.Web.Dir)
	} else if c.Web.WebFS != nil {
		webFS = c.Web.WebFS
	}

	web := templates.Config{
		WebFS:     webFS,
		LogoURL:   c.Web.LogoURL,
		IssuerURL: c.Issuer,
		Issuer:    c.Web.Issuer,
		Theme:     c.Web.Theme,
		Extra:     c.Web.Extra,
	}

	static, theme, robots, tmpls, err := templates.LoadWebConfig(web)
	if err != nil {
		return nil, fmt.Errorf("server: failed to load web static: %v", err)
	}

	now := c.Now
	if now == nil {
		now = time.Now
	}

	authRequestsValidFor := value(c.AuthRequestsValidFor, 24*time.Hour)
	deviceRequestsValidFor := value(c.DeviceRequestsValidFor, 5*time.Minute)
	idTokensValidFor := value(c.IDTokensValidFor, 24*time.Hour)

	s := &Server{
		issuerURL:     oauth2.IssuerURL{URL: *issuerURL},
		storage:       newKeyCacher(c.Storage, now),
		templates:     tmpls,
		logger:        c.Logger,
		sessionConfig: c.SessionConfig,
	}
	s.issuer = tokens.NewIssuer(s.storage, c.Signer, s.issuerURL.URL, idTokensValidFor, now, s.logger)
	s.connectors = connectors.NewCache(s.storage, connectorResolver(s.storage, s.logger))
	// Build the discovery handler once from config; both the mounted HTTP route
	// and the gRPC API (via Discovery) serve this same handler.
	s.discovery = &discovery.Handler{
		Issuer:          s.issuerURL.String(),
		AbsURL:          s.issuerURL.AbsURL,
		RenderError:     s.renderError,
		Signer:          c.Signer,
		Logger:          s.logger,
		ResponseTypes:   supportedRes,
		GrantTypes:      supportedGrants,
		PKCEMethods:     c.PKCE.CodeChallengeMethodsSupported,
		SessionsEnabled: s.sessionConfig != nil,
	}
	// sessions is shared infrastructure (session cookie, SSO, auth-session CRUD)
	// referenced by the flow steps mounted below. The steps themselves hold no
	// reference to one another; the /auth dispatcher decides MFA and consent from
	// persisted state and config, so mfa and consent are mounted inline like the
	// rest.
	sessions := &session.Manager{
		Storage:   s.storage,
		Config:    s.sessionConfig,
		Now:       now,
		Logger:    s.logger,
		IssuerURL: s.issuerURL,
	}

	// Retrieves connector objects in backend storage. This list includes the static connectors
	// defined in the ConfigMap and dynamic connectors retrieved from the storage.
	storageConnectors, err := c.Storage.ListConnectors(ctx)
	if err != nil {
		return nil, fmt.Errorf("server: failed to list connector objects from storage: %v", err)
	}

	if len(storageConnectors) == 0 && s.connectors.Len() == 0 {
		return nil, errors.New("server: no connectors specified")
	}

	var failedCount int
	for _, conn := range storageConnectors {
		if _, err := s.connectors.Open(conn); err != nil {
			failedCount++
			if c.ContinueOnConnectorFailure {
				s.logger.Error("server: Failed to open connector", "id", conn.ID, "err", err)
				continue
			}
			return nil, fmt.Errorf("server: Failed to open connector %s: %v", conn.ID, err)
		}
	}

	if c.ContinueOnConnectorFailure && failedCount == len(storageConnectors) {
		return nil, fmt.Errorf("server: failed to open all connectors (%d/%d)", failedCount, len(storageConnectors))
	}

	if featureflags.SessionsEnabled.Enabled() {
		s.logger.InfoContext(ctx, "sessions feature flag is enabled")
	}

	instrumentHandler := func(_ string, handler http.Handler) http.HandlerFunc {
		return handler.ServeHTTP
	}

	if c.PrometheusRegistry != nil {
		requestCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Count of all HTTP requests.",
		}, []string{"code", "method", "handler"})

		durationHist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "A histogram of latencies for requests.",
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10},
		}, []string{"code", "method", "handler"})

		sizeHist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "response_size_bytes",
			Help:    "A histogram of response sizes for requests.",
			Buckets: []float64{200, 500, 900, 1500},
		}, []string{"code", "method", "handler"})

		c.PrometheusRegistry.MustRegister(requestCounter, durationHist, sizeHist)

		instrumentHandler = func(handlerName string, handler http.Handler) http.HandlerFunc {
			return promhttp.InstrumentHandlerDuration(durationHist.MustCurryWith(prometheus.Labels{"handler": handlerName}),
				promhttp.InstrumentHandlerCounter(requestCounter.MustCurryWith(prometheus.Labels{"handler": handlerName}),
					promhttp.InstrumentHandlerResponseSize(sizeHist.MustCurryWith(prometheus.Labels{"handler": handlerName}), handler),
				),
			)
		}
	}

	parseRealIP := func(r *http.Request) (string, error) {
		remoteAddr, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return "", err
		}

		remoteIP, err := netip.ParseAddr(remoteAddr)
		if err != nil {
			return "", err
		}

		for _, n := range c.TrustedRealIPCIDRs {
			if !n.Contains(remoteIP) {
				return remoteAddr, nil // Fallback to the address from the request if the header is provided
			}
		}

		ipVal := r.Header.Get(c.RealIPHeader)
		if ipVal != "" {
			ip, err := netip.ParseAddr(ipVal)
			if err == nil {
				return ip.String(), nil
			}
		}

		return remoteAddr, nil
	}

	r := mux.NewRouter().SkipClean(true).UseEncodedPath()
	r.NotFoundHandler = http.NotFoundHandler()

	// Self-contained domains mount their own routes through the router.Mux
	// abstraction; this is the only place they are wired in.
	routes := router.New(router.Config{
		Router:       r,
		IssuerPath:   issuerURL.Path,
		Headers:      c.Headers,
		RealIPHeader: c.RealIPHeader,
		Instrument:   instrumentHandler,
		RealIP:       parseRealIP,
		CORSOrigins:  c.AllowedOrigins,
		CORSHeaders:  c.AllowedHeaders,
	})
	for _, h := range []router.Handler{
		s.discovery,
		&grants.Handler{
			Issuer:              s.issuer,
			Storage:             s.storage,
			Connectors:          s.connectors,
			Now:                 now,
			Logger:              s.logger,
			PasswordConnector:   c.PasswordConnector,
			RefreshPolicy:       c.RefreshTokenPolicy,
			SessionsEnabled:     s.sessionConfig != nil,
			SupportedGrantTypes: supportedGrants,
		},
		&userinfo.Handler{
			Issuer: s.issuerURL.String(),
			Signer: c.Signer,
			Logger: s.logger,
		},
		&introspection.Handler{
			Issuer:        s.issuerURL.String(),
			Signer:        c.Signer,
			Storage:       s.storage,
			Logger:        s.logger,
			RefreshPolicy: c.RefreshTokenPolicy,
		},
		&device.Handler{
			IssuerURL:        s.issuerURL,
			Storage:          s.storage,
			Templates:        s.templates,
			Now:              now,
			RequestsValidFor: deviceRequestsValidFor,
			Logger:           s.logger,
			Issuer:           s.issuer,
			Connectors:       s.connectors,
		},
		&home.Handler{
			IssuerURL: s.issuerURL,
			Storage:   s.storage,
			Templates: s.templates,
			Logger:    s.logger,
			Sessions:  sessions,
		},
		&authflow.Handler{
			IssuerURL:              s.issuerURL,
			Connectors:             s.connectors,
			Storage:                s.storage,
			Templates:              s.templates,
			Signer:                 c.Signer,
			Now:                    now,
			Logger:                 s.logger,
			AlwaysShowLogin:        c.AlwaysShowLoginScreen,
			SupportedResponseTypes: supportedRes,
			PKCE:                   c.PKCE,
			AuthRequestsValidFor:   authRequestsValidFor,
			Sessions:               sessions,
			Issuer:                 s.issuer,
			MFAEnabled:             len(c.MFAProviders) > 0,
			DefaultMFAChain:        c.DefaultMFAChain,
			SkipApproval:           c.SkipApprovalScreen,
		},
		&mfa.Handler{
			Storage:         s.storage,
			Templates:       s.templates,
			Logger:          s.logger,
			IssuerURL:       s.issuerURL,
			MFAProviders:    c.MFAProviders,
			DefaultMFAChain: c.DefaultMFAChain,
			Now:             now,
			Connectors:      s.connectors,
		},
		&consent.Handler{
			Storage:      s.storage,
			Templates:    s.templates,
			Logger:       s.logger,
			IssuerURL:    s.issuerURL,
			Sessions:     sessions,
			SkipApproval: c.SkipApprovalScreen,
		},
		&logout.Handler{
			Storage:    s.storage,
			Templates:  s.templates,
			Logger:     s.logger,
			Sessions:   sessions,
			Connectors: s.connectors,
			Issuer:     s.issuer,
			Signer:     c.Signer,
			IssuerURL:  s.issuerURL,
		},
	} {
		h.Mount(routes)
	}

	routes.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !c.HealthChecker.IsHealthy() {
			s.renderError(r, w, http.StatusInternalServerError, "Health check failed.")
			return
		}
		fmt.Fprintf(w, "Health check passed")
	}))

	routes.HandlePrefix("/static", static)
	routes.HandlePrefix("/theme", theme)
	routes.HandleFunc("/robots.txt", robots)

	s.mux = r

	c.Signer.Start(ctx)
	s.startGarbageCollection(ctx, value(c.GCFrequency, 5*time.Minute), now)

	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// newKeyCacher returns a storage which caches keys so long as the next
func newKeyCacher(s storage.Storage, now func() time.Time) storage.Storage {
	if now == nil {
		now = time.Now
	}
	return &keyCacher{Storage: s, now: now}
}

type keyCacher struct {
	storage.Storage

	now  func() time.Time
	keys atomic.Value // Always holds nil or type *storage.Keys.
}

func (k *keyCacher) GetKeys(ctx context.Context) (storage.Keys, error) {
	keys, ok := k.keys.Load().(*storage.Keys)
	if ok && keys != nil && k.now().Before(keys.NextRotation) {
		return *keys, nil
	}

	storageKeys, err := k.Storage.GetKeys(ctx)
	if err != nil {
		return storageKeys, err
	}

	if k.now().Before(storageKeys.NextRotation) {
		k.keys.Store(&storageKeys)
	}
	return storageKeys, nil
}

func (s *Server) startGarbageCollection(ctx context.Context, frequency time.Duration, now func() time.Time) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(frequency):
				if r, err := s.storage.GarbageCollect(ctx, now()); err != nil {
					s.logger.ErrorContext(ctx, "garbage collection failed", "err", err)
				} else if !r.IsEmpty() {
					s.logger.InfoContext(ctx, "garbage collection run, delete auth",
						"requests", r.AuthRequests, "auth_codes", r.AuthCodes,
						"device_requests", r.DeviceRequests, "device_tokens", r.DeviceTokens,
						"auth_sessions", r.AuthSessions)
				}
			}
		}
	}()
}

// renderError renders a user-facing error page for the non-flow endpoints the
// server still serves directly (e.g. /healthz).
func (s *Server) renderError(r *http.Request, w http.ResponseWriter, status int, description string) {
	if err := s.templates.Err(r, w, status, description); err != nil {
		s.logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

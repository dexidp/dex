package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path"
	"sort"
	"sync/atomic"
	"time"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/bcrypt"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/connector/atlassiancrowd"
	"github.com/dexidp/dex/connector/authproxy"
	"github.com/dexidp/dex/connector/bitbucketcloud"
	"github.com/dexidp/dex/connector/gitea"
	"github.com/dexidp/dex/connector/github"
	"github.com/dexidp/dex/connector/gitlab"
	"github.com/dexidp/dex/connector/google"
	"github.com/dexidp/dex/connector/keystone"
	"github.com/dexidp/dex/connector/ldap"
	"github.com/dexidp/dex/connector/linkedin"
	"github.com/dexidp/dex/connector/microsoft"
	"github.com/dexidp/dex/connector/mock"
	"github.com/dexidp/dex/connector/oauth"
	"github.com/dexidp/dex/connector/oidc"
	"github.com/dexidp/dex/connector/openshift"
	"github.com/dexidp/dex/connector/saml"
	"github.com/dexidp/dex/pkg/featureflags"
	"github.com/dexidp/dex/server/authflow"
	"github.com/dexidp/dex/server/authflow/mfa"
	"github.com/dexidp/dex/server/authflow/session"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/device"
	"github.com/dexidp/dex/server/discovery"
	"github.com/dexidp/dex/server/grants"
	"github.com/dexidp/dex/server/home"
	"github.com/dexidp/dex/server/introspection"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/passwords"
	"github.com/dexidp/dex/server/reqctx"
	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/server/userinfo"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/web"
)

// LocalConnector is the local passwordDB connector which is an internal
// connector maintained by the server.
const LocalConnector = "local"

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
	issuerURL url.URL

	// In-memory cache of opened connectors.
	connectors *connectors.Cache

	storage storage.Storage

	mux http.Handler

	templates *templates.Templates

	// If enabled, don't prompt user for approval after logging in through connector.
	skipApproval bool

	// If enabled, show the connector selection screen even if there's only one
	alwaysShowLogin bool

	// Used for password grant
	passwordConnector string

	supportedResponseTypes map[string]bool

	supportedGrantTypes []string

	pkce authflow.PKCEConfig

	now func() time.Time

	idTokensValidFor       time.Duration
	authRequestsValidFor   time.Duration
	deviceRequestsValidFor time.Duration

	refreshTokenPolicy *tokens.RefreshStrategy

	logger *slog.Logger

	signer signer.Signer

	// issuer turns an Authorization into a TokenSet.
	issuer *tokens.Issuer

	sessionConfig *session.Config

	mfaProviders    map[string]mfa.Provider
	defaultMFAChain []string
}

// routeMux adapts the server's route-registration closures to router.Mux so
// domain handlers can mount their own routes.
type routeMux struct {
	handle       func(string, http.Handler)
	handleFunc   func(string, http.HandlerFunc)
	handleCORS   func(string, http.HandlerFunc)
	handlePrefix func(string, http.Handler)
}

func (m routeMux) Handle(p string, h http.Handler)         { m.handle(p, h) }
func (m routeMux) HandleFunc(p string, h http.HandlerFunc) { m.handleFunc(p, h) }
func (m routeMux) HandleCORS(p string, h http.HandlerFunc) { m.handleCORS(p, h) }
func (m routeMux) HandlePrefix(p string, h http.Handler)   { m.handlePrefix(p, h) }

// newDiscoveryHandler builds a discovery handler from the server's settings. It
// is shared by the mounted handler and ConstructDiscovery.
func (s *Server) newDiscoveryHandler() *discovery.Handler {
	return &discovery.Handler{
		Issuer:          s.issuerURL.String(),
		AbsURL:          s.absURL,
		RenderError:     s.renderError,
		Signer:          s.signer,
		Logger:          s.logger,
		ResponseTypes:   s.supportedResponseTypes,
		GrantTypes:      s.supportedGrantTypes,
		PKCEMethods:     s.pkce.CodeChallengeMethodsSupported,
		SessionsEnabled: s.sessionConfig != nil,
	}
}

// Connectors is the server's connector cache. The gRPC API needs it to
// invalidate the cache on connector CRUD.
func (s *Server) Connectors() *connectors.Cache { return s.connectors }

// ConstructDiscovery builds the OIDC discovery document. The gRPC API's
// GetDiscovery uses it to return the same document served over HTTP.
func (s *Server) ConstructDiscovery(ctx context.Context) discovery.Document {
	return s.newDiscoveryHandler().Construct(ctx)
}

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

	s := &Server{
		issuerURL:              *issuerURL,
		storage:                newKeyCacher(c.Storage, now),
		supportedResponseTypes: supportedRes,
		supportedGrantTypes:    supportedGrants,
		pkce:                   c.PKCE,
		idTokensValidFor:       value(c.IDTokensValidFor, 24*time.Hour),
		authRequestsValidFor:   value(c.AuthRequestsValidFor, 24*time.Hour),
		deviceRequestsValidFor: value(c.DeviceRequestsValidFor, 5*time.Minute),
		refreshTokenPolicy:     c.RefreshTokenPolicy,
		skipApproval:           c.SkipApprovalScreen,
		alwaysShowLogin:        c.AlwaysShowLoginScreen,
		now:                    now,
		templates:              tmpls,
		passwordConnector:      c.PasswordConnector,
		logger:                 c.Logger,
		signer:                 c.Signer,
		sessionConfig:          c.SessionConfig,
		mfaProviders:           c.MFAProviders,
		defaultMFAChain:        c.DefaultMFAChain,
	}
	s.issuer = tokens.NewIssuer(s.storage, s.signer, s.issuerURL, s.idTokensValidFor, s.now, s.logger)
	s.connectors = connectors.NewCache(s.storage, s.resolveConnector)
	discoveryHandler := s.newDiscoveryHandler()
	userInfoHandler := &userinfo.Handler{
		Issuer: s.issuerURL.String(),
		Signer: s.signer,
		Logger: s.logger,
	}
	introspectHandler := &introspection.Handler{
		Issuer:        s.issuerURL.String(),
		Signer:        s.signer,
		Storage:       s.storage,
		Logger:        s.logger,
		RefreshPolicy: s.refreshTokenPolicy,
	}
	deviceHandler := &device.Handler{
		IssuerURL:        s.issuerURL,
		Storage:          s.storage,
		Templates:        s.templates,
		Now:              s.now,
		RequestsValidFor: s.deviceRequestsValidFor,
		Logger:           s.logger,
		Issuer:           s.issuer,
		Connectors:       s.connectors,
	}
	homeHandler := &home.Handler{
		IssuerURL:       s.issuerURL,
		Storage:         s.storage,
		Templates:       s.templates,
		Logger:          s.logger,
		SessionsEnabled: s.sessionConfig != nil,
	}
	if s.sessionConfig != nil {
		homeHandler.CookieName = s.sessionConfig.CookieName
		homeHandler.CookieEncryptionKey = s.sessionConfig.CookieEncryptionKey
	}

	tokenEndpoint := grants.NewEndpoint(grants.Config{
		Issuer:              s.issuer,
		Storage:             s.storage,
		Connectors:          s.connectors,
		Now:                 s.now,
		Logger:              s.logger,
		PasswordConnector:   s.passwordConnector,
		RefreshPolicy:       s.refreshTokenPolicy,
		SessionsEnabled:     s.sessionConfig != nil,
		SupportedGrantTypes: s.supportedGrantTypes,
	})

	authFlow := authflow.NewHandler(authflow.Config{
		IssuerURL:              s.issuerURL,
		Connectors:             s.connectors,
		Storage:                s.storage,
		Templates:              s.templates,
		Signer:                 s.signer,
		Issuer:                 s.issuer,
		Now:                    s.now,
		Logger:                 s.logger,
		SkipApproval:           s.skipApproval,
		AlwaysShowLogin:        s.alwaysShowLogin,
		SupportedResponseTypes: s.supportedResponseTypes,
		PKCE:                   s.pkce,
		AuthRequestsValidFor:   s.authRequestsValidFor,
		SessionConfig:          s.sessionConfig,
		MFAProviders:           s.mfaProviders,
		DefaultMFAChain:        s.defaultMFAChain,
	})

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
		if _, err := s.OpenConnector(conn); err != nil {
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

	handlerWithHeaders := func(handlerName string, handler http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			for k, v := range c.Headers {
				w.Header()[k] = v
			}

			// Context values are used for logging purposes with the log/slog logger.
			rCtx := r.Context()
			rCtx = reqctx.WithRequestID(rCtx)

			if c.RealIPHeader != "" {
				realIP, err := parseRealIP(r)
				if err == nil {
					rCtx = reqctx.WithRemoteIP(rCtx, realIP)
				}
			}

			r = r.WithContext(rCtx)
			instrumentHandler(handlerName, handler)(w, r)
		}
	}

	r := mux.NewRouter().SkipClean(true).UseEncodedPath()
	handle := func(p string, h http.Handler) {
		r.Handle(path.Join(issuerURL.Path, p), handlerWithHeaders(p, h))
	}
	handleFunc := func(p string, h http.HandlerFunc) {
		handle(p, h)
	}
	handlePrefix := func(p string, h http.Handler) {
		prefix := path.Join(issuerURL.Path, p)
		r.PathPrefix(prefix).Handler(http.StripPrefix(prefix, h))
	}
	handleWithCORS := func(p string, h http.HandlerFunc) {
		var handler http.Handler = h
		if len(c.AllowedOrigins) > 0 {
			cors := handlers.CORS(
				handlers.AllowedOrigins(c.AllowedOrigins),
				handlers.AllowedHeaders(c.AllowedHeaders),
			)
			handler = cors(handler)
		}
		r.Handle(path.Join(issuerURL.Path, p), handlerWithHeaders(p, handler))
	}
	r.NotFoundHandler = http.NotFoundHandler()

	// Self-contained domains mount their own routes through the router.Mux
	// abstraction, so this list is the only place they are wired in.
	mux := routeMux{handle: handle, handleFunc: handleFunc, handleCORS: handleWithCORS, handlePrefix: handlePrefix}
	for _, h := range []router.Handler{tokenEndpoint, discoveryHandler, userInfoHandler, introspectHandler, deviceHandler, homeHandler, authFlow} {
		h.Mount(mux)
	}

	// TODO(ericchiang): rate limit certain paths based on IP.
	handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !c.HealthChecker.IsHealthy() {
			s.renderError(r, w, http.StatusInternalServerError, "Health check failed.")
			return
		}
		fmt.Fprintf(w, "Health check passed")
	}))

	handlePrefix("/static", static)
	handlePrefix("/theme", theme)
	handleFunc("/robots.txt", robots)

	s.mux = r

	s.signer.Start(ctx)
	s.startGarbageCollection(ctx, value(c.GCFrequency, 5*time.Minute), now)

	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) absPath(pathItems ...string) string {
	paths := make([]string, len(pathItems)+1)
	paths[0] = s.issuerURL.Path
	copy(paths[1:], pathItems)
	return path.Join(paths...)
}

func (s *Server) absURL(pathItems ...string) string {
	u := s.issuerURL
	u.Path = s.absPath(pathItems...)
	return u.String()
}

func newPasswordDB(s storage.Storage) interface {
	connector.Connector
	connector.PasswordConnector
} {
	return passwordDB{s}
}

type passwordDB struct {
	s storage.Storage
}

func resolvePasswordName(p storage.Password) string {
	if p.Name != "" {
		return p.Name
	}
	return p.Username
}

func resolvePasswordEmailVerified(p storage.Password) bool {
	if p.EmailVerified != nil {
		return *p.EmailVerified
	}
	return true
}

func (db passwordDB) Login(ctx context.Context, s connector.Scopes, email, password string) (connector.Identity, bool, error) {
	p, err := db.s.GetPassword(ctx, email)
	if err != nil {
		if err != storage.ErrNotFound {
			return connector.Identity{}, false, fmt.Errorf("get password: %v", err)
		}
		return connector.Identity{}, false, nil
	}
	// This check prevents dex users from logging in using static passwords
	// configured with hash costs that are too high or low.
	if err := passwords.CheckCost(p.Hash); err != nil {
		return connector.Identity{}, false, err
	}
	if err := bcrypt.CompareHashAndPassword(p.Hash, []byte(password)); err != nil {
		return connector.Identity{}, false, nil
	}
	return connector.Identity{
		UserID:            p.UserID,
		Username:          resolvePasswordName(p),
		PreferredUsername: p.PreferredUsername,
		Email:             p.Email,
		EmailVerified:     resolvePasswordEmailVerified(p),
		Groups:            p.Groups,
	}, true, nil
}

func (db passwordDB) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	// If the user has been deleted, the refresh token will be rejected.
	p, err := db.s.GetPassword(ctx, identity.Email)
	if err != nil {
		if err == storage.ErrNotFound {
			return connector.Identity{}, errors.New("user not found")
		}
		return connector.Identity{}, fmt.Errorf("get password: %v", err)
	}

	// User removed but a new user with the same email exists.
	if p.UserID != identity.UserID {
		return connector.Identity{}, errors.New("user not found")
	}

	// If a user has updated their username, that will be reflected in the
	// refreshed token.
	//
	// No other fields are expected to be refreshable as email is effectively used
	// as an ID.
	identity.Username = resolvePasswordName(p)
	identity.PreferredUsername = p.PreferredUsername
	identity.EmailVerified = resolvePasswordEmailVerified(p)
	identity.Groups = p.Groups

	return identity, nil
}

func (db passwordDB) Prompt() string {
	return "Email Address"
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

// ConnectorConfig is a configuration that can open a connector.
type ConnectorConfig interface {
	Open(id string, logger *slog.Logger) (connector.Connector, error)
}

// ConnectorsConfig variable provides an easy way to return a config struct
// depending on the connector type.
var ConnectorsConfig = map[string]func() ConnectorConfig{
	"keystone":        func() ConnectorConfig { return new(keystone.Config) },
	"mockCallback":    func() ConnectorConfig { return new(mock.CallbackConfig) },
	"mockPassword":    func() ConnectorConfig { return new(mock.PasswordConfig) },
	"ldap":            func() ConnectorConfig { return new(ldap.Config) },
	"gitea":           func() ConnectorConfig { return new(gitea.Config) },
	"github":          func() ConnectorConfig { return new(github.Config) },
	"gitlab":          func() ConnectorConfig { return new(gitlab.Config) },
	"google":          func() ConnectorConfig { return new(google.Config) },
	"oidc":            func() ConnectorConfig { return new(oidc.Config) },
	"oauth":           func() ConnectorConfig { return new(oauth.Config) },
	"saml":            func() ConnectorConfig { return new(saml.Config) },
	"authproxy":       func() ConnectorConfig { return new(authproxy.Config) },
	"linkedin":        func() ConnectorConfig { return new(linkedin.Config) },
	"microsoft":       func() ConnectorConfig { return new(microsoft.Config) },
	"bitbucket-cloud": func() ConnectorConfig { return new(bitbucketcloud.Config) },
	"openshift":       func() ConnectorConfig { return new(openshift.Config) },
	"atlassian-crowd": func() ConnectorConfig { return new(atlassiancrowd.Config) },
	// Keep around for backwards compatibility.
	"samlExperimental": func() ConnectorConfig { return new(saml.Config) },
}

// openConnector will parse the connector config and open the connector.
func openConnector(logger *slog.Logger, conn storage.Connector) (connector.Connector, error) {
	var c connector.Connector

	f, ok := ConnectorsConfig[conn.Type]
	if !ok {
		return c, fmt.Errorf("unknown connector type %q", conn.Type)
	}

	connConfig := f()
	if len(conn.Config) != 0 {
		data := []byte(string(conn.Config))
		if err := json.Unmarshal(data, connConfig); err != nil {
			return c, fmt.Errorf("parse connector config: %v", err)
		}
	}

	c, err := connConfig.Open(conn.ID, logger)
	if err != nil {
		return c, fmt.Errorf("failed to create connector %s: %v", conn.ID, err)
	}

	return c, nil
}

// resolveConnector builds the underlying connector implementation for a stored
// connector: the built-in local password DB, or a configured connector.
func (s *Server) resolveConnector(conn storage.Connector) (connector.Connector, error) {
	if conn.Type == LocalConnector {
		return newPasswordDB(s.storage), nil
	}
	return openConnector(s.logger, conn)
}

// OpenConnector opens the given connector and records it in the in-memory cache.
func (s *Server) OpenConnector(conn storage.Connector) (connectors.Connector, error) {
	return s.connectors.Open(conn)
}

// CloseConnector removes the connector from the in-memory cache.
func (s *Server) CloseConnector(id string) {
	s.connectors.Close(id)
}

// renderError renders a user-facing error page for the non-flow endpoints the
// server still serves directly (e.g. /healthz).
func (s *Server) renderError(r *http.Request, w http.ResponseWriter, status int, description string) {
	if err := s.templates.Err(r, w, status, description); err != nil {
		s.logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

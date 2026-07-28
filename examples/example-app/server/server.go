package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/dexidp/dex/examples/example-app/session"
)

// Options configures the Server.
type Options struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	IssuerURL    string
	PKCE         bool
	RootCAs      string
	Debug        bool

	// SessionCheckInterval is how stale the app lets its idea of the provider's
	// session get before confirming it with a prompt=none request. Zero turns
	// the check off, which means the app will keep showing a user who has since
	// signed out of the provider.
	SessionCheckInterval time.Duration

	// gRPC API. Empty GRPCAddr leaves the admin page out of the app entirely.
	GRPCAddr       string
	GRPCCA         string
	GRPCClientCert string
	GRPCClientKey  string
}

// Server is the HTTP server for the example OIDC client application.
type Server struct {
	clientID     string
	clientSecret string
	redirectURI  string
	pkce         bool

	sessionCheckInterval time.Duration

	provider        *oidc.Provider
	verifier        *oidc.IDTokenVerifier
	scopesSupported []string
	grantsSupported []string
	offlineAsScope  bool

	// Discovered endpoint URLs.
	deviceAuthURL      string
	userInfoURL        string
	jwksURL            string
	introspectURL      string
	endSessionEndpoint string

	client   *http.Client
	renderer Renderer
	sessions *session.Store
	admin    *adminClient
}

// New creates a Server by performing OIDC discovery and initializing dependencies.
func New(opts Options) (*Server, error) {
	client, err := newHTTPClient(opts.RootCAs, opts.Debug)
	if err != nil {
		return nil, err
	}

	ctx := oidc.ClientContext(context.Background(), client)
	provider, err := oidc.NewProvider(ctx, opts.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to query provider %q: %v", opts.IssuerURL, err)
	}

	// Extract discovery metadata: scopes and endpoint URLs.
	var discovery struct {
		ScopesSupported             []string `json:"scopes_supported"`
		GrantTypesSupported         []string `json:"grant_types_supported"`
		UserInfoEndpoint            string   `json:"userinfo_endpoint"`
		DeviceAuthorizationEndpoint string   `json:"device_authorization_endpoint"`
		JWKSURI                     string   `json:"jwks_uri"`
		IntrospectionEndpoint       string   `json:"introspection_endpoint"`
		EndSessionEndpoint          string   `json:"end_session_endpoint"`
	}
	if err := provider.Claims(&discovery); err != nil {
		return nil, fmt.Errorf("failed to parse provider discovery claims: %v", err)
	}

	// Determine offline access strategy.
	offlineAsScope := true
	if len(discovery.ScopesSupported) > 0 {
		offlineAsScope = slices.Contains(discovery.ScopesSupported, oidc.ScopeOfflineAccess)
	}

	introspectURL := discovery.IntrospectionEndpoint
	if introspectURL == "" {
		// Dex serves it but has not always advertised it.
		introspectURL = strings.TrimSuffix(opts.IssuerURL, "/") + "/token/introspect"
	}

	s := &Server{
		clientID:     opts.ClientID,
		clientSecret: opts.ClientSecret,
		redirectURI:  opts.RedirectURI,
		pkce:         opts.PKCE,

		sessionCheckInterval: opts.SessionCheckInterval,

		provider:        provider,
		verifier:        provider.Verifier(&oidc.Config{ClientID: opts.ClientID}),
		scopesSupported: discovery.ScopesSupported,
		grantsSupported: discovery.GrantTypesSupported,
		offlineAsScope:  offlineAsScope,

		deviceAuthURL:      discovery.DeviceAuthorizationEndpoint,
		userInfoURL:        discovery.UserInfoEndpoint,
		jwksURL:            discovery.JWKSURI,
		introspectURL:      introspectURL,
		endSessionEndpoint: discovery.EndSessionEndpoint,

		client:   client,
		renderer: newTemplateRenderer(),
		sessions: session.NewStore(),
	}

	if opts.GRPCAddr != "" {
		admin, err := newAdminClient(opts)
		if err != nil {
			return nil, err
		}
		s.admin = admin
	}

	return s, nil
}

// supportsGrant reports whether the provider advertises a grant. A provider
// that does not is not going to start honouring it because this app offers a
// button for it, so the button is not drawn.
func (s *Server) supportsGrant(grant string) bool {
	if len(s.grantsSupported) == 0 {
		// Nothing advertised: offer everything rather than nothing, since the
		// metadata field is optional.
		return true
	}
	return slices.Contains(s.grantsSupported, grant)
}

// oauth2Config returns an oauth2.Config for the given scopes.
func (s *Server) oauth2Config(scopes []string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.clientID,
		ClientSecret: s.clientSecret,
		Endpoint:     s.provider.Endpoint(),
		Scopes:       scopes,
		RedirectURL:  s.redirectURI,
	}
}

// sessionKey marks the session already resolved for a request.
type sessionKey struct{}

// session returns this browser's session, creating one if needed.
//
// The lookup is remembered on the request: handlers ask for the session more
// than once, and on the very first request — the one that has no cookie yet —
// asking twice used to mint two sessions and set two cookies. Whatever the
// second one recorded then belonged to a session the browser never came back
// with, which looked like tokens appearing from nowhere and sign-ins that did
// not survive the redirect they arrived on.
func (s *Server) session(w http.ResponseWriter, r *http.Request) *session.Session {
	if sess, ok := r.Context().Value(sessionKey{}).(*session.Session); ok {
		return sess
	}

	sess := s.sessions.FromRequest(w, r, strings.HasPrefix(s.redirectURI, "https://"))
	*r = *r.WithContext(context.WithValue(r.Context(), sessionKey{}, sess))
	return sess
}

// routes builds the HTTP handler with all application routes.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.StripPrefix("/static/", staticHandler))

	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("POST /login", s.handleLogin)
	mux.HandleFunc("GET /logout", s.handleLogout)

	// Parse redirect URI to register callback on the correct path.
	callbackPath := "/callback"
	if u, err := url.Parse(s.redirectURI); err == nil && u.Path != "" {
		callbackPath = u.Path
	}
	mux.HandleFunc("GET "+callbackPath, s.handleAuthCallback)

	mux.HandleFunc("POST /device/login", s.handleDeviceStart)
	mux.HandleFunc("GET /device", s.handleDeviceStatus)
	mux.HandleFunc("POST /device/poll", s.handleDevicePoll)
	mux.HandleFunc("GET /device/result", s.handleDeviceComplete)

	// Grants that need no browser redirect: a page for the parameters, and the
	// endpoint the page posts to.
	mux.HandleFunc("GET /grant/{grant}", s.handleGrantForm)
	mux.HandleFunc("POST /grant/refresh", s.handleRefreshGrant)
	mux.HandleFunc("POST /grant/client-credentials", s.handleClientCredentialsGrant)
	mux.HandleFunc("POST /grant/password", s.handlePasswordGrant)
	mux.HandleFunc("POST /grant/token-exchange", s.handleTokenExchangeGrant)

	mux.HandleFunc("GET /tokens", s.handleTokens)

	// Tools for looking at a token you already hold.
	mux.HandleFunc("GET /tools", s.handleTools)
	mux.HandleFunc("GET /tools/{tool}", s.handleToolForm)
	mux.HandleFunc("POST /tools/introspect", s.handleIntrospect)
	mux.HandleFunc("POST /tools/verify", s.handleVerify)
	mux.HandleFunc("POST /userinfo", s.handleUserInfo)

	// The page is always served; without a gRPC address it says what to pass to
	// get the rest of it.
	mux.HandleFunc("GET /admin", s.handleAdmin)
	if s.admin != nil {
		mux.HandleFunc("POST /admin/client/create", s.handleAdminCreateClient)
		mux.HandleFunc("POST /admin/client/update", s.handleAdminUpdateClient)
		mux.HandleFunc("POST /admin/client/delete", s.handleAdminDeleteClient)
		mux.HandleFunc("POST /admin/password/create", s.handleAdminCreatePassword)
		mux.HandleFunc("POST /admin/password/update", s.handleAdminUpdatePassword)
		mux.HandleFunc("POST /admin/password/verify", s.handleAdminVerifyPassword)
		mux.HandleFunc("POST /admin/password/delete", s.handleAdminDeletePassword)
		mux.HandleFunc("POST /admin/connector/create", s.handleAdminCreateConnector)
		mux.HandleFunc("POST /admin/connector/delete", s.handleAdminDeleteConnector)
		mux.HandleFunc("POST /admin/identity/delete", s.handleAdminDeleteIdentity)
		mux.HandleFunc("POST /admin/session/delete", s.handleAdminDeleteSession)
		mux.HandleFunc("POST /admin/session/terminate", s.handleAdminTerminateSessions)
		mux.HandleFunc("POST /admin/session/terminate-connector", s.handleAdminTerminateByConnector)
		mux.HandleFunc("GET /admin/client/{id}", s.handleAdminClientDetail)
		mux.HandleFunc("GET /admin/connector/{id}", s.handleAdminConnectorDetail)
		mux.HandleFunc("GET /admin/user/{connector}/{user}", s.handleAdminUserDetail)
		mux.HandleFunc("POST /admin/connector/update", s.handleAdminUpdateConnector)
		mux.HandleFunc("POST /admin/consent/revoke", s.handleAdminRevokeConsent)
		mux.HandleFunc("POST /admin/webauthn/delete", s.handleAdminDeleteWebAuthn)
		mux.HandleFunc("POST /admin/mfa/reset", s.handleAdminResetMFA)
		mux.HandleFunc("POST /admin/mfa/secret/delete", s.handleAdminDeleteMFASecret)
		mux.HandleFunc("POST /admin/refresh/revoke", s.handleAdminRevokeRefresh)
	}

	return mux
}

// Run starts the HTTP(S) server with graceful shutdown on SIGINT/SIGTERM.
func (s *Server) Run(listenAddr, tlsCert, tlsKey string) error {
	u, err := url.Parse(listenAddr)
	if err != nil {
		return fmt.Errorf("parse listen address: %v", err)
	}

	srv := &http.Server{
		Addr:    u.Host,
		Handler: s.routes(),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", listenAddr)
		switch u.Scheme {
		case "http":
			errCh <- srv.ListenAndServe()
		case "https":
			errCh <- srv.ListenAndServeTLS(tlsCert, tlsKey)
		default:
			errCh <- fmt.Errorf("listen address %q is not using http or https", listenAddr)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Println("shutting down...")
		if s.admin != nil {
			s.admin.close()
		}
		return srv.Shutdown(context.Background())
	}
}

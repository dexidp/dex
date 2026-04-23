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
	"syscall"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/dexidp/dex/examples/example-app/session"
)

const (
	// exampleAppState is a static CSRF state parameter.
	// In production, this must be a cryptographically random per-request value.
	exampleAppState = "I wish to wash my irish wristwatch"

	// silentAuthState is the state value used for prompt=none session checks.
	silentAuthState = "silent-auth-check"
)

// Options configures the Server.
type Options struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	IssuerURL    string
	PKCE         bool
	SessionAware bool
	RootCAs      string
	Debug        bool
}

// Server is the HTTP server for the example OIDC client application.
type Server struct {
	clientID     string
	clientSecret string
	redirectURI  string
	pkce         bool
	sessionAware bool

	provider        *oidc.Provider
	verifier        *oidc.IDTokenVerifier
	scopesSupported []string
	offlineAsScope  bool
	codeVerifier    string
	codeChallenge   string

	// Discovered endpoint URLs
	deviceAuthURL      string
	userInfoURL        string
	jwksURL            string
	endSessionEndpoint string

	client   *http.Client
	renderer Renderer
	devices  session.DeviceStore
	auth     session.AuthStore
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
		UserInfoEndpoint            string   `json:"userinfo_endpoint"`
		DeviceAuthorizationEndpoint string   `json:"device_authorization_endpoint"`
		JWKSURI                     string   `json:"jwks_uri"`
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

	s := &Server{
		clientID:     opts.ClientID,
		clientSecret: opts.ClientSecret,
		redirectURI:  opts.RedirectURI,
		pkce:         opts.PKCE,
		sessionAware: opts.SessionAware,

		provider:        provider,
		verifier:        provider.Verifier(&oidc.Config{ClientID: opts.ClientID}),
		scopesSupported: discovery.ScopesSupported,
		offlineAsScope:  offlineAsScope,

		deviceAuthURL:      discovery.DeviceAuthorizationEndpoint,
		userInfoURL:        discovery.UserInfoEndpoint,
		jwksURL:            discovery.JWKSURI,
		endSessionEndpoint: discovery.EndSessionEndpoint,

		client:   client,
		renderer: newTemplateRenderer(),
		devices:  session.NewMemoryDeviceStore(),
		auth:     session.NewMemoryAuthStore(),
	}

	if s.pkce {
		s.codeVerifier = oauth2.GenerateVerifier()
		s.codeChallenge = oauth2.S256ChallengeFromVerifier(s.codeVerifier)
	}

	return s, nil
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

// routes builds the HTTP handler with all application routes.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.StripPrefix("/static/", staticHandler))

	mux.HandleFunc("GET /{$}", s.handleLoginPage)
	mux.HandleFunc("POST /login", s.handleLogin)

	// Parse redirect URI to register callback on the correct path.
	callbackPath := "/callback"
	if u, err := url.Parse(s.redirectURI); err == nil {
		callbackPath = u.Path
	}
	mux.HandleFunc("GET "+callbackPath, s.handleAuthCallback)
	mux.HandleFunc("POST "+callbackPath, s.handleTokenRefresh)

	mux.HandleFunc("POST /device/login", s.handleDeviceStart)
	mux.HandleFunc("GET /device", s.handleDeviceStatus)
	mux.HandleFunc("POST /device/poll", s.handleDevicePoll)
	mux.HandleFunc("GET /device/result", s.handleDeviceComplete)

	mux.HandleFunc("POST /userinfo", s.handleUserInfo)
	mux.HandleFunc("GET /app-logout", s.handleAppLogout)

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
		return srv.Shutdown(context.Background())
	}
}

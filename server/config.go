package server

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"sort"
	"time"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/dexidp/dex/server/authflow"
	"github.com/dexidp/dex/server/mfa"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/session"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/server/tokens"
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

	// DynamicClientRegistration enables the RFC 7591 client registration
	// endpoint. Nil keeps the endpoint disabled and out of discovery.
	DynamicClientRegistration *DynamicClientRegistrationConfig

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

// DynamicClientRegistrationConfig configures protected RFC 7591 client
// registration. Dex deliberately requires an initial access token when the
// endpoint is enabled; an accidentally open endpoint would allow arbitrary
// callers to create persistent clients.
type DynamicClientRegistrationConfig struct {
	InitialAccessToken string
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

// resolvedConfig is Config after defaults are filled in and values validated:
// everything the handlers are wired from, derived once so newServer only has to
// hand the pieces out.
type resolvedConfig struct {
	issuerURL oauth2.IssuerURL
	now       func() time.Time

	// responseTypes and grantTypes are what the server advertises and accepts,
	// narrowed to the configured subset.
	responseTypes map[string]bool
	grantTypes    []string

	authRequestsValidFor   time.Duration
	deviceRequestsValidFor time.Duration
	idTokensValidFor       time.Duration

	templates *templates.Templates
	static    http.Handler
	theme     http.Handler
	robots    http.HandlerFunc
}

// normalizeConfig validates c and derives everything the server is built from.
// It fills c's own defaults in place (response types, allowed headers, PKCE
// methods), because the handlers read those fields directly.
func normalizeConfig(c *Config) (resolvedConfig, error) {
	if c.Storage == nil {
		return resolvedConfig{}, errors.New("server: storage cannot be nil")
	}
	if c.DynamicClientRegistration != nil && c.DynamicClientRegistration.InitialAccessToken == "" {
		return resolvedConfig{}, errors.New("server: dynamic client registration requires an initial access token")
	}

	issuerURL, err := url.Parse(c.Issuer)
	if err != nil {
		return resolvedConfig{}, fmt.Errorf("server: can't parse issuer URL")
	}

	if len(c.SupportedResponseTypes) == 0 {
		c.SupportedResponseTypes = []string{oauth2.ResponseTypeCode}
	}
	if len(c.AllowedHeaders) == 0 {
		c.AllowedHeaders = []string{"Authorization"}
	}
	if len(c.PKCE.CodeChallengeMethodsSupported) == 0 {
		c.PKCE.CodeChallengeMethodsSupported = []string{oauth2.PKCEMethodS256, oauth2.PKCEMethodPlain}
	}
	for _, m := range c.PKCE.CodeChallengeMethodsSupported {
		if m != oauth2.PKCEMethodS256 && m != oauth2.PKCEMethodPlain {
			return resolvedConfig{}, fmt.Errorf("unsupported PKCE challenge method %q", m)
		}
	}

	responseTypes, grantTypes, err := supportedTypes(c)
	if err != nil {
		return resolvedConfig{}, err
	}

	static, theme, robots, tmpls, err := templates.LoadWebConfig(webConfig(c))
	if err != nil {
		return resolvedConfig{}, fmt.Errorf("server: failed to load web static: %v", err)
	}

	now := c.Now
	if now == nil {
		now = time.Now
	}

	return resolvedConfig{
		issuerURL:              oauth2.IssuerURL{URL: *issuerURL},
		now:                    now,
		responseTypes:          responseTypes,
		grantTypes:             grantTypes,
		authRequestsValidFor:   value(c.AuthRequestsValidFor, 24*time.Hour),
		deviceRequestsValidFor: value(c.DeviceRequestsValidFor, 5*time.Minute),
		idTokensValidFor:       value(c.IDTokensValidFor, 24*time.Hour),
		templates:              tmpls,
		static:                 static,
		theme:                  theme,
		robots:                 robots,
	}, nil
}

// supportedTypes resolves the response types the server accepts and the grant
// types it advertises. A response type enabling the implicit flow adds the
// implicit grant; AllowedGrantTypes, when set, narrows the result to it.
func supportedTypes(c *Config) (map[string]bool, []string, error) {
	allGrants := map[string]bool{
		oauth2.GrantTypeAuthorizationCode: true,
		oauth2.GrantTypeRefreshToken:      true,
		oauth2.GrantTypeDeviceCode:        true,
		oauth2.GrantTypeTokenExchange:     true,
		oauth2.GrantTypeClientCredentials: true,
	}
	responseTypes := make(map[string]bool)

	for _, respType := range c.SupportedResponseTypes {
		switch respType {
		case oauth2.ResponseTypeCode, oauth2.ResponseTypeIDToken, oauth2.ResponseTypeCodeIDToken:
			// continue
		case oauth2.ResponseTypeToken, oauth2.ResponseTypeCodeToken, oauth2.ResponseTypeIDTokenToken, oauth2.ResponseTypeCodeIDTokenToken:
			// response_type=token is an implicit flow, let's add it to the discovery info
			// https://datatracker.ietf.org/doc/html/rfc6749#section-4.2.1
			allGrants[oauth2.GrantTypeImplicit] = true
		default:
			return nil, nil, fmt.Errorf("unsupported response_type %q", respType)
		}
		responseTypes[respType] = true
	}

	if c.PasswordConnector != "" {
		allGrants[oauth2.GrantTypePassword] = true
	}

	var grantTypes []string
	if len(c.AllowedGrantTypes) > 0 {
		for _, grant := range c.AllowedGrantTypes {
			if allGrants[grant] {
				grantTypes = append(grantTypes, grant)
			}
		}
	} else {
		for grant := range allGrants {
			grantTypes = append(grantTypes, grant)
		}
	}
	sort.Strings(grantTypes)

	return responseTypes, grantTypes, nil
}

// webConfig resolves where the frontend assets are loaded from: an explicit
// directory, a caller-supplied filesystem, or the assets embedded in dex.
func webConfig(c *Config) templates.Config {
	webFS := web.FS()
	if c.Web.Dir != "" {
		webFS = os.DirFS(c.Web.Dir)
	} else if c.Web.WebFS != nil {
		webFS = c.Web.WebFS
	}

	return templates.Config{
		WebFS:     webFS,
		LogoURL:   c.Web.LogoURL,
		IssuerURL: c.Issuer,
		Issuer:    c.Web.Issuer,
		Theme:     c.Web.Theme,
		Extra:     c.Web.Extra,
	}
}

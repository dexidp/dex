package xsuaa

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/log"
)

// Config holds configuration options for OpenID Connect logins.
type Config struct {
	Issuer        string `json:"issuer"`
	ClientID      string `json:"clientID"`
	ClientSecret  string `json:"clientSecret"`
	RedirectURI   string `json:"redirectURI"`
	UsersEndpoint string `json:"usersEndpoint"`
	AppName       string `json:"appName"`
	// Causes client_secret to be passed as POST parameters instead of basic
	// auth. This is specifically "NOT RECOMMENDED" by the OAuth2 RFC, but some
	// providers require it.
	//
	// https://tools.ietf.org/html/rfc6749#section-2.3.1
	BasicAuthUnsupported *bool `json:"basicAuthUnsupported"`

	Scopes []string `json:"scopes"` // defaults to "profile" and "email"

	// Optional list of whitelisted domains when using Google
	// If this field is nonempty, only users from a listed domain will be allowed to log in
	HostedDomains []string `json:"hostedDomains"`

	// Override the value of email_verifed to true in the returned claims
	InsecureSkipEmailVerified bool `json:"insecureSkipEmailVerified"`

	// GetUserInfo uses the userinfo endpoint to get additional claims for
	// the token. This is especially useful where upstreams return "thin"
	// id tokens
	GetUserInfo bool `json:"getUserInfo"`

	// Configurable key which contains the user id claim
	UserIDKey string `json:"userIDKey"`

	// Configurable key which contains the user name claim
	UserNameKey *string `json:"userNameKey"`
}

// Domains that don't support basic auth. golang.org/x/oauth2 has an internal
// list, but it only matches specific URLs, not top level domains.
var brokenAuthHeaderDomains = []string{
	// See: https://github.com/dexidp/dex/issues/859
	"okta.com",
	"oktapreview.com",
}

// Detect auth header provider issues for known providers. This lets users
// avoid having to explicitly set "basicAuthUnsupported" in their config.
//
// Setting the config field always overrides values returned by this function.
func knownBrokenAuthHeaderProvider(issuerURL string) bool {
	if u, err := url.Parse(issuerURL); err == nil {
		for _, host := range brokenAuthHeaderDomains {
			if u.Host == host || strings.HasSuffix(u.Host, "."+host) {
				return true
			}
		}
	}
	return false
}

// golang.org/x/oauth2 doesn't do internal locking. Need to do it in this
// package ourselves and hope that other packages aren't calling it at the
// same time.
var registerMu = new(sync.Mutex)

func registerBrokenAuthHeaderProvider(url string) {
	registerMu.Lock()
	defer registerMu.Unlock()

	oauth2.RegisterBrokenAuthHeaderProvider(url)
}

// Open returns a connector which can be used to login users through an upstream
// OpenID Connect provider.
func (c *Config) Open(id string, logger log.Logger) (conn connector.Connector, err error) {
	ctx, cancel := context.WithCancel(context.Background())

	provider, err := oidc.NewProvider(ctx, c.Issuer)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to get provider: %v", err)
	}

	if c.BasicAuthUnsupported != nil {
		// Setting "basicAuthUnsupported" always overrides our detection.
		if *c.BasicAuthUnsupported {
			registerBrokenAuthHeaderProvider(provider.Endpoint().TokenURL)
		}
	} else if knownBrokenAuthHeaderProvider(c.Issuer) {
		registerBrokenAuthHeaderProvider(provider.Endpoint().TokenURL)
	}

	//This part is removed because in xsuaa returns all user groups when no scopes is defined in the token request
	//scopes := []string{oidc.ScopeOpenID}
	//if len(c.Scopes) > 0 {
	//	scopes = append(scopes, c.Scopes...)
	//}
	//else {
	//	scopes = append(scopes, "profile", "email")
	//}

	clientID := c.ClientID
	return &xsuaaConnector{
		provider:    provider,
		redirectURI: c.RedirectURI,
		oauth2Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: c.ClientSecret,
			Endpoint:     provider.Endpoint(),
			//Scopes:       []string,
			RedirectURL: c.RedirectURI,
		},
		verifier: provider.Verifier(
			&oidc.Config{ClientID: clientID},
		),
		logger:                    logger,
		cancel:                    cancel,
		hostedDomains:             c.HostedDomains,
		insecureSkipEmailVerified: c.InsecureSkipEmailVerified,
		getUserInfo:               c.GetUserInfo,
		userIDKey:                 c.UserIDKey,
		userNameKey:               *c.UserNameKey,
		userEndpoint:              c.UsersEndpoint,
		appName:                   c.AppName,
	}, nil
}

var (
	_ connector.CallbackConnector = (*xsuaaConnector)(nil)
	_ connector.RefreshConnector  = (*xsuaaConnector)(nil)
)

type scopes struct {
	Scopes []string `json:"scope"`
}

type xsuaaConnector struct {
	provider                  *oidc.Provider
	redirectURI               string
	oauth2Config              *oauth2.Config
	verifier                  *oidc.IDTokenVerifier
	cancel                    context.CancelFunc
	logger                    log.Logger
	hostedDomains             []string
	insecureSkipEmailVerified bool
	getUserInfo               bool
	userIDKey                 string
	userNameKey               string
	userEndpoint              string
	appName                   string
}

func filterUserGroups(scopes []string, prefix string) (userGroups []string) {
	//Return only scopes aka groups that contain given prefix (client display name)
	if !strings.HasSuffix(prefix, ".") {
		prefix = prefix + "."
	}

	for _, scp := range scopes {
		if strings.HasPrefix(scp, prefix) {
			name := strings.TrimPrefix(scp, prefix)
			userGroups = append(userGroups, name)
		}
	}
	return userGroups
}

func (c *xsuaaConnector) Close() error {
	c.cancel()
	return nil
}

func (c *xsuaaConnector) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
	if c.redirectURI != callbackURL {
		return "", fmt.Errorf("expected callback URL %q did not match the URL in the config %q", callbackURL, c.redirectURI)
	}

	if len(c.hostedDomains) > 0 {
		preferredDomain := c.hostedDomains[0]
		if len(c.hostedDomains) > 1 {
			preferredDomain = "*"
		}
		return c.oauth2Config.AuthCodeURL(state, oauth2.SetAuthURLParam("hd", preferredDomain)), nil
	}
	return c.oauth2Config.AuthCodeURL(state), nil
}

type oauth2Error struct {
	error            string
	errorDescription string
}

func (e *oauth2Error) Error() string {
	if e.errorDescription == "" {
		return e.error
	}
	return e.error + ": " + e.errorDescription
}

func (c *xsuaaConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return identity, &oauth2Error{errType, q.Get("error_description")}
	}
	token, err := c.oauth2Config.Exchange(r.Context(), q.Get("code"))
	if err != nil {
		return identity, fmt.Errorf("xsuaa: failed to get token: %v", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return identity, errors.New("xsuaa: no id_token in token response")
	}

	idToken, err := c.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		return identity, fmt.Errorf("xsuaa: failed to verify ID Token: %v", err)
	}

	accessToken, err := c.verifier.Verify(r.Context(), token.AccessToken)

	if err != nil {
		return identity, fmt.Errorf("xsuaa: failed to verify access_token :%v", err)
	}

	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		return identity, fmt.Errorf("xsuaa: failed to decode claims: %v", err)
	}

	c.logger.Debugf("User claims %+v", claims)

	userNameKey := "name"
	if c.userNameKey != "" {
		userNameKey = c.userNameKey
	}
	name, found := claims[userNameKey].(string)
	if !found {
		return identity, fmt.Errorf("missing \"%s\" claim", userNameKey)
	}
	email, found := claims["email"].(string)
	if !found {
		return identity, errors.New("missing \"email\" claim")
	}
	emailVerified, found := claims["email_verified"].(bool)
	if !found {
		if c.insecureSkipEmailVerified {
			emailVerified = true
		} else {
			return identity, errors.New("missing \"email_verified\" claim")
		}
	}

	//TODO replaces scopes struct with whole claims struct and remove unnecessary type assertions then
	var scp scopes

	if err := accessToken.Claims(&scp); err != nil {
		return identity, fmt.Errorf("xsuaa: failed to decode scopes: %v", err)
	}

	hostedDomain, _ := claims["hd"].(string)

	if len(c.hostedDomains) > 0 {
		found := false
		for _, domain := range c.hostedDomains {
			if hostedDomain == domain {
				found = true
				break
			}
		}

		if !found {
			return identity, fmt.Errorf("oidc: unexpected hd claim %v", hostedDomain)
		}
	}

	if c.getUserInfo {
		userInfo, err := c.provider.UserInfo(r.Context(), oauth2.StaticTokenSource(token))
		if err != nil {
			return identity, fmt.Errorf("oidc: error loading userinfo: %v", err)
		}
		if err := userInfo.Claims(&claims); err != nil {
			return identity, fmt.Errorf("oidc: failed to decode userinfo claims: %v", err)
		}
	}

	tenantID, found := claims["zid"].(string)

	if !found {
		return identity, errors.New("missing tenantID claim")
	}

	userGroups := filterUserGroups(scp.Scopes, c.appName)
	userGroups = append(userGroups, fmt.Sprintf("tenantID=%s", tenantID))

	identity = connector.Identity{
		UserID:        accessToken.Subject,
		Username:      name,
		Email:         email,
		EmailVerified: emailVerified,
		Groups:        userGroups,
	}

	if c.userIDKey != "" {
		userID, found := claims[c.userIDKey].(string)
		if !found {
			return identity, fmt.Errorf("oidc: not found %v claim", c.userIDKey)
		}
		identity.UserID = userID
	}

	return identity, nil
}

// Refresh is implemented for backwards compatibility, even though it's a no-op.
func (c *xsuaaConnector) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	return identity, nil
}

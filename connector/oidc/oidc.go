// Package oidc implements logging in through OpenID Connect providers.
package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/log"
)

// Config holds configuration options for OpenID Connect logins.
type Config struct {
	Issuer       string `json:"issuer"`
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	RedirectURI  string `json:"redirectURI"`

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

	// InsecureEnableGroups enables groups claims. This is disabled by default until https://github.com/dexidp/dex/issues/1065 is resolved
	InsecureEnableGroups bool `json:"insecureEnableGroups"`

	// GetUserInfo uses the userinfo endpoint to get additional claims for
	// the token. This is especially useful where upstreams return "thin"
	// id tokens
	GetUserInfo bool `json:"getUserInfo"`

	// Configurable key which contains the user id claim
	UserIDKey string `json:"userIDKey"`

	// Configurable key which contains the user name claim
	UserNameKey string `json:"userNameKey"`
}

// Domains that don't support basic auth. golang.org/x/oauth2 has an internal
// list, but it only matches specific URLs, not top level domains.
var brokenAuthHeaderDomains = []string{
	// See: https://github.com/dexidp/dex/issues/859
	"okta.com",
	"oktapreview.com",
}

// connectorData stores information for sessions authenticated by this connector
type connectorData struct {
	RefreshToken []byte
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

// Open returns a connector which can be used to login users through an upstream
// OpenID Connect provider.
func (c *Config) Open(id string, logger log.Logger) (conn connector.Connector, err error) {
	ctx, cancel := context.WithCancel(context.Background())

	provider, err := oidc.NewProvider(ctx, c.Issuer)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to get provider: %v", err)
	}

	endpoint := provider.Endpoint()

	if c.BasicAuthUnsupported != nil {
		// Setting "basicAuthUnsupported" always overrides our detection.
		if *c.BasicAuthUnsupported {
			endpoint.AuthStyle = oauth2.AuthStyleInParams
		}
	} else if knownBrokenAuthHeaderProvider(c.Issuer) {
		endpoint.AuthStyle = oauth2.AuthStyleInParams
	}

	scopes := []string{oidc.ScopeOpenID}
	if len(c.Scopes) > 0 {
		scopes = append(scopes, c.Scopes...)
	} else {
		scopes = append(scopes, "profile", "email")
	}

	clientID := c.ClientID
	return &oidcConnector{
		provider:    provider,
		redirectURI: c.RedirectURI,
		oauth2Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: c.ClientSecret,
			Endpoint:     endpoint,
			Scopes:       scopes,
			RedirectURL:  c.RedirectURI,
		},
		verifier: provider.Verifier(
			&oidc.Config{ClientID: clientID},
		),
		logger:                    logger,
		cancel:                    cancel,
		hostedDomains:             c.HostedDomains,
		insecureSkipEmailVerified: c.InsecureSkipEmailVerified,
		insecureEnableGroups:      c.InsecureEnableGroups,
		getUserInfo:               c.GetUserInfo,
		userIDKey:                 c.UserIDKey,
		userNameKey:               c.UserNameKey,
	}, nil
}

var (
	_ connector.CallbackConnector = (*oidcConnector)(nil)
	_ connector.RefreshConnector  = (*oidcConnector)(nil)
)

type oidcConnector struct {
	provider                  *oidc.Provider
	redirectURI               string
	oauth2Config              *oauth2.Config
	verifier                  *oidc.IDTokenVerifier
	cancel                    context.CancelFunc
	logger                    log.Logger
	hostedDomains             []string
	insecureSkipEmailVerified bool
	insecureEnableGroups      bool
	getUserInfo               bool
	userIDKey                 string
	userNameKey               string
}

func (c *oidcConnector) Close() error {
	c.cancel()
	return nil
}

func (c *oidcConnector) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
	if c.redirectURI != callbackURL {
		return "", fmt.Errorf("expected callback URL %q did not match the URL in the config %q", callbackURL, c.redirectURI)
	}

	var opts []oauth2.AuthCodeOption
	if len(c.hostedDomains) > 0 {
		preferredDomain := c.hostedDomains[0]
		if len(c.hostedDomains) > 1 {
			preferredDomain = "*"
		}
		opts = append(opts, oauth2.SetAuthURLParam("hd", preferredDomain))
	}

	if s.OfflineAccess {
		opts = append(opts, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
	}
	return c.oauth2Config.AuthCodeURL(state, opts...), nil
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

func (c *oidcConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return identity, &oauth2Error{errType, q.Get("error_description")}
	}
	token, err := c.oauth2Config.Exchange(r.Context(), q.Get("code"))
	if err != nil {
		return identity, fmt.Errorf("oidc: failed to get token: %v", err)
	}

	return c.createIdentity(r.Context(), identity, token)
}

// Refresh is used to refresh a session with the refresh token provided by the IdP
func (c *oidcConnector) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	cd := connectorData{}
	err := json.Unmarshal(identity.ConnectorData, &cd)
	if err != nil {
		return identity, fmt.Errorf("oidc: failed to unmarshal connector data: %v", err)
	}

	t := &oauth2.Token{
		RefreshToken: string(cd.RefreshToken),
		Expiry:       time.Now().Add(-time.Hour),
	}
	token, err := c.oauth2Config.TokenSource(ctx, t).Token()
	if err != nil {
		return identity, fmt.Errorf("oidc: failed to get refresh token: %v", err)
	}

	return c.createIdentity(ctx, identity, token)
}

func (c *oidcConnector) createIdentity(ctx context.Context, identity connector.Identity, token *oauth2.Token) (connector.Identity, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return identity, errors.New("oidc: no id_token in token response")
	}
	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return identity, fmt.Errorf("oidc: failed to verify ID Token: %v", err)
	}

	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		return identity, fmt.Errorf("oidc: failed to decode claims: %v", err)
	}

	// We immediately want to run getUserInfo if configured before we validate the claims
	if c.getUserInfo {
		userInfo, err := c.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
		if err != nil {
			return identity, fmt.Errorf("oidc: error loading userinfo: %v", err)
		}
		if err := userInfo.Claims(&claims); err != nil {
			return identity, fmt.Errorf("oidc: failed to decode userinfo claims: %v", err)
		}
	}

	userNameKey := "name"
	if c.userNameKey != "" {
		userNameKey = c.userNameKey
	}
	name, found := claims[userNameKey].(string)
	if !found {
		return identity, fmt.Errorf("missing \"%s\" claim", userNameKey)
	}

	hasEmailScope := false
	for _, s := range c.oauth2Config.Scopes {
		if s == "email" {
			hasEmailScope = true
			break
		}
	}

	email, found := claims["email"].(string)
	if !found && hasEmailScope {
		return identity, errors.New("missing \"email\" claim")
	}

	emailVerified, found := claims["email_verified"].(bool)
	if !found {
		if c.insecureSkipEmailVerified {
			emailVerified = true
		} else if hasEmailScope {
			return identity, errors.New("missing \"email_verified\" claim")
		}
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

	cd := connectorData{
		RefreshToken: []byte(token.RefreshToken),
	}

	connData, err := json.Marshal(&cd)
	if err != nil {
		return identity, fmt.Errorf("oidc: failed to encode connector data: %v", err)
	}

	identity = connector.Identity{
		UserID:        idToken.Subject,
		Username:      name,
		Email:         email,
		EmailVerified: emailVerified,
		ConnectorData: connData,
	}

	if c.userIDKey != "" {
		userID, found := claims[c.userIDKey].(string)
		if !found {
			return identity, fmt.Errorf("oidc: not found %v claim", c.userIDKey)
		}
		identity.UserID = userID
	}

	if c.insecureEnableGroups {
		vs, ok := claims["groups"].([]interface{})
		if ok {
			for _, v := range vs {
				if s, ok := v.(string); ok {
					identity.Groups = append(identity.Groups, s)
				} else {
					return identity, errors.New("malformed \"groups\" claim")
				}
			}
		}
	}

	return identity, nil
}

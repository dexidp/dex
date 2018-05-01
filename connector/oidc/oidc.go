// Package oidc implements logging in through OpenID Connect providers.
package oidc

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/coreos/go-oidc"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/dexidp/dex/connector"
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

	// Configurable key which contains the groups claims
	GroupsKey string `json:"groupsKey"` // defaults to "groups"
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
func (c *Config) Open(id string, logger logrus.FieldLogger) (conn connector.Connector, err error) {
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

	scopes := []string{oidc.ScopeOpenID}
	if len(c.Scopes) > 0 {
		scopes = append(scopes, c.Scopes...)
	} else {
		scopes = append(scopes, "profile", "email")
	}

	clientID := c.ClientID
	return &oidcConnector{
		redirectURI: c.RedirectURI,
		oauth2Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: c.ClientSecret,
			Endpoint:     provider.Endpoint(),
			Scopes:       scopes,
			RedirectURL:  c.RedirectURI,
		},
		verifier: provider.Verifier(
			&oidc.Config{ClientID: clientID},
		),
		logger:        logger,
		cancel:        cancel,
		hostedDomains: c.HostedDomains,
		groupsKey:     c.GroupsKey,
	}, nil
}

var (
	_ connector.CallbackConnector = (*oidcConnector)(nil)
	_ connector.RefreshConnector  = (*oidcConnector)(nil)
)

type oidcConnector struct {
	redirectURI   string
	oauth2Config  *oauth2.Config
	verifier      *oidc.IDTokenVerifier
	ctx           context.Context
	cancel        context.CancelFunc
	logger        logrus.FieldLogger
	hostedDomains []string
	groupsKey     string
}

func (c *oidcConnector) Close() error {
	c.cancel()
	return nil
}

func (c *oidcConnector) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
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

func (c *oidcConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return identity, &oauth2Error{errType, q.Get("error_description")}
	}

	token, err := c.oauth2Config.Exchange(r.Context(), q.Get("code"))
	if err != nil {
		return identity, fmt.Errorf("oidc: failed to get token: %v", err)
	}

	rawIdToken, ok := token.Extra("id_token").(string)
	if !ok {
		return identity, fmt.Errorf("oidc: no Id token present")
	}

	idToken, err := c.verifier.Verify(r.Context(), rawIdToken)
	if err != nil {
		return identity, fmt.Errorf("oidc: failed to verify ID Token: %v", err)
	}

	var claimsMaps map[string]interface{}

	if err := idToken.Claims(&claimsMaps); err != nil {
		return identity, fmt.Errorf("oidc: failed to decode claims: %v", err)
	}

	hostedDomain, _ := claimsMaps["hd"].(string)
	name, _ := claimsMaps["name"].(string)
	userName, _ := claimsMaps["username"].(string)
	email, _ := claimsMaps["email"].(string)
	emailVerified, _ := claimsMaps["email_verified"].(bool)

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

	identity = connector.Identity{
		UserID:        idToken.Subject,
		Name:          name,
		Username:      userName,
		Email:         email,
		EmailVerified: emailVerified,
	}

	if s.Groups {
		if c.groupsKey == "" {
			c.groupsKey = "groups"
		}

		groupsClaim, _ := claimsMaps[c.groupsKey].([]interface{})

		for _, group := range groupsClaim {
			if groupString, ok := group.(string); ok {
				identity.Groups = append(identity.Groups, groupString)
			}
		}
	}
	return identity, nil
}

// Refresh is implemented for backwards compatibility, even though it's a no-op.
func (c *oidcConnector) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	return identity, nil
}

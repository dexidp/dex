// Package oidc implements logging in through OpenID Connect providers.
package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

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
}

type connectorData struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	Expiry       time.Time `json:"expiry"`
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
		provider:      provider,
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
	provider      *oidc.Provider
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

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return identity, errors.New("oidc: no id_token in token response")
	}
	idToken, err := c.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		return identity, fmt.Errorf("oidc: failed to verify ID Token: %v", err)
	}

	var claims struct {
		Username      string `json:"name"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		HostedDomain  string `json:"hd"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return identity, fmt.Errorf("oidc: failed to decode claims: %v", err)
	}

	if len(c.hostedDomains) > 0 {
		found := false
		for _, domain := range c.hostedDomains {
			if claims.HostedDomain == domain {
				found = true
				break
			}
		}

		if !found {
			return identity, fmt.Errorf("oidc: unexpected hd claim %v", claims.HostedDomain)
		}
	}

	identity = connector.Identity{
		UserID:        idToken.Subject,
		Username:      claims.Username,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
	}

	if claims.Email == "" || s.Groups {

		userinfo, err := c.provider.UserInfo(r.Context(), oauth2.StaticTokenSource(token))
		if err != nil {
			fmt.Errorf("Failed to query userinfo for additional claims")
			return identity, nil
		}

		// If the email did not come back in the JWT id_token, then
		// there is a chance the userinfo contents at the profileURL
		// contain the email.  Make a query to the userinfo endpoint
		// and attempt to locate the email from there.
		if claims.Email == "" {
			identity.Email = userinfo.Email
			// TODO: Add an option to only force this for misbehaving IDP
			// identity.EmailVerified = userinfo.EmailVerified
			identity.EmailVerified = true
		}

		// If downstream request included scope for "groups", return the groups
		// TODO: Don't force this
		if s.Groups || true {
			groups, err := c.getGroups(userinfo)
			if err != nil {
				return identity, nil
			}
			identity.Groups = groups
		}
	}

	if s.OfflineAccess {
		data := connectorData{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			Expiry:       token.Expiry,
		}
		connData, err := json.Marshal(data)
		if err != nil {
			return identity, fmt.Errorf("microsoft: marshal connector data: %v", err)
		}
		identity.ConnectorData = connData
	}

	return identity, nil
}

func (c *oidcConnector) getGroups(userinfo *oidc.UserInfo) (groups []string, err error) {

	// Unmarshal the claims into an anonymous struct
	var userinfoClaims map[string]interface{}
	if err := userinfo.Claims(&userinfoClaims); err != nil {
		fmt.Printf("failed to unmarshal claims %v", err)
		return nil, err
	}

	// Ping federate returns group claims in userInfo endpoint under the "memberof" key.
	// Break early if no "memberof" key found in userinfoClaims.
	if _, ok := userinfoClaims["memberof"]; !ok {
		return nil, errors.New("userinfo claims did not contain 'memberof' groups")
	}

	// Get a list of raw, unprocessed groups
        rawGroups := []string{}
        for _, v := range userinfoClaims["memberof"].([]interface{}) {
                rawGroups = append(rawGroups, v.(string))
        }

	/* This is implementation specific

	   Process the raw group strings which look like this

	    "CN=all-users,OU=Groups,O=department.example.com",
	    "CN=kube-admin,OU=Groups,O=department.example.com",

           To return only the CN names that start with kube-*

	   This shortens the group claims so that it stays well under the 4000
	   byte limit for cookies, so that browsers will not error.
	*/
	// TODO: make the filter pattern configurable
        groups = []string{}
        re := regexp.MustCompile("^CN=(kube-.*?),.*$")
        for _, group := range rawGroups  {
                match := re.FindStringSubmatch(group)
                if match == nil { // no pattern match
                        continue
                }
                groups = append(groups, match[1])
        }
	return groups, nil
}

// Refresh is implemented for backwards compatibility, even though it's a no-op.
func (c *oidcConnector) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	return identity, nil
}

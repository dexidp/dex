// Package apple connector
package apple

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
	"github.com/dexidp/dex/connector/apple/secret"
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

	UserIDKey string `json:"userIDKey"`

	UserNameKey string `json:"userNameKey"`

	// PromptType will be used fot the prompt parameter (when offline_access, by default prompt=consent)
	PromptType string `json:"promptType"`

	ClaimMapping struct {
		// Configurable key which contains the preferred username claims
		PreferredUsernameKey string `json:"preferred_username"` // defaults to "preferred_username"

		// Configurable key which contains the email claims
		EmailKey string `json:"email"` // defaults to "email"

		// Configurable key which contains the groups claims
		GroupsKey string `json:"groups"` // defaults to "groups"
	} `json:"claimMapping"`

	KeyID string `json:"key_id"`

	TeamID string `json:"team_id"`

	PrivateKeyFile string `json:"private_key_file"`

	SecretDuration int64 `json:"secret_duration"`

	SecretExpiryMin int64 `json:"secret_expiry_min"`
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
	/*
		secretConfig := secret.Config{

			TeamID:          c.TeamID,
			KeyID:           c.KeyID,
			PrivateKeyFile:  c.PrivateKeyFile,
			ClientID:        c.ClientID,
			Issuer:          c.Issuer,
			SecretDuration:  c.SecretDuration,
			SecretExpiryMin: c.SecretExpiryMin,
		}

		secretHandler, err := secret.NewSecret(&secretConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build apple secret handler: %v", err)
		}

		clientSecret, err := secretHandler.GetSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to get a client secret: %v", err)
		}
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
			scopes = append(scopes, "email")
		}

		// PromptType should be "consent" by default, if not set
		if c.PromptType == "" {
			c.PromptType = "consent"
		}
	*/

	//clientID := c.ClientID
	return &oidcConnector{
		/*
			provider:    provider,
			redirectURI: c.RedirectURI,
			oauth2Config: &oauth2.Config{
				ClientID:     clientID,
				ClientSecret: clientSecret,
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
			promptType:                c.PromptType,
			userIDKey:                 c.UserIDKey,
			userNameKey:               c.UserNameKey,
			preferredUsernameKey:      c.ClaimMapping.PreferredUsernameKey,
			emailKey:                  c.ClaimMapping.EmailKey,
			groupsKey:                 c.ClaimMapping.GroupsKey,
			secret:                    secretHandler,
		*/
	}, nil
}

var (
	_ connector.CallbackConnectorFormPOST = (*oidcConnector)(nil)
	_ connector.RefreshConnector          = (*oidcConnector)(nil)
)

type oidcConnector struct {
	provider                  *oidc.Provider
	secret                    *secret.Secret
	redirectURI               string
	oauth2Config              *oauth2.Config
	verifier                  *oidc.IDTokenVerifier
	cancel                    context.CancelFunc
	logger                    log.Logger
	hostedDomains             []string
	insecureSkipEmailVerified bool
	insecureEnableGroups      bool
	getUserInfo               bool
	promptType                string
	userIDKey                 string
	userNameKey               string
	preferredUsernameKey      string
	emailKey                  string
	groupsKey                 string
}

func (c *oidcConnector) Close() error {
	c.cancel()
	return nil
}

func (c *oidcConnector) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
	if c.redirectURI != callbackURL {
		return "", fmt.Errorf("expected callback URL %q did not match the URL in the config %q", callbackURL, c.redirectURI)
	}
	if c.secret.IsExpired() {
		// secret has expired, grab a new one.
		clientSecret, err := c.secret.GetSecret()
		if err != nil {
			return "", fmt.Errorf("failed to generated client secret: %v", err)
		}
		c.oauth2Config.ClientSecret = clientSecret
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
		opts = append(opts, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", c.promptType))
	}
	opts = append(opts, oauth2.SetAuthURLParam("response_mode", "form_post"))
	return c.oauth2Config.AuthCodeURL(state, opts...), nil
}

func (c *oidcConnector) HandlePOST(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	code := r.PostFormValue("code")

	token, err := c.oauth2Config.Exchange(r.Context(), code)
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

	userNameKey := "email"
	if c.userNameKey != "" {
		userNameKey = c.userNameKey
	}
	name, found := claims[userNameKey].(string)
	if !found {
		return identity, fmt.Errorf("missing \"%s\" claim", userNameKey)
	}

	preferredUsername, found := claims["preferred_username"].(string)
	if !found {
		preferredUsername, _ = claims[c.preferredUsernameKey].(string)
	}

	hasEmailScope := false
	for _, s := range c.oauth2Config.Scopes {
		if s == "email" {
			hasEmailScope = true
			break
		}
	}

	var email string
	emailKey := "email"
	email, found = claims[emailKey].(string)
	if !found && c.emailKey != "" {
		emailKey = c.emailKey
		email, found = claims[emailKey].(string)
	}

	if !found && hasEmailScope {
		return identity, fmt.Errorf("missing email claim, not found \"%s\" key", emailKey)
	}

	_emailVerified, found := claims["email_verified"].(string)
	var emailVerified bool
	if found && _emailVerified == "true" {
		emailVerified = true
	}
	if !found {
		if c.insecureSkipEmailVerified {
			emailVerified = true
		} else if hasEmailScope {
			return identity, errors.New("missing \"email_verified\" claim")
		}
	}

	var groups []string
	if c.insecureEnableGroups {
		groupsKey := "groups"
		vs, found := claims[groupsKey].([]interface{})
		if !found {
			groupsKey = c.groupsKey
			vs, found = claims[groupsKey].([]interface{})
		}

		if found {
			for _, v := range vs {
				if s, ok := v.(string); ok {
					groups = append(groups, s)
				} else {
					return identity, fmt.Errorf("malformed \"%v\" claim", groupsKey)
				}
			}
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
		UserID:            idToken.Subject,
		Username:          name,
		PreferredUsername: preferredUsername,
		Email:             email,
		EmailVerified:     emailVerified,
		Groups:            groups,
		ConnectorData:     connData,
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

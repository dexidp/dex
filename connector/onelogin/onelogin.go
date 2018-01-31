// Package onelogin implements logging in through Onelogin OpenID Connect.
package onelogin

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/coreos/dex/connector"
)

// Config holds configuration options for OpenID Connect logins.
type Config struct {
	Subdomain             string   `json:"subdomain"`
	APIID                 string   `json:"apiID"`
	APISecret             string   `json:"apiSecret"`
	ClientID              string   `json:"clientID"`
	ClientSecret          string   `json:"clientSecret"`
	RedirectURI           string   `json:"redirectURI"`
	RolesPrefix           string   `json:"rolesPrefix"`
	EmailVerifiedOverride bool     `json:"emailVerifiedOverride"`
	Scopes                []string `json:"scopes"` // defaults to "profile" and "email"
}

// Open returns a connector which can be used to login users through an upstream
// OpenID Connect provider.
func (c *Config) Open(id string, logger logrus.FieldLogger) (conn connector.Connector, err error) {
	ctx, cancel := context.WithCancel(context.Background())

	provider, err := oidc.NewProvider(ctx, "https://openid-connect.onelogin.com/oidc")
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to get provider: %v", err)
	}

	scopes := []string{oidc.ScopeOpenID}
	if len(c.Scopes) > 0 {
		scopes = append(scopes, c.Scopes...)
	} else {
		scopes = append(scopes, "profile", "email")
	}

	clientID := c.ClientID
	var apiAuth string
	if c.APIID != "" {
		apiAuth = fmt.Sprintf("client_id:%s, client_secret:%s", c.APIID, c.APISecret)
	}
	return &oidcConnector{
		redirectURI: c.RedirectURI,
		oauth2Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: c.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://" + c.Subdomain + ".onelogin.com/oidc/auth",
				TokenURL: "https://" + c.Subdomain + ".onelogin.com/oidc/token",
			},
			Scopes:      scopes,
			RedirectURL: c.RedirectURI,
		},
		verifier: provider.Verifier(
			&oidc.Config{ClientID: clientID},
		),
		logger:                logger,
		cancel:                cancel,
		apiAuth:               apiAuth,
		rolesPrefix:           c.RolesPrefix,
		emailVerifiedOverride: c.EmailVerifiedOverride,
	}, nil
}

var (
	_ connector.CallbackConnector = (*oidcConnector)(nil)
	_ connector.RefreshConnector  = (*oidcConnector)(nil)
)

type oidcConnector struct {
	redirectURI           string
	oauth2Config          *oauth2.Config
	verifier              *oidc.IDTokenVerifier
	ctx                   context.Context
	cancel                context.CancelFunc
	logger                logrus.FieldLogger
	apiAuth               string
	rolesPrefix           string
	emailVerifiedOverride bool
}

func (c *oidcConnector) Close() error {
	c.cancel()
	return nil
}

func (c *oidcConnector) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
	if c.redirectURI != callbackURL {
		return "", fmt.Errorf("expected callback URL %q did not match the URL in the config %q", callbackURL, c.redirectURI)
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
		return identity, fmt.Errorf("onelogin: failed to get token: %v", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return identity, errors.New("onelogin: no id_token in token response")
	}
	idToken, err := c.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		return identity, fmt.Errorf("onelogin: failed to verify ID Token: %v", err)
	}

	var claims struct {
		Username      string `json:"name"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return identity, fmt.Errorf("onelogin: failed to decode claims: %v", err)
	}

	groups := []string{}
	if c.apiAuth != "" {
		accessToken, err := getAPIAccessToken(c.apiAuth)
		if err != nil {
			return identity, fmt.Errorf("onelogin: failed to get api access token: %v", err)
		}

		groups, err = getAPIUserRoles(c.rolesPrefix, accessToken, idToken.Subject)
		if err != nil {
			return identity, fmt.Errorf("onelogin: failed to get api user groups: %v", err)
		}
	}

	identity = connector.Identity{
		UserID:        idToken.Subject,
		Username:      claims.Username,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified || c.emailVerifiedOverride,
		Groups:        groups,
	}
	return identity, nil
}

// Refresh is implemented for backwards compatibility, even though it's a no-op.
func (c *oidcConnector) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	return identity, nil
}

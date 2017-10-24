// Package linkedin provides authentication strategies using LinkedIn
package linkedin

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"golang.org/x/oauth2"

	"github.com/coreos/dex/connector"
	"github.com/sirupsen/logrus"
)

const (
	apiURL   = "https://api.linkedin.com/v1"
	authURL  = "https://www.linkedin.com/oauth/v2/authorization"
	tokenURL = "https://www.linkedin.com/oauth/v2/accessToken"
)

// Config holds configuration options for LinkedIn logins.
type Config struct {
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	RedirectURI  string `json:"redirectURI"`
}

// Open returns a strategy for logging in through LinkedIn
func (c *Config) Open(id string, logger logrus.FieldLogger) (connector.Connector, error) {
	return &linkedInConnector{
		oauth2Config: &oauth2.Config{
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  authURL,
				TokenURL: tokenURL,
			},
			Scopes:      []string{"r_basicprofile", "r_emailaddress"},
			RedirectURL: c.RedirectURI,
		},
		logger: logger,
	}, nil
}

type connectorData struct {
	AccessToken string `json:"accessToken"`
}

type linkedInConnector struct {
	oauth2Config *oauth2.Config
	logger       logrus.FieldLogger
}

// LinkedIn doesn't provide refresh tokens, so refresh tokens issued by Dex
// will expire in 60 days (default LinkedIn token lifetime).
var (
	_ connector.CallbackConnector = (*linkedInConnector)(nil)
	_ connector.RefreshConnector  = (*linkedInConnector)(nil)
)

// LoginURL returns an access token request URL
func (c *linkedInConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, error) {
	if c.oauth2Config.RedirectURL != callbackURL {
		return "", fmt.Errorf("expected callback URL %q did not match the URL in the config %q",
			callbackURL, c.oauth2Config.RedirectURL)
	}

	return c.oauth2Config.AuthCodeURL(state), nil
}

// HandleCallback handles HTTP redirect from LinkedIn
func (c *linkedInConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return identity, &oauth2Error{errType, q.Get("error_description")}
	}

	ctx := r.Context()
	token, err := c.oauth2Config.Exchange(ctx, q.Get("code"))
	if err != nil {
		return identity, fmt.Errorf("linkedin: get token: %v", err)
	}

	client := c.oauth2Config.Client(ctx, token)
	profile, err := c.profile(ctx, client)
	if err != nil {
		return identity, fmt.Errorf("linkedin: get profile: %v", err)
	}

	identity = connector.Identity{
		UserID:        profile.ID,
		Username:      profile.fullname(),
		Email:         profile.Email,
		EmailVerified: true,
	}

	if s.OfflineAccess {
		data := connectorData{AccessToken: token.AccessToken}
		connData, err := json.Marshal(data)
		if err != nil {
			return identity, fmt.Errorf("linkedin: marshal connector data: %v", err)
		}
		identity.ConnectorData = connData
	}

	return identity, nil
}

func (c *linkedInConnector) Refresh(ctx context.Context, s connector.Scopes, ident connector.Identity) (connector.Identity, error) {
	if len(ident.ConnectorData) == 0 {
		return ident, fmt.Errorf("linkedin: no upstream access token found")
	}

	var data connectorData
	if err := json.Unmarshal(ident.ConnectorData, &data); err != nil {
		return ident, fmt.Errorf("linkedin: unmarshal access token: %v", err)
	}

	client := c.oauth2Config.Client(ctx, &oauth2.Token{AccessToken: data.AccessToken})
	profile, err := c.profile(ctx, client)
	if err != nil {
		return ident, fmt.Errorf("linkedin: get profile: %v", err)
	}

	ident.Username = profile.fullname()
	ident.Email = profile.Email

	return ident, nil
}

type profile struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"emailAddress"`
}

// fullname returns a full name of a person, or email if the resulting name is
// empty
func (p profile) fullname() string {
	fname := strings.TrimSpace(p.FirstName + " " + p.LastName)
	if fname == "" {
		return p.Email
	}

	return fname
}

func (c *linkedInConnector) profile(ctx context.Context, client *http.Client) (p profile, err error) {
	// https://developer.linkedin.com/docs/fields/basic-profile
	req, err := http.NewRequest("GET", apiURL+"/people/~:(id,first-name,last-name,email-address)", nil)
	if err != nil {
		return p, fmt.Errorf("new req: %v", err)
	}
	q := req.URL.Query()
	q.Add("format", "json")
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return p, fmt.Errorf("get URL %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return p, fmt.Errorf("read body: %v", err)
		}
		return p, fmt.Errorf("%s: %s", resp.Status, body)
	}

	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return p, fmt.Errorf("JSON decode: %v", err)
	}

	if p.Email == "" {
		return p, fmt.Errorf("email is not set")
	}

	return p, err
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

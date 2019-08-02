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

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/log"
)

const (
	apiURL   = "https://api.linkedin.com/v2"
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
func (c *Config) Open(id string, logger log.Logger) (connector.Connector, error) {
	return &linkedInConnector{
		oauth2Config: &oauth2.Config{
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  authURL,
				TokenURL: tokenURL,
			},
			Scopes:      []string{"r_liteprofile", "r_emailaddress"},
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
	logger       log.Logger
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
	FirstName string `json:"localizedFirstName"`
	LastName  string `json:"localizedLastName"`
	Email     string `json:"emailAddress"`
}

type emailresp struct {
	Elements []struct {
		Handle struct {
			EmailAddress string `json:"emailAddress"`
		} `json:"handle~"`
	} `json:"elements"`
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

func (c *linkedInConnector) primaryEmail(ctx context.Context, client *http.Client) (email string, err error) {
	req, err := http.NewRequest("GET", apiURL+"/emailAddress?q=members&projection=(elements*(handle~))", nil)
	if err != nil {
		return email, fmt.Errorf("new req: %v", err)
	}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return email, fmt.Errorf("get URL %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return email, fmt.Errorf("read body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return email, fmt.Errorf("%s: %s", resp.Status, body)
	}

	var parsedResp emailresp
	err = json.Unmarshal(body, &parsedResp)
	if err == nil {
		for _, elem := range parsedResp.Elements {
			email = elem.Handle.EmailAddress
		}
	}

	if email == "" {
		err = fmt.Errorf("email is not set")
	}

	return email, err
}

func (c *linkedInConnector) profile(ctx context.Context, client *http.Client) (p profile, err error) {
	// https://docs.microsoft.com/en-us/linkedin/shared/integrations/people/profile-api
	// https://docs.microsoft.com/en-us/linkedin/shared/integrations/people/primary-contact-api
	// https://docs.microsoft.com/en-us/linkedin/consumer/integrations/self-serve/migration-faq#how-do-i-retrieve-the-members-email-address
	req, err := http.NewRequest("GET", apiURL+"/me", nil)
	if err != nil {
		return p, fmt.Errorf("new req: %v", err)
	}

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

	email, err := c.primaryEmail(ctx, client)
	if err != nil {
		return p, fmt.Errorf("fetching email: %v", err)
	}
	p.Email = email

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

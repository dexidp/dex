// Package facebook provides authentication strategies using Facebook.
package facebook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/dex/connector"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2"
)

const (
	scopeEmail         = "email"
	scopeOfflineAccess = "offline_access"
)

// Config holds configuration options for facebook logins.
type Config struct {
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	RedirectURI  string `json:"redirectURI"`
}

// Open returns a strategy for logging in through Facebook.
func (c *Config) Open(logger logrus.FieldLogger) (connector.Connector, error) {
	return &facebookConnector{c, logger}, nil
}

type connectorData struct {
	// GitHub's OAuth2 tokens never expire. We don't need a refresh token.
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

var (
	_ connector.CallbackConnector = (*facebookConnector)(nil)
	_ connector.RefreshConnector  = (*facebookConnector)(nil)
)

type facebookConnector struct {
	*Config
	logger logrus.FieldLogger
}

func (c *facebookConnector) oauth2Config(scopes connector.Scopes) *oauth2.Config {
	githubScopes := []string{scopeEmail}

	//we ignore scopes.Groups

	if scopes.OfflineAccess {
		githubScopes = append(githubScopes, scopeOfflineAccess)
	}

	return &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURL:  c.RedirectURI,
		Scopes:       []string{"public_profile"},
		Endpoint:     facebook.Endpoint,
	}
}

func (c *facebookConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, error) {
	if c.RedirectURI != callbackURL {
		return "", fmt.Errorf("expected callback URL did not match the URL in the config")
	}

	return c.oauth2Config(scopes).AuthCodeURL(state), nil
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

func (c *facebookConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return identity, &oauth2Error{errType, q.Get("error_description")}
	}

	oauth2Config := c.oauth2Config(s)
	ctx := r.Context()

	token, err := oauth2Config.Exchange(ctx, q.Get("code"))
	if err != nil {
		return identity, fmt.Errorf("facebook: failed to get token: %v", err)
	}

	client := oauth2Config.Client(ctx, token)

	user, err := c.user(ctx, client)
	if err != nil {
		return identity, fmt.Errorf("facebook: get user: %v", err)
	}

	identity = connector.Identity{
		UserID:        user.ID,
		Username:      user.Name,
		Email:         user.Email,
		EmailVerified: true,
	}

	if s.OfflineAccess {
		data := connectorData{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
		}
		connData, err := json.Marshal(data)
		if err != nil {
			return identity, fmt.Errorf("marshal connector data: %v", err)
		}
		identity.ConnectorData = connData
	}

	return identity, nil
}

func (c *facebookConnector) Refresh(ctx context.Context, s connector.Scopes, ident connector.Identity) (connector.Identity, error) {
	if len(ident.ConnectorData) == 0 {
		return ident, errors.New("no upstream access token found")
	}

	var data connectorData
	if err := json.Unmarshal(ident.ConnectorData, &data); err != nil {
		return ident, fmt.Errorf("facebook: unmarshal access token: %v", err)
	}

	client := c.oauth2Config(s).Client(ctx, &oauth2.Token{AccessToken: data.AccessToken})
	user, err := c.user(ctx, client)
	if err != nil {
		return ident, fmt.Errorf("facebook: get user: %v", err)
	}

	ident.Username = user.Name
	ident.Email = user.Email

	return ident, nil
}

type user struct {
	Name  string `json:"name"`
	ID    string `json:"id"`
	Email string `json:"email"`
}

// user queries the Facebook API for profile information using the provided client. The HTTP
// client is expected to be constructed by the golang.org/x/oauth2 package, which inserts
// a bearer token as part of the request.
func (c *facebookConnector) user(ctx context.Context, client *http.Client) (user, error) {
	var u user
	req, err := http.NewRequest("GET", "https://graph.facebook.com/me?fields=id,name,email", nil)
	if err != nil {
		return u, fmt.Errorf("facebook: new req: %v", err)
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return u, fmt.Errorf("facebook: get URL %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return u, fmt.Errorf("facebook: read body: %v", err)
		}
		return u, fmt.Errorf("%s: %s", resp.Status, body)
	}

	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return u, fmt.Errorf("failed to decode response: %v", err)
	}
	return u, nil
}

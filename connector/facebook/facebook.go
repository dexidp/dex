// Package facebook provides authentication strategies using Facebook.
package facebook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/coreos/dex/connector"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	fb "golang.org/x/oauth2/facebook"
)

const (
	baseURL            = "https://graph.facebook.com"
	scopeEmail         = "email"
	scopePublicProfile = "public_profile"
	scopeUserFriends   = "user_friends"
)

// Config holds configuration options for facebook logins.
type Config struct {
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	RedirectURI  string `json:"redirectURI"`
	Org          string `json:"org"`
	AuthType     string `json:"authType"`
}

// Open returns a strategy for logging in through Facebook.
func (c *Config) Open(id string, logger logrus.FieldLogger) (connector.Connector, error) {
	return &facebookConnector{
		redirectURI:  c.RedirectURI,
		org:          c.Org,
		clientID:     c.ClientID,
		clientSecret: c.ClientSecret,
		logger:       logger,
	}, nil
}

type connectorData struct {
	// Facebook's OAuth2 tokens never expire. We don't need a refresh token.
	AccessToken string `json:"accessToken"`
}

var (
	_ connector.CallbackConnector = (*facebookConnector)(nil)
	_ connector.RefreshConnector  = (*facebookConnector)(nil)
)

type facebookConnector struct {
	redirectURI  string
	org          string
	clientID     string
	clientSecret string
	logger       logrus.FieldLogger
}

func (c *facebookConnector) oauth2Config(scopes connector.Scopes) *oauth2.Config {
	var facebookScopes []string = []string{scopeEmail, scopePublicProfile, scopeUserFriends}
	return &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     fb.Endpoint,
		Scopes:       facebookScopes,
		RedirectURL:  c.redirectURI,
	}
}

func (c *facebookConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, error) {
	if c.redirectURI != callbackURL {
		return "", fmt.Errorf("expected callback URL did not match the URL in the config %s %s", c.redirectURI, callbackURL)
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
	code := q.Get("code")
	if errType := q.Get("error"); errType != "" {
		return identity, &oauth2Error{errType, q.Get("error_description")}
	}

	oauth2Config := c.oauth2Config(s)
	ctx := r.Context()

	token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		return identity, fmt.Errorf("facebook: failed to get token: %v", err)
	}

	client := oauth2Config.Client(ctx, token)
	user, err := c.user(ctx, client)
	if err != nil {
		return identity, fmt.Errorf("facebook: get user: %v", err)
	}

	identity = connector.Identity{
		UserID:   user.ID,
		Username: user.ID + "," + user.FirstName + "," + user.LastName,
		Email:    user.Email,
	}

	if s.OfflineAccess {
		data := connectorData{AccessToken: token.AccessToken}
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

	username := user.Name
	ident.Username = username
	ident.Email = user.Email
	return ident, nil
}

// user queries the Facebook API for profile information using the provided client. The HTTP
// client is expected to be constructed by the golang.org/x/oauth2 package, which inserts
// a bearer token as part of the request.
func (c *facebookConnector) user(ctx context.Context, client *http.Client) (user, error) {
	var u user
	req, err := http.NewRequest("GET", baseURL+"/me?fields=id,name,first_name,last_name,email", nil)
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
		fmt.Printf("%v", body)
		return u, fmt.Errorf("%s: %s", resp.Status, body)
	}

	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return u, fmt.Errorf("failed to decode response: %v", err)
	}
	return u, nil
}

type user struct {
	Name      string `json:"name"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	ID        string `json:"id"`
	Email     string `json:"email"`
}

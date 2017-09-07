// Package facebook provides authentication strategies using LinkedIn.
package linkedin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/coreos/dex/connector"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	ln "golang.org/x/oauth2/linkedin"
)

const (
	baseURL            = "https://api.linkedin.com"
	scopeEmail         = "r_emailaddress"
	scopePublicProfile = "r_basicprofile"
)

// Config holds configuration options for linkedin logins.
type Config struct {
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	RedirectURI  string `json:"redirectURI"`
	Org          string `json:"org"`
	AuthType     string `json:"authType"`
}

// Open returns a strategy for logging in through LinkedIn.
func (c *Config) Open(logger logrus.FieldLogger) (connector.Connector, error) {
	return &linkedinConnector{
		redirectURI:  c.RedirectURI,
		org:          c.Org,
		clientID:     c.ClientID,
		clientSecret: c.ClientSecret,
		logger:       logger,
	}, nil
}

type connectorData struct {
	// LinkedIn's OAuth2 tokens never expire. We don't need a refresh token.
	AccessToken string `json:"accessToken"`
}

var (
	_ connector.CallbackConnector = (*linkedinConnector)(nil)
	_ connector.RefreshConnector  = (*linkedinConnector)(nil)
)

type linkedinConnector struct {
	redirectURI  string
	org          string
	clientID     string
	clientSecret string
	logger       logrus.FieldLogger
}

func (c *linkedinConnector) oauth2Config(scopes connector.Scopes) *oauth2.Config {
	var linkedinScopes []string = []string{scopeEmail, scopePublicProfile}
	return &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     ln.Endpoint,
		Scopes:       linkedinScopes,
		RedirectURL:  c.redirectURI,
	}
}

func (c *linkedinConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, error) {
	if c.redirectURI != callbackURL {
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

func (c *linkedinConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	q := r.URL.Query()
	code := q.Get("code")
	if errType := q.Get("error"); errType != "" {
		return identity, &oauth2Error{errType, q.Get("error_description")}
	}

	oauth2Config := c.oauth2Config(s)
	ctx := r.Context()

	token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		return identity, fmt.Errorf("linkedin: failed to get token: %v", err)
	}

	client := oauth2Config.Client(ctx, token)
	user, err := c.user(ctx, client)
	if err != nil {
		return identity, fmt.Errorf("linkedin: get user: %v", err)
	}
	identity = connector.Identity{
		UserID:        user.ID,
		Username:      user.ID + "," + user.FirstName + "," + user.LastName,
		Email:         user.Email,
		EmailVerified: true,
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

func (c *linkedinConnector) Refresh(ctx context.Context, s connector.Scopes, ident connector.Identity) (connector.Identity, error) {
	if len(ident.ConnectorData) == 0 {
		return ident, errors.New("no upstream access token found")
	}

	var data connectorData
	if err := json.Unmarshal(ident.ConnectorData, &data); err != nil {
		return ident, fmt.Errorf("linkedin: unmarshal access token: %v", err)
	}

	client := c.oauth2Config(s).Client(ctx, &oauth2.Token{AccessToken: data.AccessToken})
	user, err := c.user(ctx, client)
	if err != nil {
		return ident, fmt.Errorf("linkedin: get user: %v", err)
	}

	username := user.Name
	ident.Username = username
	ident.Email = user.Email
	return ident, nil
}

type user struct {
	Name      string `json:"formattedName"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	ID        string `json:"publicProfileUrl"`
	Email     string `json:"emailAddress"`
}

// user queries the LinkedIn API for profile information using the provided client. The HTTP
// client is expected to be constructed by the golang.org/x/oauth2 package, which inserts
// a bearer token as part of the request.
func (c *linkedinConnector) user(ctx context.Context, client *http.Client) (user, error) {
	var u user
	req, err := http.NewRequest("GET", baseURL+"/v1/people/~:(id,first-name,last-name,email-address,public-profile-url)?format=json", nil)
	if err != nil {
		return u, fmt.Errorf("linkedin: new req: %v", err)
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return u, fmt.Errorf("linkedin: get URL %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return u, fmt.Errorf("linkedin: read body: %v", err)
		}
		fmt.Printf("%v", body)
		return u, fmt.Errorf("%s: %s", resp.Status, body)
	}

	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return u, fmt.Errorf("failed to decode response: %v", err)
	}
	lastBin := strings.LastIndex(u.ID, "/")
	u.ID = u.ID[lastBin+1 : len(u.ID)]
	return u, nil
}

type utcFormatter struct {
	f logrus.Formatter
}

func (f *utcFormatter) Format(e *logrus.Entry) ([]byte, error) {
	e.Time = e.Time.UTC()
	return f.f.Format(e)
}

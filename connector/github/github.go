// Package github provides authentication strategies using GitHub.
package github

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/coreos/dex/connector"
)

const baseURL = "https://api.github.com"

// Config holds configuration options for github logins.
type Config struct {
	ClientID     string `yaml:"clientID"`
	ClientSecret string `yaml:"clientSecret"`
	RedirectURI  string `yaml:"redirectURI"`
	Org          string `yaml:"org"`
}

// Open returns a strategy for logging in through GitHub.
func (c *Config) Open() (connector.Connector, error) {
	return &githubConnector{
		redirectURI: c.RedirectURI,
		org:         c.Org,
		oauth2Config: &oauth2.Config{
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			Endpoint:     github.Endpoint,
			Scopes: []string{
				"user:email", // View user's email
				"read:org",   // View user's org teams.
			},
		},
	}, nil
}

type connectorData struct {
	// GitHub's OAuth2 tokens never expire. We don't need a refresh token.
	AccessToken string `json:"accessToken"`
}

var (
	_ connector.CallbackConnector = (*githubConnector)(nil)
	_ connector.GroupsConnector   = (*githubConnector)(nil)
)

type githubConnector struct {
	redirectURI  string
	org          string
	oauth2Config *oauth2.Config
	ctx          context.Context
	cancel       context.CancelFunc
}

func (c *githubConnector) Close() error {
	return nil
}

func (c *githubConnector) LoginURL(callbackURL, state string) (string, error) {
	if c.redirectURI != callbackURL {
		return "", fmt.Errorf("expected callback URL did not match the URL in the config")
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

func (c *githubConnector) HandleCallback(r *http.Request) (identity connector.Identity, err error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return identity, &oauth2Error{errType, q.Get("error_description")}
	}
	token, err := c.oauth2Config.Exchange(c.ctx, q.Get("code"))
	if err != nil {
		return identity, fmt.Errorf("github: failed to get token: %v", err)
	}

	resp, err := c.oauth2Config.Client(c.ctx, token).Get(baseURL + "/user")
	if err != nil {
		return identity, fmt.Errorf("github: get URL %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return identity, fmt.Errorf("github: read body: %v", err)
		}
		return identity, fmt.Errorf("%s: %s", resp.Status, body)
	}
	var user struct {
		Name  string `json:"name"`
		Login string `json:"login"`
		ID    int    `json:"id"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return identity, fmt.Errorf("failed to decode response: %v", err)
	}

	data := connectorData{AccessToken: token.AccessToken}
	connData, err := json.Marshal(data)
	if err != nil {
		return identity, fmt.Errorf("marshal connector data: %v", err)
	}

	username := user.Name
	if username == "" {
		username = user.Login
	}
	identity = connector.Identity{
		UserID:        strconv.Itoa(user.ID),
		Username:      username,
		Email:         user.Email,
		EmailVerified: true,
		ConnectorData: connData,
	}
	return identity, nil
}

func (c *githubConnector) Groups(identity connector.Identity) ([]string, error) {
	var data connectorData
	if err := json.Unmarshal(identity.ConnectorData, &data); err != nil {
		return nil, fmt.Errorf("decode connector data: %v", err)
	}
	token := &oauth2.Token{AccessToken: data.AccessToken}
	resp, err := c.oauth2Config.Client(c.ctx, token).Get(baseURL + "/user/teams")
	if err != nil {
		return nil, fmt.Errorf("github: get teams: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("github: read body: %v", err)
		}
		return nil, fmt.Errorf("%s: %s", resp.Status, body)
	}

	// https://developer.github.com/v3/orgs/teams/#response-12
	var teams []struct {
		Name string `json:"name"`
		Org  struct {
			Login string `json:"login"`
		} `json:"organization"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&teams); err != nil {
		return nil, fmt.Errorf("github: unmarshal groups: %v", err)
	}
	groups := []string{}
	for _, team := range teams {
		if team.Org.Login == c.org {
			groups = append(groups, team.Name)
		}
	}
	return groups, nil
}

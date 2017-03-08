// Package github provides authentication strategies using GitHub.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/dex/connector"
)

const (
	baseURL    = "https://api.github.com"
	scopeEmail = "user:email"
	scopeOrgs  = "read:org"
)

// Config holds configuration options for github logins.
type Config struct {
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	RedirectURI  string `json:"redirectURI"`
	Org          string `json:"org"`
}

// Open returns a strategy for logging in through GitHub.
func (c *Config) Open(logger logrus.FieldLogger) (connector.Connector, error) {
	return &githubConnector{
		redirectURI:  c.RedirectURI,
		org:          c.Org,
		clientID:     c.ClientID,
		clientSecret: c.ClientSecret,
		logger:       logger,
	}, nil
}

type connectorData struct {
	// GitHub's OAuth2 tokens never expire. We don't need a refresh token.
	AccessToken string `json:"accessToken"`
}

var (
	_ connector.CallbackConnector = (*githubConnector)(nil)
	_ connector.RefreshConnector  = (*githubConnector)(nil)
)

type githubConnector struct {
	redirectURI  string
	org          string
	clientID     string
	clientSecret string
	logger       logrus.FieldLogger
}

func (c *githubConnector) oauth2Config(scopes connector.Scopes) *oauth2.Config {
	var githubScopes []string
	if scopes.Groups {
		githubScopes = []string{scopeEmail, scopeOrgs}
	} else {
		githubScopes = []string{scopeEmail}
	}
	return &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     github.Endpoint,
		Scopes:       githubScopes,
	}
}

func (c *githubConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, error) {
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

func (c *githubConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return identity, &oauth2Error{errType, q.Get("error_description")}
	}

	oauth2Config := c.oauth2Config(s)
	ctx := r.Context()

	token, err := oauth2Config.Exchange(ctx, q.Get("code"))
	if err != nil {
		return identity, fmt.Errorf("github: failed to get token: %v", err)
	}

	client := oauth2Config.Client(ctx, token)

	user, err := c.user(ctx, client)
	if err != nil {
		return identity, fmt.Errorf("github: get user: %v", err)
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
	}

	if s.Groups && c.org != "" {
		groups, err := c.teams(ctx, client, c.org)
		if err != nil {
			return identity, fmt.Errorf("github: get teams: %v", err)
		}
		identity.Groups = groups
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

func (c *githubConnector) Refresh(ctx context.Context, s connector.Scopes, ident connector.Identity) (connector.Identity, error) {
	if len(ident.ConnectorData) == 0 {
		return ident, errors.New("no upstream access token found")
	}

	var data connectorData
	if err := json.Unmarshal(ident.ConnectorData, &data); err != nil {
		return ident, fmt.Errorf("github: unmarshal access token: %v", err)
	}

	client := c.oauth2Config(s).Client(ctx, &oauth2.Token{AccessToken: data.AccessToken})
	user, err := c.user(ctx, client)
	if err != nil {
		return ident, fmt.Errorf("github: get user: %v", err)
	}

	username := user.Name
	if username == "" {
		username = user.Login
	}
	ident.Username = username
	ident.Email = user.Email

	if s.Groups && c.org != "" {
		groups, err := c.teams(ctx, client, c.org)
		if err != nil {
			return ident, fmt.Errorf("github: get teams: %v", err)
		}
		ident.Groups = groups
	}
	return ident, nil
}

type user struct {
	Name  string `json:"name"`
	Login string `json:"login"`
	ID    int    `json:"id"`
	Email string `json:"email"`
}

// user queries the GitHub API for profile information using the provided client. The HTTP
// client is expected to be constructed by the golang.org/x/oauth2 package, which inserts
// a bearer token as part of the request.
func (c *githubConnector) user(ctx context.Context, client *http.Client) (user, error) {
	var u user
	req, err := http.NewRequest("GET", baseURL+"/user", nil)
	if err != nil {
		return u, fmt.Errorf("github: new req: %v", err)
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return u, fmt.Errorf("github: get URL %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return u, fmt.Errorf("github: read body: %v", err)
		}
		return u, fmt.Errorf("%s: %s", resp.Status, body)
	}

	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return u, fmt.Errorf("failed to decode response: %v", err)
	}
	return u, nil
}

// teams queries the GitHub API for team membership within a specific organization.
//
// The HTTP passed client is expected to be constructed by the golang.org/x/oauth2 package,
// which inserts a bearer token as part of the request.
func (c *githubConnector) teams(ctx context.Context, client *http.Client, org string) ([]string, error) {

	groups := []string{}

	// https://developer.github.com/v3/#pagination
	reNext := regexp.MustCompile("<(.*)>; rel=\"next\"")
	reLast := regexp.MustCompile("<(.*)>; rel=\"last\"")
	apiURL := baseURL + "/user/teams"

	for {
		req, err := http.NewRequest("GET", apiURL, nil)

		if err != nil {
			return nil, fmt.Errorf("github: new req: %v", err)
		}
		req = req.WithContext(ctx)
		resp, err := client.Do(req)
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

		for _, team := range teams {
			if team.Org.Login == org {
				groups = append(groups, team.Name)
			}
		}

		links := resp.Header.Get("Link")
		if len(reLast.FindStringSubmatch(links)) > 1 {
			lastPageURL := reLast.FindStringSubmatch(links)[1]
			if apiURL == lastPageURL {
				break
			}
		} else {
			break
		}

		if len(reNext.FindStringSubmatch(links)) > 1 {
			apiURL = reNext.FindStringSubmatch(links)[1]
		} else {
			break
		}
	}
	return groups, nil
}

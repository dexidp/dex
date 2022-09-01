// Package gitea provides authentication strategies using Gitea.
package gitea

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/log"
)

// Config holds configuration options for gitea logins.
type Config struct {
	BaseURL       string `json:"baseURL"`
	ClientID      string `json:"clientID"`
	ClientSecret  string `json:"clientSecret"`
	RedirectURI   string `json:"redirectURI"`
	Orgs          []Org  `json:"orgs"`
	LoadAllGroups bool   `json:"loadAllGroups"`
	UseLoginAsID  bool   `json:"useLoginAsID"`
}

// Org holds org-team filters, in which teams are optional.
type Org struct {
	// Organization name in gitea (not slug, full name). Only users in this gitea
	// organization can authenticate.
	Name string `json:"name"`

	// Names of teams in a gitea organization. A user will be able to
	// authenticate if they are members of at least one of these teams. Users
	// in the organization can authenticate if this field is omitted from the
	// config file.
	Teams []string `json:"teams,omitempty"`
}

type giteaUser struct {
	ID       int    `json:"id"`
	Name     string `json:"full_name"`
	Username string `json:"login"`
	Email    string `json:"email"`
	IsAdmin  bool   `json:"is_admin"`
}

// Open returns a strategy for logging in through Gitea
func (c *Config) Open(id string, logger log.Logger) (connector.Connector, error) {
	if c.BaseURL == "" {
		c.BaseURL = "https://gitea.com"
	}
	return &giteaConnector{
		baseURL:       c.BaseURL,
		redirectURI:   c.RedirectURI,
		orgs:          c.Orgs,
		clientID:      c.ClientID,
		clientSecret:  c.ClientSecret,
		logger:        logger,
		loadAllGroups: c.LoadAllGroups,
		useLoginAsID:  c.UseLoginAsID,
	}, nil
}

type connectorData struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	Expiry       time.Time `json:"expiry"`
}

var (
	_ connector.CallbackConnector = (*giteaConnector)(nil)
	_ connector.RefreshConnector  = (*giteaConnector)(nil)
)

type giteaConnector struct {
	baseURL      string
	redirectURI  string
	orgs         []Org
	clientID     string
	clientSecret string
	logger       log.Logger
	httpClient   *http.Client
	// if set to true and no orgs are configured then connector loads all user claims (all orgs and team)
	loadAllGroups bool
	// if set to true will use the user's handle rather than their numeric id as the ID
	useLoginAsID bool
}

func (c *giteaConnector) oauth2Config(_ connector.Scopes) *oauth2.Config {
	giteaEndpoint := oauth2.Endpoint{AuthURL: c.baseURL + "/login/oauth/authorize", TokenURL: c.baseURL + "/login/oauth/access_token"}
	return &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     giteaEndpoint,
		RedirectURL:  c.redirectURI,
	}
}

func (c *giteaConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, error) {
	if c.redirectURI != callbackURL {
		return "", fmt.Errorf("expected callback URL %q did not match the URL in the config %q", c.redirectURI, callbackURL)
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

func (c *giteaConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return identity, &oauth2Error{errType, q.Get("error_description")}
	}

	oauth2Config := c.oauth2Config(s)

	ctx := r.Context()
	if c.httpClient != nil {
		ctx = context.WithValue(r.Context(), oauth2.HTTPClient, c.httpClient)
	}

	token, err := oauth2Config.Exchange(ctx, q.Get("code"))
	if err != nil {
		return identity, fmt.Errorf("gitea: failed to get token: %v", err)
	}

	client := oauth2Config.Client(ctx, token)

	user, err := c.user(ctx, client)
	if err != nil {
		return identity, fmt.Errorf("gitea: get user: %v", err)
	}

	username := user.Name
	if username == "" {
		username = user.Email
	}

	identity = connector.Identity{
		UserID:            strconv.Itoa(user.ID),
		Username:          username,
		PreferredUsername: user.Username,
		Email:             user.Email,
		EmailVerified:     true,
	}
	if c.useLoginAsID {
		identity.UserID = user.Username
	}

	// Only set identity.Groups if 'orgs', 'org', or 'groups' scope are specified.
	if c.groupsRequired() {
		groups, err := c.getGroups(ctx, client)
		if err != nil {
			return identity, err
		}
		identity.Groups = groups
	}

	if s.OfflineAccess {
		data := connectorData{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			Expiry:       token.Expiry,
		}
		connData, err := json.Marshal(data)
		if err != nil {
			return identity, fmt.Errorf("gitea: marshal connector data: %v", err)
		}
		identity.ConnectorData = connData
	}

	return identity, nil
}

// Refreshing tokens
// https://github.com/golang/oauth2/issues/84#issuecomment-332860871
type tokenNotifyFunc func(*oauth2.Token) error

// notifyRefreshTokenSource is essentially `oauth2.ReuseTokenSource` with `TokenNotifyFunc` added.
type notifyRefreshTokenSource struct {
	new oauth2.TokenSource
	mu  sync.Mutex // guards t
	t   *oauth2.Token
	f   tokenNotifyFunc // called when token refreshed so new refresh token can be persisted
}

// Token returns the current token if it's still valid, else will
// refresh the current token (using r.Context for HTTP client
// information) and return the new one.
func (s *notifyRefreshTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.t.Valid() {
		return s.t, nil
	}
	t, err := s.new.Token()
	if err != nil {
		return nil, err
	}
	s.t = t
	return t, s.f(t)
}

func (c *giteaConnector) Refresh(ctx context.Context, s connector.Scopes, ident connector.Identity) (connector.Identity, error) {
	if len(ident.ConnectorData) == 0 {
		return ident, errors.New("gitea: no upstream access token found")
	}

	var data connectorData
	if err := json.Unmarshal(ident.ConnectorData, &data); err != nil {
		return ident, fmt.Errorf("gitea: unmarshal access token: %v", err)
	}

	tok := &oauth2.Token{
		AccessToken:  data.AccessToken,
		RefreshToken: data.RefreshToken,
		Expiry:       data.Expiry,
	}

	client := oauth2.NewClient(ctx, &notifyRefreshTokenSource{
		new: c.oauth2Config(s).TokenSource(ctx, tok),
		t:   tok,
		f: func(tok *oauth2.Token) error {
			data := connectorData{
				AccessToken:  tok.AccessToken,
				RefreshToken: tok.RefreshToken,
				Expiry:       tok.Expiry,
			}
			connData, err := json.Marshal(data)
			if err != nil {
				return fmt.Errorf("gitea: marshal connector data: %v", err)
			}
			ident.ConnectorData = connData
			return nil
		},
	})
	user, err := c.user(ctx, client)
	if err != nil {
		return ident, fmt.Errorf("gitea: get user: %v", err)
	}

	username := user.Name
	if username == "" {
		username = user.Email
	}
	ident.Username = username
	ident.PreferredUsername = user.Username
	ident.Email = user.Email

	// Only set identity.Groups if 'orgs', 'org', or 'groups' scope are specified.
	if c.groupsRequired() {
		groups, err := c.getGroups(ctx, client)
		if err != nil {
			return ident, err
		}
		ident.Groups = groups
	}

	return ident, nil
}

// getGroups retrieves Gitea orgs and teams a user is in, if any.
func (c *giteaConnector) getGroups(ctx context.Context, client *http.Client) ([]string, error) {
	if len(c.orgs) > 0 {
		return c.groupsForOrgs(ctx, client)
	} else if c.loadAllGroups {
		return c.userGroups(ctx, client)
	}
	return nil, nil
}

// formatTeamName returns unique team name.
// Orgs might have the same team names. To make team name unique it should be prefixed with the org name.
func formatTeamName(org string, team string) string {
	return fmt.Sprintf("%s:%s", org, team)
}

// groupsForOrgs returns list of groups that user belongs to in approved list
func (c *giteaConnector) groupsForOrgs(ctx context.Context, client *http.Client) ([]string, error) {
	groups, err := c.userGroups(ctx, client)
	if err != nil {
		return groups, err
	}

	keys := make(map[string]bool)
	for _, o := range c.orgs {
		keys[o.Name] = true
		if o.Teams != nil {
			for _, t := range o.Teams {
				keys[formatTeamName(o.Name, t)] = true
			}
		}
	}
	atLeastOne := false
	filteredGroups := make([]string, 0)
	for _, g := range groups {
		if _, value := keys[g]; value {
			filteredGroups = append(filteredGroups, g)
			atLeastOne = true
		}
	}

	if !atLeastOne {
		return []string{}, fmt.Errorf("gitea: User does not belong to any of the approved groups")
	}
	return filteredGroups, nil
}

type organization struct {
	ID   int64  `json:"id"`
	Name string `json:"username"`
}

type team struct {
	ID           int64         `json:"id"`
	Name         string        `json:"name"`
	Organization *organization `json:"organization"`
}

func (c *giteaConnector) userGroups(ctx context.Context, client *http.Client) ([]string, error) {
	apiURL := c.baseURL + "/api/v1/user/teams"
	groups := make([]string, 0)
	page := 1
	limit := 20
	for {
		var teams []team
		req, err := http.NewRequest("GET", fmt.Sprintf("%s?page=%d&limit=%d", apiURL, page, limit), nil)
		if err != nil {
			return groups, fmt.Errorf("gitea: new req: %v", err)
		}

		req = req.WithContext(ctx)
		resp, err := client.Do(req)
		if err != nil {
			return groups, fmt.Errorf("gitea: get URL %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return groups, fmt.Errorf("gitea: read body: %v", err)
			}
			return groups, fmt.Errorf("%s: %s", resp.Status, body)
		}

		if err := json.NewDecoder(resp.Body).Decode(&teams); err != nil {
			return groups, fmt.Errorf("failed to decode response: %v", err)
		}

		if len(teams) == 0 {
			break
		}

		for _, t := range teams {
			groups = append(groups, t.Organization.Name)
			groups = append(groups, formatTeamName(t.Organization.Name, t.Name))
		}

		page++
	}

	// remove duplicate slice variables
	keys := make(map[string]struct{})
	list := []string{}
	for _, group := range groups {
		if _, exists := keys[group]; !exists {
			keys[group] = struct{}{}
			list = append(list, group)
		}
	}
	groups = list
	return groups, nil
}

// user queries the Gitea API for profile information using the provided client. The HTTP
// client is expected to be constructed by the golang.org/x/oauth2 package, which inserts
// a bearer token as part of the request.
func (c *giteaConnector) user(ctx context.Context, client *http.Client) (giteaUser, error) {
	var u giteaUser
	req, err := http.NewRequest("GET", c.baseURL+"/api/v1/user", nil)
	if err != nil {
		return u, fmt.Errorf("gitea: new req: %v", err)
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return u, fmt.Errorf("gitea: get URL %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return u, fmt.Errorf("gitea: read body: %v", err)
		}
		return u, fmt.Errorf("%s: %s", resp.Status, body)
	}

	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return u, fmt.Errorf("failed to decode response: %v", err)
	}
	return u, nil
}

// groupsRequired returns whether dex needs to request groups from Gitea.
func (c *giteaConnector) groupsRequired() bool {
	return len(c.orgs) > 0 || c.loadAllGroups
}

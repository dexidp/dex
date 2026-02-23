// Package gitlab provides authentication strategies using GitLab.
package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/groups"
	"github.com/dexidp/dex/pkg/httpclient"
)

const (
	// read operations of the /api/v4/user endpoint
	scopeUser = "read_user"
	// used to retrieve groups from /oauth/userinfo
	// https://docs.gitlab.com/ee/integration/openid_connect_provider.html
	scopeOpenID = "openid"
	// read operations of the /api/v4/groups endpoint
	scopeReadApi = "read_api"
)

// Config holds configuration options for gitlab logins.
type Config struct {
	BaseURL             string   `json:"baseURL"`
	ClientID            string   `json:"clientID"`
	ClientSecret        string   `json:"clientSecret"`
	RedirectURI         string   `json:"redirectURI"`
	Groups              []string `json:"groups"`
	UseLoginAsID        bool     `json:"useLoginAsID"`
	GetGroupsPermission bool     `json:"getGroupsPermission"`
	RootCAData          []byte   `json:"rootCAData,omitempty"`
}

type gitlabUser struct {
	ID       int
	Name     string
	Username string
	State    string
	Email    string
	IsAdmin  bool
}

// Open returns a strategy for logging in through GitLab.
func (c *Config) Open(id string, logger *slog.Logger) (connector.Connector, error) {
	if c.BaseURL == "" {
		c.BaseURL = "https://gitlab.com"
	}
	var httpClient *http.Client
	if len(c.RootCAData) > 0 {
		var err error
		httpClient, err = httpclient.NewHTTPClient([]string{string(c.RootCAData)}, false)
		if err != nil {
			// Keep backward-compatible error semantics for invalid PEM input.
			if strings.Contains(err.Error(), "not in PEM format") {
				return nil, fmt.Errorf("gitlab: invalid rootCAData")
			}
			return nil, fmt.Errorf("gitlab: failed to create HTTP client: %v", err)
		}
		httpClient.Timeout = 30 * time.Second
	}
	return &gitlabConnector{
		baseURL:             c.BaseURL,
		redirectURI:         c.RedirectURI,
		clientID:            c.ClientID,
		clientSecret:        c.ClientSecret,
		logger:              logger.With(slog.Group("connector", "type", "gitlab", "id", id)),
		groups:              c.Groups,
		useLoginAsID:        c.UseLoginAsID,
		getGroupsPermission: c.GetGroupsPermission,
		httpClient:          httpClient,
	}, nil
}

type connectorData struct {
	// Support GitLab's Access Tokens and Refresh tokens.
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

var (
	_ connector.CallbackConnector = (*gitlabConnector)(nil)
	_ connector.RefreshConnector  = (*gitlabConnector)(nil)
)

type gitlabConnector struct {
	baseURL      string
	redirectURI  string
	groups       []string
	clientID     string
	clientSecret string
	logger       *slog.Logger
	httpClient   *http.Client
	// if set to true will use the user's handle rather than their numeric id as the ID
	useLoginAsID bool

	// if set to true permissions will be added to list of groups
	getGroupsPermission bool
}

func (c *gitlabConnector) oauth2Config(scopes connector.Scopes) *oauth2.Config {
	gitlabScopes := []string{scopeUser}
	if c.groupsRequired(scopes.Groups) {
		gitlabScopes = []string{scopeUser, scopeOpenID, scopeReadApi}
	}

	gitlabEndpoint := oauth2.Endpoint{AuthURL: c.baseURL + "/oauth/authorize", TokenURL: c.baseURL + "/oauth/token"}
	return &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     gitlabEndpoint,
		Scopes:       gitlabScopes,
		RedirectURL:  c.redirectURI,
	}
}

func (c *gitlabConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, []byte, error) {
	if c.redirectURI != callbackURL {
		return "", nil, fmt.Errorf("expected callback URL %q did not match the URL in the config %q", c.redirectURI, callbackURL)
	}
	return c.oauth2Config(scopes).AuthCodeURL(state), nil, nil
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

func (c *gitlabConnector) HandleCallback(s connector.Scopes, connData []byte, r *http.Request) (identity connector.Identity, err error) {
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
		return identity, fmt.Errorf("gitlab: failed to get token: %v", err)
	}

	return c.identity(ctx, s, token)
}

func (c *gitlabConnector) identity(ctx context.Context, s connector.Scopes, token *oauth2.Token) (identity connector.Identity, err error) {
	oauth2Config := c.oauth2Config(s)
	client := oauth2Config.Client(ctx, token)

	user, err := c.user(ctx, client)
	if err != nil {
		return identity, fmt.Errorf("gitlab: get user: %v", err)
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

	if c.groupsRequired(s.Groups) {
		groups, err := c.getGroups(ctx, client, s.Groups, user.Username, user.ID)
		if err != nil {
			return identity, fmt.Errorf("gitlab: get groups: %v", err)
		}
		identity.Groups = groups
	}

	if s.OfflineAccess {
		data := connectorData{RefreshToken: token.RefreshToken, AccessToken: token.AccessToken}
		connData, err := json.Marshal(data)
		if err != nil {
			return identity, fmt.Errorf("gitlab: marshal connector data: %v", err)
		}
		identity.ConnectorData = connData
	}

	return identity, nil
}

func (c *gitlabConnector) Refresh(ctx context.Context, s connector.Scopes, ident connector.Identity) (connector.Identity, error) {
	var data connectorData
	if err := json.Unmarshal(ident.ConnectorData, &data); err != nil {
		return ident, fmt.Errorf("gitlab: unmarshal connector data: %v", err)
	}
	oauth2Config := c.oauth2Config(s)

	if c.httpClient != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	}

	switch {
	case data.RefreshToken != "":
		{
			t := &oauth2.Token{
				RefreshToken: data.RefreshToken,
				Expiry:       time.Now().Add(-time.Hour),
			}
			token, err := oauth2Config.TokenSource(ctx, t).Token()
			if err != nil {
				return ident, fmt.Errorf("gitlab: failed to get refresh token: %v", err)
			}
			return c.identity(ctx, s, token)
		}
	case data.AccessToken != "":
		{
			token := &oauth2.Token{
				AccessToken: data.AccessToken,
			}
			return c.identity(ctx, s, token)
		}
	default:
		return ident, errors.New("no refresh or access token found")
	}
}

func (c *gitlabConnector) groupsRequired(groupScope bool) bool {
	return len(c.groups) > 0 || groupScope
}

// user queries the GitLab API for profile information using the provided client. The HTTP
// client is expected to be constructed by the golang.org/x/oauth2 package, which inserts
// a bearer token as part of the request.
func (c *gitlabConnector) user(ctx context.Context, client *http.Client) (gitlabUser, error) {
	var u gitlabUser
	req, err := http.NewRequest("GET", c.baseURL+"/api/v4/user", nil)
	if err != nil {
		return u, fmt.Errorf("gitlab: new req: %v", err)
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return u, fmt.Errorf("gitlab: get URL %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return u, fmt.Errorf("gitlab: read body: %v", err)
		}
		return u, fmt.Errorf("%s: %s", resp.Status, body)
	}

	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return u, fmt.Errorf("failed to decode response: %v", err)
	}
	return u, nil
}

type group struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	FullName string `json:"full_name"`
	FullPath string `json:"full_path"`
}

// userGroups queries the GitLab API for group membership.
//
// The HTTP passed client is expected to be constructed by the golang.org/x/oauth2 package,
// which inserts a bearer token as part of the request.
func (c *gitlabConnector) userGroups(ctx context.Context, client *http.Client, userId int) ([]string, error) {
	groupsRaw, err := c.getGitlabGroups(ctx, client)
	if err != nil {
		return []string{}, err
	}
	groups := []string{}
	for _, group := range groupsRaw {
		groupMembership, notMember, err := c.getGroupsMembership(ctx, client, group.ID, userId)
		if err != nil {
			return []string{}, fmt.Errorf("gitlab: get group membership: %v", err)
		}

		if _, ok := groupMembership["access_level"]; notMember || !ok {
			continue
		}

		groups = append(groups, group.FullPath)

		if !c.getGroupsPermission {
			continue
		}

		switch groupMembership["access_level"].(float64) {
		case 10:
			groups = append(groups, fmt.Sprintf("%s:guest", group.FullPath))
			break
		case 20:
			groups = append(groups, fmt.Sprintf("%s:reporter", group.FullPath))
			break
		case 30:
			groups = append(groups, fmt.Sprintf("%s:developer", group.FullPath))
			break
		case 40:
			groups = append(groups, fmt.Sprintf("%s:maintainer", group.FullPath))
			break
		case 50:
			groups = append(groups, fmt.Sprintf("%s:owner", group.FullPath))
			break
		case 60:
			groups = append(groups, fmt.Sprintf("%s:admin", group.FullPath))
			break
		}
	}

	return groups, nil
}

func (c *gitlabConnector) getGitlabGroups(ctx context.Context, client *http.Client) ([]group, error) {
	var group []group
	req, err := http.NewRequest("GET", c.baseURL+"/api/v4/groups", nil)
	if err != nil {
		return nil, fmt.Errorf("gitlab: new req: %v", err)
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitlab: get URL %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gitlab: read body: %v", err)
		}
		return nil, fmt.Errorf("%s: %s", resp.Status, body)
	}
	if err = json.NewDecoder(resp.Body).Decode(&group); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	return group, nil
}

func (c *gitlabConnector) getGroupsMembership(ctx context.Context, client *http.Client, groupId int, userId int) (map[string]interface{}, bool, error) {
	var groupMembership map[string]interface{}
	req, err := http.NewRequest("GET", c.baseURL+fmt.Sprintf("/api/v4/groups/%d/members/all/%d", groupId, userId), nil)
	if err != nil {
		return map[string]interface{}{}, false, fmt.Errorf("gitlab: new req: %v", err)
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return map[string]interface{}{}, false, fmt.Errorf("gitlab: get URL %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return map[string]interface{}{}, true, nil
	}
	if err = json.NewDecoder(resp.Body).Decode(&groupMembership); err != nil {
		return map[string]interface{}{}, false, fmt.Errorf("failed to decode response: %v", err)
	}
	return groupMembership, false, nil
}

func (c *gitlabConnector) getGroups(ctx context.Context, client *http.Client, groupScope bool, userLogin string, userId int) ([]string, error) {
	gitlabGroups, err := c.userGroups(ctx, client, userId)
	if err != nil {
		return nil, err
	}

	if len(c.groups) > 0 {
		filteredGroups := groups.Filter(gitlabGroups, c.groups)
		if len(filteredGroups) == 0 {
			return nil, fmt.Errorf("gitlab: user %q is not in any of the required groups", userLogin)
		}
		return filteredGroups, nil
	} else if groupScope {
		return gitlabGroups, nil
	}

	return nil, nil
}

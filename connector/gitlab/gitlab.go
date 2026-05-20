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
	// read operations of the REST API, including /api/v4/groups
	scopeReadAPI = "read_api"
	// used to retrieve groups from /oauth/userinfo
	// https://docs.gitlab.com/ee/integration/openid_connect_provider.html
	scopeOpenID = "openid"
)

const (
	//constants for inheritedGroups flag
	inheritedGroupsPerPage = 100
	accessLevelMinimalAccess = 5
	accessLevelGuest         = 10
	accessLevelPlanner       = 15
	accessLevelReporter      = 20
	accessLevelSecurityMgr   = 25
	accessLevelDeveloper     = 30
	accessLevelMaintainer    = 40
	accessLevelOwner         = 50
	accessLevelAdmin         = 60
)

// Config holds configuration options for gitlab logins.
type Config struct {
	// BaseURL is the root URL of the GitLab instance. Defaults to https://gitlab.com.
	BaseURL string `json:"baseURL"`
	// ClientID is the OAuth client ID registered in GitLab.
	ClientID string `json:"clientID"`
	// ClientSecret is the OAuth client secret registered in GitLab.
	ClientSecret string `json:"clientSecret"`
	// RedirectURI is the callback URL configured for the GitLab OAuth application.
	RedirectURI string `json:"redirectURI"`
	// Groups limits logins to users who belong to at least one of the configured GitLab groups.
	Groups []string `json:"groups"`
	// UseLoginAsID uses the GitLab username as the Dex user ID instead of the numeric GitLab user ID.
	UseLoginAsID bool `json:"useLoginAsID"`
	// GetGroupsPermission appends role-qualified entries, such as group:owner, to the groups claim.
	GetGroupsPermission bool `json:"getGroupsPermission"`
	// When enabled, Dex uses /api/v4/groups as the source of truth for group names so
	// inherited memberships are included as well. This requires GitLab's read_api scope.
	InheritedGroups bool   `json:"inheritedGroups"`
    // RootCAData is a PEM-encoded CA bundle used to trust custom TLS certificates on the GitLab instance.
	RootCAData      []byte `json:"rootCAData,omitempty"`
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
		inheritedGroups:     c.InheritedGroups,
		httpClient:          httpClient,
	}, nil
}

type connectorData struct {
	// Support GitLab's Access Tokens and Refresh tokens.
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

var (
	_ connector.CallbackConnector      = (*gitlabConnector)(nil)
	_ connector.RefreshConnector       = (*gitlabConnector)(nil)
	_ connector.TokenIdentityConnector = (*gitlabConnector)(nil)
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

	// if set to true inherited groups will be retrieved from /api/v4/groups
	inheritedGroups bool
}

func (c *gitlabConnector) oauth2Config(scopes connector.Scopes) *oauth2.Config {
	gitlabScopes := []string{scopeUser, scopeOpenID}
	if c.groupsRequired(scopes.Groups) {
		if c.inheritedGroups {
			gitlabScopes = append(gitlabScopes, scopeReadAPI)
		}
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

// TokenIdentity is used for token exchange, verifying a GitLab access token
// and returning the associated user identity. This enables direct authentication
// with Dex using an existing GitLab token without going through the OAuth flow.
//
// Note: The connector decides whether to fetch groups based on its configuration
// (groups filter, getGroupsPermission), not on the scopes from the token exchange request.
// The server will then decide whether to include groups in the final token based on
// the requested scopes. This matches the behavior of other connectors (e.g., OIDC).
func (c *gitlabConnector) TokenIdentity(ctx context.Context, _, subjectToken string) (connector.Identity, error) {
	if c.httpClient != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	}

	token := &oauth2.Token{
		AccessToken: subjectToken,
		TokenType:   "Bearer", // GitLab tokens are typically Bearer tokens even if the type is not explicitly provided.
	}

	// For token exchange, we determine if groups should be fetched based on connector configuration.
	// If the connector has groups filter or getGroupsPermission enabled, we fetch groups.
	scopes := connector.Scopes{
		// Scopes are not provided in token exchange, so we request groups every time and return only if configured.
		Groups: true,
	}

	return c.identity(ctx, scopes, token)
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

type userInfo struct {
	Groups               []string `json:"groups"`
	OwnerPermission      []string `json:"https://gitlab.org/claims/groups/owner"`
	MaintainerPermission []string `json:"https://gitlab.org/claims/groups/maintainer"`
	DeveloperPermission  []string `json:"https://gitlab.org/claims/groups/developer"`
}

type gitlabGroup struct {
	ID       int    `json:"id"`
	FullPath string `json:"full_path"`
	Path     string `json:"path"`
}

type gitlabGroupMember struct {
	AccessLevel int `json:"access_level"`
}

// userInfo queries the GitLab OIDC userinfo endpoint for profile and direct group membership.
//
// The HTTP passed client is expected to be constructed by the golang.org/x/oauth2 package,
// which inserts a bearer token as part of the request.
func (c *gitlabConnector) userInfo(ctx context.Context, client *http.Client) (userInfo, error) {
	var u userInfo
	req, err := http.NewRequest("GET", c.baseURL+"/oauth/userinfo", nil)
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

// apiUserGroups queries the GitLab groups API for all groups the current user is a member of.
// When inheritedGroups is enabled, this becomes the source of truth for group names.
func (c *gitlabConnector) apiUserGroups(ctx context.Context, client *http.Client) ([]gitlabGroup, error) {
	apiGroupsAll := make([]gitlabGroup, 0)
	for page := 1; ; page++ {
		req, err := http.NewRequest("GET", c.baseURL+"/api/v4/groups", nil)
		if err != nil {
			return nil, fmt.Errorf("gitlab: new req: %v", err)
		}

		q := req.URL.Query()
		q.Set("all_available", "false")
		q.Set("per_page", strconv.Itoa(inheritedGroupsPerPage))
		q.Set("page", strconv.Itoa(page))
		req.URL.RawQuery = q.Encode()
		req = req.WithContext(ctx)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("gitlab: get URL %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("gitlab: read body: %v", err)
			}
			return nil, fmt.Errorf("%s: %s", resp.Status, body)
		}

		var apiGroups []gitlabGroup
		if err := json.NewDecoder(resp.Body).Decode(&apiGroups); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %v", err)
		}
		_ = resp.Body.Close()

		apiGroupsAll = append(apiGroupsAll, apiGroups...)

		if len(apiGroups) < inheritedGroupsPerPage {
			break
		}
	}

	return apiGroupsAll, nil
}

// userGroups queries the GitLab APIs for group membership.
//
// The HTTP passed client is expected to be constructed by the golang.org/x/oauth2 package,
// which inserts a bearer token as part of the request.
func (c *gitlabConnector) userGroups(ctx context.Context, client *http.Client, userID int) ([]string, error) {
	if c.inheritedGroups {
		apiGroups, err := c.apiUserGroups(ctx, client)
		if err != nil {
			return nil, err
		}

		if !c.getGroupsPermission {
			return groupNames(apiGroups), nil
		}

		u, err := c.userInfo(ctx, client)
		if err != nil {
			return nil, err
		}

		return c.apiGroupsWithPermission(ctx, client, apiGroups, userID, u)
	}

	u, err := c.userInfo(ctx, client)
	if err != nil {
		return nil, err
	}

	if c.getGroupsPermission {
		return c.setGroupsPermission(u), nil
	}

	return u.Groups, nil
}

func (c *gitlabConnector) apiGroupsWithPermission(ctx context.Context, client *http.Client, apiGroups []gitlabGroup, userID int, u userInfo) ([]string, error) {
	if userID == 0 {
		return nil, errors.New("gitlab: user id is required to fetch effective group permissions")
	}

	groups := groupNames(apiGroups)
	for _, g := range apiGroups {
		groupPath := groupName(g)
		if groupPath == "" {
			continue
		}

		if permission, ok := userInfoPermission(groupPath, u); ok {
			groups = append(groups, fmt.Sprintf("%s:%s", groupPath, permission))
			continue
		}

		permission, ok, err := c.apiGroupPermission(ctx, client, g.ID, userID)
		if err != nil {
			return nil, err
		}
		if ok {
			groups = append(groups, fmt.Sprintf("%s:%s", groupPath, permission))
		}
	}

	return groups, nil
}

func (c *gitlabConnector) apiGroupPermission(ctx context.Context, client *http.Client, groupID, userID int) (string, bool, error) {
	if groupID == 0 {
		return "", false, errors.New("gitlab: group id is required to fetch effective group permissions")
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v4/groups/%d/members/all/%d", c.baseURL, groupID, userID), nil)
	if err != nil {
		return "", false, fmt.Errorf("gitlab: new req: %v", err)
	}

	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("gitlab: get URL %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", false, fmt.Errorf("gitlab: read body: %v", err)
		}
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
			if c.logger != nil {
				c.logger.Debug("gitlab: skipping effective group permission lookup", "groupID", groupID, "userID", userID, "status", resp.Status)
			}
			return "", false, nil
		}
		return "", false, fmt.Errorf("%s: %s", resp.Status, body)
	}

	var member gitlabGroupMember
	if err := json.NewDecoder(resp.Body).Decode(&member); err != nil {
		return "", false, fmt.Errorf("failed to decode response: %v", err)
	}

	permission, ok := accessLevelPermission(member.AccessLevel)
	return permission, ok, nil
}

func groupNames(apiGroups []gitlabGroup) []string {
	groups := make([]string, 0, len(apiGroups))
	for _, g := range apiGroups {
		if groupPath := groupName(g); groupPath != "" {
			groups = append(groups, groupPath)
		}
	}

	return groups
}

func groupName(g gitlabGroup) string {
	if g.FullPath != "" {
		return g.FullPath
	}
	return g.Path
}

func userInfoPermission(groupPath string, u userInfo) (string, bool) {
	if permissionMatches(groupPath, u.OwnerPermission) {
		return "owner", true
	}
	if permissionMatches(groupPath, u.MaintainerPermission) {
		return "maintainer", true
	}
	if permissionMatches(groupPath, u.DeveloperPermission) {
		return "developer", true
	}
	return "", false
}

func permissionMatches(groupPath string, permissions []string) bool {
	for _, permissionPath := range permissions {
		// Exact group match, for example "ops" matches "ops".
		if groupPath == permissionPath {
			return true
		}

		// A parent-group permission cannot match a shorter or equally long path.
		if len(groupPath) <= len(permissionPath) {
			continue
		}

		// The permission path must be a prefix of the subgroup path.
		if groupPath[0:len(permissionPath)] != permissionPath {
			continue
		}

		// Require a path separator so "dev" does not match "developer".
		if string(groupPath[len(permissionPath)]) != "/" {
			continue
		}

		// Parent-group permissions apply to descendant subgroups.
		return true
	}
	return false
}

func accessLevelPermission(accessLevel int) (string, bool) {
	switch accessLevel {
	case accessLevelMinimalAccess:
		return "minimal_access", true
	case accessLevelGuest:
		return "guest", true
	case accessLevelPlanner:
		return "planner", true
	case accessLevelReporter:
		return "reporter", true
	case accessLevelSecurityMgr:
		return "security_manager", true
	case accessLevelDeveloper:
		return "developer", true
	case accessLevelMaintainer:
		return "maintainer", true
	case accessLevelOwner:
		return "owner", true
	case accessLevelAdmin:
		return "admin", true
	default:
		return "", false
	}
}

func (c *gitlabConnector) setGroupsPermission(u userInfo) []string {
	groups := u.Groups

L1:
	for _, g := range groups {
		for _, op := range u.OwnerPermission {
			if g == op {
				groups = append(groups, fmt.Sprintf("%s:owner", g))
				continue L1
			}
			if len(g) > len(op) {
				if g[0:len(op)] == op && string(g[len(op)]) == "/" {
					groups = append(groups, fmt.Sprintf("%s:owner", g))
					continue L1
				}
			}
		}

		for _, mp := range u.MaintainerPermission {
			if g == mp {
				groups = append(groups, fmt.Sprintf("%s:maintainer", g))
				continue L1
			}
			if len(g) > len(mp) {
				if g[0:len(mp)] == mp && string(g[len(mp)]) == "/" {
					groups = append(groups, fmt.Sprintf("%s:maintainer", g))
					continue L1
				}
			}
		}

		for _, dp := range u.DeveloperPermission {
			if g == dp {
				groups = append(groups, fmt.Sprintf("%s:developer", g))
				continue L1
			}
			if len(g) > len(dp) {
				if g[0:len(dp)] == dp && string(g[len(dp)]) == "/" {
					groups = append(groups, fmt.Sprintf("%s:developer", g))
					continue L1
				}
			}
		}
	}

	return groups
}

func (c *gitlabConnector) getGroups(ctx context.Context, client *http.Client, groupScope bool, userLogin string, userID int) ([]string, error) {
	gitlabGroups, err := c.userGroups(ctx, client, userID)
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

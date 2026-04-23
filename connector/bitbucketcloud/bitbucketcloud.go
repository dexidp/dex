// Package bitbucketcloud provides authentication strategies using Bitbucket Cloud.
package bitbucketcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/bitbucket"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/groups"
)

const (
	apiURL = "https://api.bitbucket.org/2.0"
	// Bitbucket requires this scope to access '/user' API endpoints.
	scopeAccount = "account"
	// Bitbucket requires this scope to access '/user/emails' API endpoints.
	scopeEmail = "email"
)

// Config holds configuration options for Bitbucket logins.
type Config struct {
	ClientID     string   `json:"clientID"`
	ClientSecret string   `json:"clientSecret"`
	RedirectURI  string   `json:"redirectURI"`
	Teams        []string `json:"teams"`

	// Deprecated: The Bitbucket 1.0 API (/1.0/groups/{team}) that this feature
	// relied on has been removed by Atlassian. This option is ignored; if set,
	// a warning is logged at startup. Consider using getWorkspacePermissions.
	IncludeTeamGroups bool `json:"includeTeamGroups,omitempty"`

	// When enabled, appends workspace permission suffixes (e.g. "workspace:owner",
	// "workspace:member") to the groups claim, similar to GitLab's getGroupsPermission.
	GetWorkspacePermissions bool `json:"getWorkspacePermissions,omitempty"`
}

// Open returns a strategy for logging in through Bitbucket.
func (c *Config) Open(id string, logger *slog.Logger) (connector.Connector, error) {
	if c.IncludeTeamGroups {
		logger.Warn("bitbucket: includeTeamGroups is deprecated and has no effect; " +
			"the Bitbucket 1.0 API it relied on has been removed by Atlassian")
	}

	b := bitbucketConnector{
		redirectURI:             c.RedirectURI,
		teams:                   c.Teams,
		clientID:                c.ClientID,
		clientSecret:            c.ClientSecret,
		getWorkspacePermissions: c.GetWorkspacePermissions,
		apiURL:                  apiURL,
		logger:                  logger.With(slog.Group("connector", "type", "bitbucketcloud", "id", id)),
	}

	return &b, nil
}

type connectorData struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	Expiry       time.Time `json:"expiry"`
}

var (
	_ connector.CallbackConnector = (*bitbucketConnector)(nil)
	_ connector.RefreshConnector  = (*bitbucketConnector)(nil)
)

type bitbucketConnector struct {
	redirectURI             string
	teams                   []string
	clientID                string
	clientSecret            string
	logger                  *slog.Logger
	apiURL                  string
	getWorkspacePermissions bool

	// the following are used only for tests
	hostName   string
	httpClient *http.Client
}

// groupsRequired returns whether dex needs to fetch Bitbucket workspace membership.
func (b *bitbucketConnector) groupsRequired(groupScope bool) bool {
	return len(b.teams) > 0 || groupScope
}

func (b *bitbucketConnector) oauth2Config(scopes connector.Scopes) *oauth2.Config {
	bitbucketScopes := []string{scopeAccount, scopeEmail}

	endpoint := bitbucket.Endpoint
	if b.hostName != "" {
		endpoint = oauth2.Endpoint{
			AuthURL:  "https://" + b.hostName + "/site/oauth2/authorize",
			TokenURL: "https://" + b.hostName + "/site/oauth2/access_token",
		}
	}

	return &oauth2.Config{
		ClientID:     b.clientID,
		ClientSecret: b.clientSecret,
		Endpoint:     endpoint,
		Scopes:       bitbucketScopes,
	}
}

func (b *bitbucketConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, []byte, error) {
	if b.redirectURI != callbackURL {
		return "", nil, fmt.Errorf("expected callback URL %q did not match the URL in the config %q", callbackURL, b.redirectURI)
	}

	return b.oauth2Config(scopes).AuthCodeURL(state), nil, nil
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

func (b *bitbucketConnector) HandleCallback(s connector.Scopes, connData []byte, r *http.Request) (identity connector.Identity, err error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return identity, &oauth2Error{errType, q.Get("error_description")}
	}

	oauth2Config := b.oauth2Config(s)

	ctx := r.Context()
	if b.httpClient != nil {
		ctx = context.WithValue(r.Context(), oauth2.HTTPClient, b.httpClient)
	}

	token, err := oauth2Config.Exchange(ctx, q.Get("code"))
	if err != nil {
		return identity, fmt.Errorf("bitbucket: failed to get token: %v", err)
	}

	client := oauth2Config.Client(ctx, token)

	user, err := b.user(ctx, client)
	if err != nil {
		return identity, fmt.Errorf("bitbucket: get user: %v", err)
	}

	identity = connector.Identity{
		UserID:        user.UUID,
		Username:      user.Username,
		Email:         user.Email,
		EmailVerified: true,
	}

	if b.groupsRequired(s.Groups) {
		groups, err := b.getGroups(ctx, client, s.Groups, user.Username)
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
			return identity, fmt.Errorf("bitbucket: marshal connector data: %v", err)
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

func (b *bitbucketConnector) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	if len(identity.ConnectorData) == 0 {
		return identity, errors.New("bitbucket: no upstream access token found")
	}

	var data connectorData
	if err := json.Unmarshal(identity.ConnectorData, &data); err != nil {
		return identity, fmt.Errorf("bitbucket: unmarshal access token: %v", err)
	}

	tok := &oauth2.Token{
		AccessToken:  data.AccessToken,
		RefreshToken: data.RefreshToken,
		Expiry:       data.Expiry,
	}

	client := oauth2.NewClient(ctx, &notifyRefreshTokenSource{
		new: b.oauth2Config(s).TokenSource(ctx, tok),
		t:   tok,
		f: func(tok *oauth2.Token) error {
			data := connectorData{
				AccessToken:  tok.AccessToken,
				RefreshToken: tok.RefreshToken,
				Expiry:       tok.Expiry,
			}
			connData, err := json.Marshal(data)
			if err != nil {
				return fmt.Errorf("bitbucket: marshal connector data: %v", err)
			}
			identity.ConnectorData = connData
			return nil
		},
	})

	user, err := b.user(ctx, client)
	if err != nil {
		return identity, fmt.Errorf("bitbucket: get user: %v", err)
	}

	identity.Username = user.Username
	identity.Email = user.Email

	if b.groupsRequired(s.Groups) {
		groups, err := b.getGroups(ctx, client, s.Groups, user.Username)
		if err != nil {
			return identity, err
		}
		identity.Groups = groups
	}

	return identity, nil
}

// Bitbucket pagination wrapper
type pagedResponse struct {
	Size     int     `json:"size"`
	Page     int     `json:"page"`
	PageLen  int     `json:"pagelen"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
}

// user holds Bitbucket user information (relevant to dex) as defined by
// https://developer.atlassian.com/bitbucket/api/2/reference/resource/user
type user struct {
	Username string `json:"username"`
	UUID     string `json:"uuid"`
	Email    string `json:"email"`
}

// user queries the Bitbucket API for profile information using the provided client.
//
// The HTTP client is expected to be constructed by the golang.org/x/oauth2 package,
// which inserts a bearer token as part of the request.
func (b *bitbucketConnector) user(ctx context.Context, client *http.Client) (user, error) {
	// https://developer.atlassian.com/bitbucket/api/2/reference/resource/user
	var (
		u   user
		err error
	)

	if err = get(ctx, client, b.apiURL+"/user", &u); err != nil {
		return user{}, err
	}

	if u.Email, err = b.userEmail(ctx, client); err != nil {
		return user{}, err
	}

	return u, nil
}

// userEmail holds Bitbucket user email information as defined by
// https://developer.atlassian.com/bitbucket/api/2/reference/resource/user/emails
type userEmail struct {
	IsPrimary   bool   `json:"is_primary"`
	IsConfirmed bool   `json:"is_confirmed"`
	Email       string `json:"email"`
}

type userEmailResponse struct {
	pagedResponse
	Values []userEmail
}

// userEmail returns the users primary, confirmed email
//
// The HTTP client is expected to be constructed by the golang.org/x/oauth2 package,
// which inserts a bearer token as part of the request.
func (b *bitbucketConnector) userEmail(ctx context.Context, client *http.Client) (string, error) {
	apiURL := b.apiURL + "/user/emails"
	for {
		// https://developer.atlassian.com/bitbucket/api/2/reference/resource/user/emails
		var response userEmailResponse

		if err := get(ctx, client, apiURL, &response); err != nil {
			return "", err
		}

		for _, email := range response.Values {
			if email.IsConfirmed && email.IsPrimary {
				return email.Email, nil
			}
		}

		if response.Next == nil {
			break
		}
		apiURL = *response.Next
	}

	return "", errors.New("bitbucket: user has no confirmed, primary email")
}

// getGroups retrieves Bitbucket teams a user is in, if any.
func (b *bitbucketConnector) getGroups(ctx context.Context, client *http.Client, groupScope bool, userLogin string) ([]string, error) {
	bitbucketTeams, err := b.userWorkspaces(ctx, client)
	if err != nil {
		return nil, err
	}

	if len(b.teams) > 0 {
		filteredTeams := groups.Filter(bitbucketTeams, b.teams)
		if len(filteredTeams) == 0 {
			return nil, fmt.Errorf("bitbucket: user %q is not in any of the required teams", userLogin)
		}
		return filteredTeams, nil
	} else if groupScope {
		return bitbucketTeams, nil
	}

	return nil, nil
}

type workspaceRef struct {
	Slug string `json:"slug"`
}

type workspaceAccess struct {
	Workspace workspaceRef `json:"workspace"`
}

type workspacesResponse struct {
	pagedResponse
	Values []workspaceAccess `json:"values"`
}

type workspacePermission struct {
	Permission string `json:"permission"`
}

func (b *bitbucketConnector) userWorkspaces(ctx context.Context, client *http.Client) ([]string, error) {
	var teams []string
	apiURL := b.apiURL + "/user/workspaces"

	for {
		// https://developer.atlassian.com/cloud/bitbucket/rest/api-group-user/#api-user-workspaces-get
		var response workspacesResponse

		if err := get(ctx, client, apiURL, &response); err != nil {
			return nil, fmt.Errorf("bitbucket: get user workspaces: %v", err)
		}

		for _, value := range response.Values {
			teams = append(teams, value.Workspace.Slug)
		}

		if response.Next == nil {
			break
		}
		apiURL = *response.Next
	}

	if b.getWorkspacePermissions {
		var permissionGroups []string
		for _, team := range teams {
			perm, err := b.userWorkspacePermission(ctx, client, team)
			if err != nil {
				b.logger.Warn("bitbucket: failed to get permission for workspace, skipping permission suffix",
					"workspace", team, "error", err)
				continue
			}
			permissionGroups = append(permissionGroups, team+":"+perm)
		}
		teams = append(teams, permissionGroups...)
	}

	return teams, nil
}

func (b *bitbucketConnector) userWorkspacePermission(ctx context.Context, client *http.Client, workspaceSlug string) (string, error) {
	apiURL := b.apiURL + "/user/workspaces/" + workspaceSlug + "/permission"
	var response workspacePermission
	if err := get(ctx, client, apiURL, &response); err != nil {
		return "", fmt.Errorf("get workspace %q permission: %v", workspaceSlug, err)
	}
	return response.Permission, nil
}

// get creates a "GET `apiURL`" request with context, sends the request using
// the client, and decodes the resulting response body into v.
// Any errors encountered when building requests, sending requests, and
// reading and decoding response data are returned.
func get(ctx context.Context, client *http.Client, apiURL string, v interface{}) error {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("bitbucket: new req: %v", err)
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("bitbucket: get URL %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("bitbucket: read body: %s: %v", resp.Status, err)
		}
		return fmt.Errorf("%s: %s", resp.Status, body)
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("bitbucket: failed to decode response: %v", err)
	}

	return nil
}

// Package github provides authentication strategies using GitHub.
package github

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/dexidp/dex/connector"
)

const (
	apiURL = "https://api.github.com"
	// GitHub requires this scope to access '/user' and '/user/emails' API endpoints.
	scopeEmail = "user:email"
	// GitHub requires this scope to access '/user/teams' and '/orgs' API endpoints
	// which are used when a client includes the 'groups' scope.
	scopeOrgs = "read:org"
)

// Pagination URL patterns
// https://developer.github.com/v3/#pagination
var reNext = regexp.MustCompile("<([^>]+)>; rel=\"next\"")
var reLast = regexp.MustCompile("<([^>]+)>; rel=\"last\"")

// Config holds configuration options for github logins.
type Config struct {
	ClientID      string `json:"clientID"`
	ClientSecret  string `json:"clientSecret"`
	RedirectURI   string `json:"redirectURI"`
	Org           string `json:"org"`
	Orgs          []Org  `json:"orgs"`
	HostName      string `json:"hostName"`
	RootCA        string `json:"rootCA"`
	TeamNameField string `json:"teamNameField"`
	LoadAllGroups bool   `json:"loadAllGroups"`
	UseLoginAsID  bool   `json:"useLoginAsID"`
}

// Org holds org-team filters, in which teams are optional.
type Org struct {

	// Organization name in github (not slug, full name). Only users in this github
	// organization can authenticate.
	Name string `json:"name"`

	// Names of teams in a github organization. A user will be able to
	// authenticate if they are members of at least one of these teams. Users
	// in the organization can authenticate if this field is omitted from the
	// config file.
	Teams []string `json:"teams,omitempty"`
}

// Open returns a strategy for logging in through GitHub.
func (c *Config) Open(id string, logger logrus.FieldLogger) (connector.Connector, error) {

	if c.Org != "" {
		// Return error if both 'org' and 'orgs' fields are used.
		if len(c.Orgs) > 0 {
			return nil, errors.New("github: cannot use both 'org' and 'orgs' fields simultaneously")
		}
		logger.Warnln("github: legacy field 'org' being used. Switch to the newer 'orgs' field structure")
	}

	g := githubConnector{
		redirectURI:  c.RedirectURI,
		org:          c.Org,
		orgs:         c.Orgs,
		clientID:     c.ClientID,
		clientSecret: c.ClientSecret,
		apiURL:       apiURL,
		logger:       logger,
		useLoginAsID: c.UseLoginAsID,
	}

	if c.HostName != "" {
		// ensure this is a hostname and not a URL or path.
		if strings.Contains(c.HostName, "/") {
			return nil, errors.New("invalid hostname: hostname cannot contain `/`")
		}

		g.hostName = c.HostName
		g.apiURL = "https://" + c.HostName + "/api/v3"
	}

	if c.RootCA != "" {
		if c.HostName == "" {
			return nil, errors.New("invalid connector config: Host name field required for a root certificate file")
		}
		g.rootCA = c.RootCA

		var err error
		if g.httpClient, err = newHTTPClient(g.rootCA); err != nil {
			return nil, fmt.Errorf("failed to create HTTP client: %v", err)
		}

	}
	g.loadAllGroups = c.LoadAllGroups

	switch c.TeamNameField {
	case "name", "slug", "both", "":
		g.teamNameField = c.TeamNameField
	default:
		return nil, fmt.Errorf("invalid connector config: unsupported team name field value `%s`", c.TeamNameField)
	}

	return &g, nil
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
	orgs         []Org
	clientID     string
	clientSecret string
	logger       logrus.FieldLogger
	// apiURL defaults to "https://api.github.com"
	apiURL string
	// hostName of the GitHub enterprise account.
	hostName string
	// Used to support untrusted/self-signed CA certs.
	rootCA string
	// HTTP Client that trusts the custom delcared rootCA cert.
	httpClient *http.Client
	// optional choice between 'name' (default) or 'slug'
	teamNameField string
	// if set to true and no orgs are configured then connector loads all user claims (all orgs and team)
	loadAllGroups bool
	// if set to true will use the users handle rather than their numeric id as the ID
	useLoginAsID bool
}

// groupsRequired returns whether dex requires GitHub's 'read:org' scope. Dex
// needs 'read:org' if 'orgs' or 'org' fields are populated in a config file.
// Clients can require 'groups' scope without setting 'orgs'/'org'.
func (c *githubConnector) groupsRequired(groupScope bool) bool {
	return len(c.orgs) > 0 || c.org != "" || groupScope
}

func (c *githubConnector) oauth2Config(scopes connector.Scopes) *oauth2.Config {
	// 'read:org' scope is required by the GitHub API, and thus for dex to ensure
	// a user is a member of orgs and teams provided in configs.
	githubScopes := []string{scopeEmail}
	if c.groupsRequired(scopes.Groups) {
		githubScopes = append(githubScopes, scopeOrgs)
	}

	endpoint := github.Endpoint
	// case when it is a GitHub Enterprise account.
	if c.hostName != "" {
		endpoint = oauth2.Endpoint{
			AuthURL:  "https://" + c.hostName + "/login/oauth/authorize",
			TokenURL: "https://" + c.hostName + "/login/oauth/access_token",
		}
	}

	return &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     endpoint,
		Scopes:       githubScopes,
	}
}

func (c *githubConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, error) {
	if c.redirectURI != callbackURL {
		return "", fmt.Errorf("expected callback URL %q did not match the URL in the config %q", callbackURL, c.redirectURI)
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

// newHTTPClient returns a new HTTP client that trusts the custom delcared rootCA cert.
func newHTTPClient(rootCA string) (*http.Client, error) {
	tlsConfig := tls.Config{RootCAs: x509.NewCertPool()}
	rootCABytes, err := ioutil.ReadFile(rootCA)
	if err != nil {
		return nil, fmt.Errorf("failed to read root-ca: %v", err)
	}
	if !tlsConfig.RootCAs.AppendCertsFromPEM(rootCABytes) {
		return nil, fmt.Errorf("no certs found in root CA file %q", rootCA)
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tlsConfig,
			Proxy:           http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}, nil
}

func (c *githubConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return identity, &oauth2Error{errType, q.Get("error_description")}
	}

	oauth2Config := c.oauth2Config(s)

	ctx := r.Context()
	// GitHub Enterprise account
	if c.httpClient != nil {
		ctx = context.WithValue(r.Context(), oauth2.HTTPClient, c.httpClient)
	}

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
	if c.useLoginAsID {
		identity.UserID = user.Login
	}

	// Only set identity.Groups if 'orgs', 'org', or 'groups' scope are specified.
	if c.groupsRequired(s.Groups) {
		groups, err := c.getGroups(ctx, client, s.Groups, user.Login)
		if err != nil {
			return identity, err
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

func (c *githubConnector) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	if len(identity.ConnectorData) == 0 {
		return identity, errors.New("no upstream access token found")
	}

	var data connectorData
	if err := json.Unmarshal(identity.ConnectorData, &data); err != nil {
		return identity, fmt.Errorf("github: unmarshal access token: %v", err)
	}

	client := c.oauth2Config(s).Client(ctx, &oauth2.Token{AccessToken: data.AccessToken})
	user, err := c.user(ctx, client)
	if err != nil {
		return identity, fmt.Errorf("github: get user: %v", err)
	}

	username := user.Name
	if username == "" {
		username = user.Login
	}
	identity.Username = username
	identity.Email = user.Email

	// Only set identity.Groups if 'orgs', 'org', or 'groups' scope are specified.
	if c.groupsRequired(s.Groups) {
		groups, err := c.getGroups(ctx, client, s.Groups, user.Login)
		if err != nil {
			return identity, err
		}
		identity.Groups = groups
	}

	return identity, nil
}

// getGroups retrieves GitHub orgs and teams a user is in, if any.
func (c *githubConnector) getGroups(ctx context.Context, client *http.Client, groupScope bool, userLogin string) ([]string, error) {
	if len(c.orgs) > 0 {
		return c.groupsForOrgs(ctx, client, userLogin)
	} else if c.org != "" {
		return c.teamsForOrg(ctx, client, c.org)
	} else if groupScope && c.loadAllGroups {
		return c.userGroups(ctx, client)
	}
	return nil, nil
}

// formatTeamName returns unique team name.
// Orgs might have the same team names. To make team name unique it should be prefixed with the org name.
func formatTeamName(org string, team string) string {
	return fmt.Sprintf("%s:%s", org, team)
}

// groupsForOrgs enforces org and team constraints on user authorization
// Cases in which user is authorized:
// 	N orgs, no teams: user is member of at least 1 org
// 	N orgs, M teams per org: user is member of any team from at least 1 org
// 	N-1 orgs, M teams per org, 1 org with no teams: user is member of any team
// from at least 1 org, or member of org with no teams
func (c *githubConnector) groupsForOrgs(ctx context.Context, client *http.Client, userName string) ([]string, error) {
	groups := make([]string, 0)
	var inOrgNoTeams bool
	for _, org := range c.orgs {
		inOrg, err := c.userInOrg(ctx, client, userName, org.Name)
		if err != nil {
			return nil, err
		}
		if !inOrg {
			continue
		}

		teams, err := c.teamsForOrg(ctx, client, org.Name)
		if err != nil {
			return nil, err
		}
		// User is in at least one org. User is authorized if no teams are specified
		// in config; include all teams in claim. Otherwise filter out teams not in
		// 'teams' list in config.
		if len(org.Teams) == 0 {
			inOrgNoTeams = true
		} else if teams = filterTeams(teams, org.Teams); len(teams) == 0 {
			c.logger.Infof("github: user %q in org %q but no teams", userName, org.Name)
		}

		for _, teamName := range teams {
			groups = append(groups, formatTeamName(org.Name, teamName))
		}
	}
	if inOrgNoTeams || len(groups) > 0 {
		return groups, nil
	}
	return groups, fmt.Errorf("github: user %q not in required orgs or teams", userName)
}

func (c *githubConnector) userGroups(ctx context.Context, client *http.Client) ([]string, error) {
	orgs, err := c.userOrgs(ctx, client)
	if err != nil {
		return nil, err
	}

	orgTeams, err := c.userOrgTeams(ctx, client)
	if err != nil {
		return nil, err
	}

	groups := make([]string, 0)
	for _, o := range orgs {
		groups = append(groups, o)
		if teams, ok := orgTeams[o]; ok {
			for _, t := range teams {
				groups = append(groups, formatTeamName(o, t))
			}
		}
	}

	return groups, nil
}

// userOrgs retrieves list of current user orgs
func (c *githubConnector) userOrgs(ctx context.Context, client *http.Client) ([]string, error) {
	groups := make([]string, 0)
	apiURL := c.apiURL + "/user/orgs"
	for {
		// https://developer.github.com/v3/orgs/#list-your-organizations
		var (
			orgs []org
			err  error
		)
		if apiURL, err = get(ctx, client, apiURL, &orgs); err != nil {
			return nil, fmt.Errorf("github: get orgs: %v", err)
		}

		for _, o := range orgs {
			groups = append(groups, o.Login)
		}

		if apiURL == "" {
			break
		}
	}

	return groups, nil
}

// userOrgTeams retrieves teams which current user belongs to.
// Method returns a map where key is an org name and value list of teams under the org.
func (c *githubConnector) userOrgTeams(ctx context.Context, client *http.Client) (map[string][]string, error) {
	groups := make(map[string][]string, 0)
	apiURL := c.apiURL + "/user/teams"
	for {
		// https://developer.github.com/v3/orgs/teams/#list-user-teams
		var (
			teams []team
			err   error
		)
		if apiURL, err = get(ctx, client, apiURL, &teams); err != nil {
			return nil, fmt.Errorf("github: get teams: %v", err)
		}

		for _, t := range teams {
			groups[t.Org.Login] = append(groups[t.Org.Login], c.teamGroupClaims(t)...)
		}

		if apiURL == "" {
			break
		}
	}

	return groups, nil
}

// Filter the users' team memberships by 'teams' from config.
func filterTeams(userTeams, configTeams []string) (teams []string) {
	teamFilter := make(map[string]struct{})
	for _, team := range configTeams {
		if _, ok := teamFilter[team]; !ok {
			teamFilter[team] = struct{}{}
		}
	}
	for _, team := range userTeams {
		if _, ok := teamFilter[team]; ok {
			teams = append(teams, team)
		}
	}
	return
}

// get creates a "GET `apiURL`" request with context, sends the request using
// the client, and decodes the resulting response body into v. A pagination URL
// is returned if one exists. Any errors encountered when building requests,
// sending requests, and reading and decoding response data are returned.
func get(ctx context.Context, client *http.Client, apiURL string, v interface{}) (string, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("github: new req: %v", err)
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("github: get URL %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("github: read body: %v", err)
		}
		return "", fmt.Errorf("%s: %s", resp.Status, body)
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	return getPagination(apiURL, resp), nil
}

// getPagination checks the "Link" header field for "next" or "last" pagination URLs,
// and returns "next" page URL or empty string to indicate that there are no more pages.
// Non empty next pages' URL is returned if both "last" and "next" URLs are found and next page
// URL is not equal to last.
//
// https://developer.github.com/v3/#pagination
func getPagination(apiURL string, resp *http.Response) string {
	if resp == nil {
		return ""
	}

	links := resp.Header.Get("Link")
	if len(reLast.FindStringSubmatch(links)) > 1 {
		lastPageURL := reLast.FindStringSubmatch(links)[1]
		if apiURL == lastPageURL {
			return ""
		}
	} else {
		return ""
	}

	if len(reNext.FindStringSubmatch(links)) > 1 {
		return reNext.FindStringSubmatch(links)[1]
	}

	return ""
}

// user holds GitHub user information (relevant to dex) as defined by
// https://developer.github.com/v3/users/#response-with-public-profile-information
type user struct {
	Name  string `json:"name"`
	Login string `json:"login"`
	ID    int    `json:"id"`
	Email string `json:"email"`
}

// user queries the GitHub API for profile information using the provided client.
//
// The HTTP client is expected to be constructed by the golang.org/x/oauth2 package,
// which inserts a bearer token as part of the request.
func (c *githubConnector) user(ctx context.Context, client *http.Client) (user, error) {
	// https://developer.github.com/v3/users/#get-the-authenticated-user
	var u user
	if _, err := get(ctx, client, c.apiURL+"/user", &u); err != nil {
		return u, err
	}

	// Only public user emails are returned by 'GET /user'. u.Email will be empty
	// if a users' email is private. We must retrieve private emails explicitly.
	if u.Email == "" {
		var err error
		if u.Email, err = c.userEmail(ctx, client); err != nil {
			return u, err
		}
	}
	return u, nil
}

// userEmail holds GitHub user email information as defined by
// https://developer.github.com/v3/users/emails/#response
type userEmail struct {
	Email      string `json:"email"`
	Verified   bool   `json:"verified"`
	Primary    bool   `json:"primary"`
	Visibility string `json:"visibility"`
}

// userEmail queries the GitHub API for a users' email information using the
// provided client. Only returns the users' verified, primary email (private or
// public).
//
// The HTTP client is expected to be constructed by the golang.org/x/oauth2 package,
// which inserts a bearer token as part of the request.
func (c *githubConnector) userEmail(ctx context.Context, client *http.Client) (string, error) {
	apiURL := c.apiURL + "/user/emails"
	for {
		// https://developer.github.com/v3/users/emails/#list-email-addresses-for-a-user
		var (
			emails []userEmail
			err    error
		)
		if apiURL, err = get(ctx, client, apiURL, &emails); err != nil {
			return "", err
		}

		for _, email := range emails {
			/*
				if GitHub Enterprise, set email.Verified to true
				This change being made because GitHub Enterprise does not
				support email verification. CircleCI indicated that GitHub
				advised them not to check for verified emails
				(https://circleci.com/enterprise/changelog/#1-47-1).
				In addition, GitHub Enterprise support replied to a support
				ticket with "There is no way to verify an email address in
				GitHub Enterprise."
			*/
			if c.hostName != "" {
				email.Verified = true
			}

			if email.Verified && email.Primary {
				return email.Email, nil
			}
		}

		if apiURL == "" {
			break
		}
	}

	return "", errors.New("github: user has no verified, primary email")
}

// userInOrg queries the GitHub API for a users' org membership.
//
// The HTTP passed client is expected to be constructed by the golang.org/x/oauth2 package,
// which inserts a bearer token as part of the request.
func (c *githubConnector) userInOrg(ctx context.Context, client *http.Client, userName, orgName string) (bool, error) {
	// requester == user, so GET-ing this endpoint should return 404/302 if user
	// is not a member
	//
	// https://developer.github.com/v3/orgs/members/#check-membership
	apiURL := fmt.Sprintf("%s/orgs/%s/members/%s", c.apiURL, orgName, userName)

	req, err := http.NewRequest("GET", apiURL, nil)

	if err != nil {
		return false, fmt.Errorf("github: new req: %v", err)
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("github: get teams: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
	case http.StatusFound, http.StatusNotFound:
		c.logger.Infof("github: user %q not in org %q or application not authorized to read org data", userName, orgName)
	default:
		err = fmt.Errorf("github: unexpected return status: %q", resp.Status)
	}

	// 204 if user is a member
	return resp.StatusCode == http.StatusNoContent, err
}

// teams holds GitHub a users' team information as defined by
// https://developer.github.com/v3/orgs/teams/#response-12
type team struct {
	Name string `json:"name"`
	Org  org    `json:"organization"`
	Slug string `json:"slug"`
}

type org struct {
	Login string `json:"login"`
}

// teamsForOrg queries the GitHub API for team membership within a specific organization.
//
// The HTTP passed client is expected to be constructed by the golang.org/x/oauth2 package,
// which inserts a bearer token as part of the request.
func (c *githubConnector) teamsForOrg(ctx context.Context, client *http.Client, orgName string) ([]string, error) {
	apiURL, groups := c.apiURL+"/user/teams", []string{}
	for {
		// https://developer.github.com/v3/orgs/teams/#list-user-teams
		var (
			teams []team
			err   error
		)
		if apiURL, err = get(ctx, client, apiURL, &teams); err != nil {
			return nil, fmt.Errorf("github: get teams: %v", err)
		}

		for _, t := range teams {
			if t.Org.Login == orgName {
				groups = append(groups, c.teamGroupClaims(t)...)
			}
		}

		if apiURL == "" {
			break
		}
	}

	return groups, nil
}

// teamGroupClaims returns team slug if 'teamNameField' option is set to
// 'slug', returns the slug *and* name if set to 'both', otherwise returns team
// name.
func (c *githubConnector) teamGroupClaims(t team) []string {
	switch c.teamNameField {
	case "both":
		return []string{t.Name, t.Slug}
	case "slug":
		return []string{t.Slug}
	default:
		return []string{t.Name}
	}
}

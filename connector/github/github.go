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

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/coreos/dex/connector"
	"github.com/sirupsen/logrus"
)

const (
	apiURL     = "https://api.github.com"
	scopeEmail = "user:email"
	scopeOrgs  = "read:org"
)

// Config holds configuration options for github logins.
type Config struct {
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	RedirectURI  string `json:"redirectURI"`
	Org          string `json:"org"`
	Orgs         []Org  `json:"orgs"`
	HostName     string `json:"hostName"`
	RootCA       string `json:"rootCA"`
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
func (c *Config) Open(logger logrus.FieldLogger) (connector.Connector, error) {

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
}

func (c *githubConnector) oauth2Config(scopes connector.Scopes) *oauth2.Config {
	var githubScopes []string
	if scopes.Groups {
		githubScopes = []string{scopeEmail, scopeOrgs}
	} else {
		githubScopes = []string{scopeEmail}
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

	if s.Groups {
		var groups []string
		if len(c.orgs) > 0 {
			if groups, err = c.listGroups(ctx, client, username); err != nil {
				return identity, err
			}
		} else if c.org != "" {
			inOrg, err := c.userInOrg(ctx, client, username, c.org)
			if err != nil {
				return identity, err
			}
			if !inOrg {
				return identity, fmt.Errorf("github: user %q not a member of org %q", username, c.org)
			}
			if groups, err = c.teams(ctx, client, c.org); err != nil {
				return identity, fmt.Errorf("github: get teams: %v", err)
			}
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

	if s.Groups {
		var groups []string
		if len(c.orgs) > 0 {
			if groups, err = c.listGroups(ctx, client, username); err != nil {
				return identity, err
			}
		} else if c.org != "" {
			inOrg, err := c.userInOrg(ctx, client, username, c.org)
			if err != nil {
				return identity, err
			}
			if !inOrg {
				return identity, fmt.Errorf("github: user %q not a member of org %q", username, c.org)
			}
			if groups, err = c.teams(ctx, client, c.org); err != nil {
				return identity, fmt.Errorf("github: get teams: %v", err)
			}
		}
		identity.Groups = groups
	}

	return identity, nil
}

// listGroups enforces org and team constraints on user authorization
// Cases in which user is authorized:
// 	N orgs, no teams: user is member of at least 1 org
// 	N orgs, M teams per org: user is member of any team from at least 1 org
// 	N-1 orgs, M teams per org, 1 org with no teams: user is member of any team
// from at least 1 org, or member of org with no teams
func (c *githubConnector) listGroups(ctx context.Context, client *http.Client, userName string) (groups []string, err error) {
	var inOrgNoTeams bool
	for _, org := range c.orgs {
		inOrg, err := c.userInOrg(ctx, client, userName, org.Name)
		if err != nil {
			return groups, err
		}
		if !inOrg {
			continue
		}

		teams, err := c.teams(ctx, client, org.Name)
		if err != nil {
			return groups, err
		}
		// User is in at least one org. User is authorized if no teams are specified
		// in config; include all teams in claim. Otherwise filter out teams not in
		// 'teams' list in config.
		if len(org.Teams) == 0 {
			inOrgNoTeams = true
			c.logger.Debugf("github: user %q in org %q", userName, org.Name)
		} else if teams = filterTeams(teams, org.Teams); len(teams) == 0 {
			c.logger.Debugf("github: user %q in org %q but no teams", userName, org.Name)
		}

		// Orgs might have the same team names. We append orgPrefix to team name,
		// i.e. "org:team", to make team names unique across orgs.
		orgPrefix := org.Name + ":"
		for _, teamName := range teams {
			groups = append(groups, orgPrefix+teamName)
			c.logger.Debugf("github: user %q in org %q team %q", userName, org.Name, teamName)
		}
	}
	if inOrgNoTeams || len(groups) > 0 {
		return
	}
	return groups, fmt.Errorf("github: user %q not in required orgs or teams", userName)
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
	req, err := http.NewRequest("GET", c.apiURL+"/user", nil)
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
		c.logger.Debugf("github: user %q not in org %q", userName, orgName)
	default:
		err = fmt.Errorf("github: unexpected return status: %q", resp.Status)
	}

	// 204 if user is a member
	return resp.StatusCode == http.StatusNoContent, err
}

// teams queries the GitHub API for team membership within a specific organization.
//
// The HTTP passed client is expected to be constructed by the golang.org/x/oauth2 package,
// which inserts a bearer token as part of the request.
func (c *githubConnector) teams(ctx context.Context, client *http.Client, orgName string) ([]string, error) {

	groups := []string{}

	// https://developer.github.com/v3/#pagination
	reNext := regexp.MustCompile("<(.*)>; rel=\"next\"")
	reLast := regexp.MustCompile("<(.*)>; rel=\"last\"")
	apiURL := c.apiURL + "/user/teams"

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
			if team.Org.Login == orgName {
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

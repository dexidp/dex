// Package gitlab provides authentication strategies using Gitlab.
package gitlab

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

	"github.com/coreos/dex/connector"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const (
	hostName   = "gitlab.com"
	scopeEmail = "user:email"
	scopeOrgs  = "read:org"
)

// Config holds configuration options for GitLab logins.
type Config struct {
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	RedirectURI  string `json:"redirectURI"`
	HostName     string `json:"hostName"`
	RootCA       string `json:"rootCA"`
}

type gitlabUser struct {
	ID       int
	Name     string
	Username string
	State    string
	Email    string
	IsAdmin  bool
}

type gitlabGroup struct {
	ID   int
	Name string
	Path string
}

// Open returns a strategy for logging in through GitLab.
func (c *Config) Open(logger logrus.FieldLogger) (connector.Connector, error) {
	g := &gitlabConnector{
		redirectURI:  c.RedirectURI,
		clientID:     c.ClientID,
		clientSecret: c.ClientSecret,
		hostName:     hostName,
		apiURL:       "https://" + hostName + "/api/v4",
		logger:       logger,
	}

	if c.HostName != "" {
		// ensure this is a hostname and not a URL or path.
		if strings.Contains(c.HostName, "/") {
			return nil, errors.New("invalid hostname: hostname cannot contain `/`")
		}

		g.hostName = c.HostName
		g.apiURL = "https://" + c.HostName + "/api/v4"
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

type connectorData struct {
	// GitLab's OAuth2 tokens never expire. We don't need a refresh token.
	AccessToken string `json:"accessToken"`
}

var (
	_ connector.CallbackConnector = (*gitlabConnector)(nil)
	_ connector.RefreshConnector  = (*gitlabConnector)(nil)
)

type gitlabConnector struct {
	redirectURI  string
	org          string
	clientID     string
	clientSecret string
	logger       logrus.FieldLogger
	// apiURL defaults to "https://www.gitlab.com/api/v4".
	apiURL string
	// host name of the Gitlab enterprise account.
	hostName string
	// Used to support untrusted/self-signed CA certs.
	rootCA string
	// HTTP Client that trusts the custom delcared rootCA cert.
	httpClient *http.Client
}

func (c *gitlabConnector) oauth2Config(scopes connector.Scopes) *oauth2.Config {
	gitlabScopes := []string{"api"}

	gitlabEndpoint := oauth2.Endpoint{
		AuthURL:  "https://" + c.hostName + "/oauth/authorize",
		TokenURL: "https://" + c.hostName + "/oauth/token",
	}

	return &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     gitlabEndpoint,
		Scopes:       gitlabScopes,
		RedirectURL:  c.redirectURI,
	}
}

func (c *gitlabConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, error) {
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

func (c *gitlabConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return identity, &oauth2Error{errType, q.Get("error_description")}
	}

	oauth2Config := c.oauth2Config(s)
	ctx := r.Context()

	token, err := oauth2Config.Exchange(ctx, q.Get("code"))
	if err != nil {
		return identity, fmt.Errorf("gitlab: failed to get token: %v", err)
	}

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
		UserID:        strconv.Itoa(user.ID),
		Username:      username,
		Email:         user.Email,
		EmailVerified: true,
	}

	if s.Groups {
		groups, err := c.groups(ctx, client)
		if err != nil {
			return identity, fmt.Errorf("gitlab: get groups: %v", err)
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

func (c *gitlabConnector) Refresh(ctx context.Context, s connector.Scopes, ident connector.Identity) (connector.Identity, error) {
	if len(ident.ConnectorData) == 0 {
		return ident, errors.New("no upstream access token found")
	}

	var data connectorData
	if err := json.Unmarshal(ident.ConnectorData, &data); err != nil {
		return ident, fmt.Errorf("gitlab: unmarshal access token: %v", err)
	}

	client := c.oauth2Config(s).Client(ctx, &oauth2.Token{AccessToken: data.AccessToken})
	user, err := c.user(ctx, client)
	if err != nil {
		return ident, fmt.Errorf("gitlab: get user: %v", err)
	}

	username := user.Name
	if username == "" {
		username = user.Email
	}
	ident.Username = username
	ident.Email = user.Email

	if s.Groups {
		groups, err := c.groups(ctx, client)
		if err != nil {
			return ident, fmt.Errorf("gitlab: get groups: %v", err)
		}
		ident.Groups = groups
	}
	return ident, nil
}

// user queries the GitLab API for profile information using the provided client. The HTTP
// client is expected to be constructed by the golang.org/x/oauth2 package, which inserts
// a bearer token as part of the request.
func (c *gitlabConnector) user(ctx context.Context, client *http.Client) (gitlabUser, error) {
	var u gitlabUser
	req, err := http.NewRequest("GET", c.apiURL+"/user", nil)
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
		body, err := ioutil.ReadAll(resp.Body)
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

// groups queries the GitLab API for group membership.
//
// The HTTP passed client is expected to be constructed by the golang.org/x/oauth2 package,
// which inserts a bearer token as part of the request.
func (c *gitlabConnector) groups(ctx context.Context, client *http.Client) ([]string, error) {

	apiURL := c.apiURL + "/groups"

	reNext := regexp.MustCompile("<(.*)>; rel=\"next\"")
	reLast := regexp.MustCompile("<(.*)>; rel=\"last\"")

	groups := []string{}
	var gitlabGroups []gitlabGroup
	for {
		// 100 is the maximum number for per_page that allowed by gitlab
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("gitlab: new req: %v", err)
		}
		req = req.WithContext(ctx)
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("gitlab: get groups: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("gitlab: read body: %v", err)
			}
			return nil, fmt.Errorf("%s: %s", resp.Status, body)
		}

		if err := json.NewDecoder(resp.Body).Decode(&gitlabGroups); err != nil {
			return nil, fmt.Errorf("gitlab: unmarshal groups: %v", err)
		}

		for _, group := range gitlabGroups {
			groups = append(groups, group.Name)
		}

		link := resp.Header.Get("Link")

		if len(reLast.FindStringSubmatch(link)) > 1 {
			lastPageURL := reLast.FindStringSubmatch(link)[1]

			if apiURL == lastPageURL {
				break
			}
		} else {
			break
		}

		if len(reNext.FindStringSubmatch(link)) > 1 {
			apiURL = reNext.FindStringSubmatch(link)[1]
		} else {
			break
		}

	}
	return groups, nil
}

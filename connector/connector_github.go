package connector

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"strconv"

	chttp "github.com/coreos/go-oidc/http"
	"github.com/coreos/go-oidc/oauth2"
	"github.com/coreos/go-oidc/oidc"
)

const (
	GitHubConnectorType = "github"
	githubAuthURL       = "https://github.com/login/oauth/authorize"
	githubTokenURL      = "https://github.com/login/oauth/access_token"
	githubAPIUserURL    = "https://api.github.com/user"
)

func init() {
	RegisterConnectorConfigType(GitHubConnectorType, func() ConnectorConfig { return &GitHubConnectorConfig{} })
}

type GitHubConnectorConfig struct {
	ID           string `json:"id"`
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
}

func (cfg *GitHubConnectorConfig) ConnectorID() string {
	return cfg.ID
}

func (cfg *GitHubConnectorConfig) ConnectorType() string {
	return GitHubConnectorType
}

func (cfg *GitHubConnectorConfig) Connector(ns url.URL, lf oidc.LoginFunc, _ NewSessionFunc, tpls *template.Template) (Connector, error) {
	ns.Path = path.Join(ns.Path, httpPathCallback)
	oauth2Conn, err := newGitHubConnector(cfg.ClientID, cfg.ClientSecret, ns.String())
	if err != nil {
		return nil, err
	}
	return &OAuth2Connector{
		id:        cfg.ID,
		loginFunc: lf,
		cbURL:     ns,
		conn:      oauth2Conn,
	}, nil
}

type githubOAuth2Connector struct {
	clientID     string
	clientSecret string
	client       *oauth2.Client
}

func newGitHubConnector(clientID, clientSecret, cbURL string) (oauth2Connector, error) {
	config := oauth2.Config{
		Credentials: oauth2.ClientCredentials{ID: clientID, Secret: clientSecret},
		AuthURL:     githubAuthURL,
		TokenURL:    githubTokenURL,
		Scope:       []string{"user:email"},
		AuthMethod:  oauth2.AuthMethodClientSecretPost,
		RedirectURL: cbURL,
	}

	cli, err := oauth2.NewClient(http.DefaultClient, config)
	if err != nil {
		return nil, err
	}

	return &githubOAuth2Connector{
		clientID:     clientID,
		clientSecret: clientSecret,
		client:       cli,
	}, nil
}

// standard error form returned by github
type githubError struct {
	Message string `json:"message"`
}

func (err githubError) Error() string {
	return fmt.Sprintf("github: %s", err.Message)
}

func (c *githubOAuth2Connector) Client() *oauth2.Client {
	return c.client
}

func (c *githubOAuth2Connector) Identity(cli chttp.Client) (oidc.Identity, error) {
	req, err := http.NewRequest("GET", githubAPIUserURL, nil)
	if err != nil {
		return oidc.Identity{}, err
	}
	resp, err := cli.Do(req)
	if err != nil {
		return oidc.Identity{}, fmt.Errorf("get: %v", err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode >= 400 && resp.StatusCode < 600:
		// attempt to decode error from github
		var authErr githubError
		if err := json.NewDecoder(resp.Body).Decode(&authErr); err != nil {
			return oidc.Identity{}, oauth2.NewError(oauth2.ErrorAccessDenied)
		}
		return oidc.Identity{}, authErr
	case resp.StatusCode == http.StatusOK:
	default:
		return oidc.Identity{}, fmt.Errorf("unexpected status from providor %s", resp.Status)
	}
	var user struct {
		Login string `json:"login"`
		ID    int64  `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return oidc.Identity{}, fmt.Errorf("getting user info: %v", err)
	}
	name := user.Name
	if name == "" {
		name = user.Login
	}
	return oidc.Identity{
		ID:    strconv.FormatInt(user.ID, 10),
		Name:  name,
		Email: user.Email,
	}, nil
}

func (c *githubOAuth2Connector) Healthy() error {
	return nil
}

func (c *githubOAuth2Connector) TrustedEmailProvider() bool {
	return false
}

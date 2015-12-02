package connector

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/go-oidc/oauth2"
	"github.com/coreos/go-oidc/oidc"
)

const (
	GitHubConnectorType  = "github"
	gitHubAuthorizeURL   = "https://github.com/login/oauth/authorize"
	gitHubAccessTokenURL = "https://github.com/login/oauth/access_token"
	gitHubAPIUserURL     = "https://api.github.com/user"
)

func init() {
	RegisterConnectorConfigType(GitHubConnectorType, func() ConnectorConfig { return &GitHubConnectorConfig{} })
}

type GitHubConnectorConfig struct {
	ID           string   `json:"id"`
	ClientID     string   `json:"clientID"`
	ClientSecret string   `json:"clientSecret"`
	Scopes       []string `json:"scopes"`
}

func (cfg *GitHubConnectorConfig) ConnectorID() string {
	return cfg.ID
}

func (cfg *GitHubConnectorConfig) ConnectorType() string {
	return GitHubConnectorType
}

type GitHubConnector struct {
	id           string
	cbURL        url.URL
	loginFunc    oidc.LoginFunc
	clientID     string
	clientSecret string
	scopes       string // comma separated list of scopes
}

func (cfg *GitHubConnectorConfig) Connector(ns url.URL, lf oidc.LoginFunc, tpls *template.Template) (Connector, error) {
	ns.Path = path.Join(ns.Path, httpPathCallback)
	return &GitHubConnector{
		id:           cfg.ID,
		cbURL:        ns,
		loginFunc:    lf,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		scopes:       strings.Join(cfg.Scopes, ","),
	}, nil
}

func (c *GitHubConnector) ID() string {
	return c.id
}

func (c *GitHubConnector) Healthy() error {
	return nil
}

func (c *GitHubConnector) Sync() chan struct{} {
	stop := make(chan struct{}, 1)
	return stop
}

func (c *GitHubConnector) LoginURL(sessionKey, prompt string) (string, error) {
	v := url.Values{}
	v.Add("client_id", c.clientID)
	v.Add("redirect_uri", c.cbURL.String())
	v.Add("scope", c.scopes)
	v.Add("state", sessionKey)

	return gitHubAuthorizeURL + "?" + v.Encode(), nil
}

func (c *GitHubConnector) TrustedEmailProvider() bool {
	return true
}

func (c *GitHubConnector) Register(mux *http.ServeMux, errorURL url.URL) {
	log.Errorf("path %s", c.cbURL.Path)
	mux.Handle(c.cbURL.Path, c.handleCallbackFunc(c.loginFunc, errorURL))
}

func (c *GitHubConnector) handleCallbackFunc(lf oidc.LoginFunc, errorURL url.URL) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		e := q.Get("error")
		if e != "" {
			redirectError(w, errorURL, q)
			return
		}

		code := q.Get("code")
		if code == "" {
			q.Set("error", oauth2.ErrorInvalidRequest)
			q.Set("error_description", "code query param must be set")
			redirectError(w, errorURL, q)
			return
		}

		sessionKey := q.Get("state")
		token, err := c.exchangeAuthCode(code, sessionKey)
		if err != nil {
			log.Errorf("Unable to verify auth code with issuer: %v", err)
			q.Set("error", oauth2.ErrorUnsupportedResponseType)
			q.Set("error_description", "unable to verify auth code with issuer")
			redirectError(w, errorURL, q)
			return
		}
		user, err := getGitHubUser(token)
		if err != nil {
			log.Errorf("Unable to query GitHub API with token: %v", err)
			q.Set("error", oauth2.ErrorUnsupportedResponseType)
			// TODO(eric): better error messages
			q.Set("error_description", "unable to convert claims to identity")
			redirectError(w, errorURL, q)
			return
		}

		ident := oidc.Identity{
			ID:    strconv.FormatInt(user.ID, 10),
			Name:  user.Login,
			Email: user.Email,
		}
		redirectURL, err := lf(ident, sessionKey)
		if err != nil {
			log.Errorf("Unable to log in %#v: %v", ident, err)
			q.Set("error", oauth2.ErrorAccessDenied)
			q.Set("error_description", "login failed")
			redirectError(w, errorURL, q)
			return
		}
		w.Header().Set("Location", redirectURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	}
}

func (c *GitHubConnector) exchangeAuthCode(code, state string) (string, error) {
	v := url.Values{}
	v.Add("client_id", c.clientID)
	v.Add("client_secret", c.clientSecret)
	v.Add("code", code)
	v.Add("redirect_uri", c.cbURL.String())
	if state != "" {
		v.Add("state", state)
	}

	u := gitHubAccessTokenURL + "?" + v.Encode()

	resp, err := http.Post(u, "", nil)
	if err != nil {
		return "", fmt.Errorf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status from GitHub %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %v", err)
	}

	q, err := url.ParseQuery(string(body))
	if err != nil {
		return "", fmt.Errorf("parsing response body: %v", err)
	}
	token := q.Get("access_token")
	if token == "" {
		return "", fmt.Errorf("no access token in response body: %s", body)
	}

	if got := q.Get("scope"); got != c.scopes {
		return "", fmt.Errorf("requested scope %s got %s", c.scopes, got)
	}
	return token, nil
}

type gitHubUser struct {
	Login             string    `json:"login"`
	ID                int64     `json:"id"`
	AvatarURL         string    `json:"avatar_url"`
	GravatarID        string    `json:"gravatar_id"`
	URL               string    `json:"url"`
	HTMLURL           string    `json:"html_url"`
	FollowersURL      string    `json:"followers_url"`
	FollowingURL      string    `json:"following_url"`
	GistsURL          string    `json:"gists_url"`
	StarredURL        string    `json:"starred_url"`
	SubscriptionsURL  string    `json:"subscriptions_url"`
	OrganizationsURL  string    `json:"organizations_url"`
	ReposURL          string    `json:"repos_url"`
	EventsURL         string    `json:"events_url"`
	ReceivedEventsURL string    `json:"received_events_url"`
	Type              string    `json:"type"`
	SiteAdmin         bool      `json:"site_admin"`
	Name              string    `json:"name"`
	Company           string    `json:"company"`
	Blog              string    `json:"blog"`
	Location          string    `json:"location"`
	Email             string    `json:"email"`
	Hireable          bool      `json:"hireable"`
	Bio               string    `json:"bio"`
	PublicRepos       int       `json:"public_repos"`
	PublicGists       int       `json:"public_gists"`
	Followers         int       `json:"followers"`
	Following         int       `json:"following"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func getGitHubUser(token string) (gitHubUser, error) {
	u := gitHubAPIUserURL + "?" + (url.Values{"access_token": []string{token}}).Encode()
	resp, err := http.Get(u)
	if err != nil {
		return gitHubUser{}, fmt.Errorf("get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// TODO (ericchiang): get actual error from response body
		return gitHubUser{}, fmt.Errorf("bad status from GitHub %s", resp.Status)
	}

	var user gitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return gitHubUser{}, fmt.Errorf("decode body: %v", err)
	}
	return user, nil
}

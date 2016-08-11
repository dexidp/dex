package connector

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"

	chttp "github.com/coreos/go-oidc/http"
	"github.com/coreos/go-oidc/oauth2"
	"github.com/coreos/go-oidc/oidc"
)

const (
	UAAConnectorType = "uaa"
)

type UAAConnectorConfig struct {
	ID           string `json:"id"`
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	ServerURL    string `json:"serverURL"`
}

// standard error form returned by UAA
type uaaError struct {
	ErrorDescription string `json:"error_description"`
	ErrorType        string `json:"error"`
}

type uaaOAuth2Connector struct {
	clientID     string
	clientSecret string
	client       *oauth2.Client
	uaaBaseURL   *url.URL
}

func init() {
	RegisterConnectorConfigType(UAAConnectorType, func() ConnectorConfig { return &UAAConnectorConfig{} })
}

func (cfg *UAAConnectorConfig) ConnectorID() string {
	return cfg.ID
}

func (cfg *UAAConnectorConfig) ConnectorType() string {
	return UAAConnectorType
}

func (cfg *UAAConnectorConfig) Connector(ns url.URL, lf oidc.LoginFunc, tpls *template.Template) (Connector, error) {
	uaaBaseURL, err := url.ParseRequestURI(cfg.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("Invalid configuration. UAA URL is invalid: %v", err)
	}
	if !uaaBaseURL.IsAbs() {
		return nil, fmt.Errorf("Invalid configuration. UAA URL must be absolute")
	}
	ns.Path = path.Join(ns.Path, httpPathCallback)
	oauth2Conn, err := newUAAConnector(cfg, uaaBaseURL, ns.String())
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

func (err uaaError) Error() string {
	return fmt.Sprintf("uaa (%s): %s", err.ErrorType, err.ErrorDescription)
}

func (c *uaaOAuth2Connector) Client() *oauth2.Client {
	return c.client
}

func (c *uaaOAuth2Connector) Healthy() error {
	return nil
}

func (c *uaaOAuth2Connector) Identity(cli chttp.Client) (oidc.Identity, error) {
	uaaUserInfoURL := *c.uaaBaseURL
	uaaUserInfoURL.Path = path.Join(uaaUserInfoURL.Path, "/userinfo")
	req, err := http.NewRequest("GET", uaaUserInfoURL.String(), nil)
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
		// attempt to decode error from UAA
		var authErr uaaError
		if err := json.NewDecoder(resp.Body).Decode(&authErr); err != nil {
			return oidc.Identity{}, oauth2.NewError(oauth2.ErrorAccessDenied)
		}
		return oidc.Identity{}, authErr
	case resp.StatusCode == http.StatusOK:
	default:
		return oidc.Identity{}, fmt.Errorf("unexpected status from providor %s", resp.Status)
	}
	var user struct {
		UserID   string `json:"user_id"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		UserName string `json:"user_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return oidc.Identity{}, fmt.Errorf("getting user info: %v", err)
	}
	name := user.Name
	if name == "" {
		name = user.UserName
	}
	return oidc.Identity{
		ID:    user.UserID,
		Name:  name,
		Email: user.Email,
	}, nil
}

func (c *uaaOAuth2Connector) TrustedEmailProvider() bool {
	return false
}

func newUAAConnector(cfg *UAAConnectorConfig, uaaBaseURL *url.URL, cbURL string) (oauth2Connector, error) {
	uaaAuthURL := *uaaBaseURL
	uaaTokenURL := *uaaBaseURL
	uaaAuthURL.Path = path.Join(uaaAuthURL.Path, "/oauth/authorize")
	uaaTokenURL.Path = path.Join(uaaTokenURL.Path, "/oauth/token")
	config := oauth2.Config{
		Credentials: oauth2.ClientCredentials{ID: cfg.ClientID, Secret: cfg.ClientSecret},
		AuthURL:     uaaAuthURL.String(),
		TokenURL:    uaaTokenURL.String(),
		Scope:       []string{"openid"},
		AuthMethod:  oauth2.AuthMethodClientSecretPost,
		RedirectURL: cbURL,
	}

	cli, err := oauth2.NewClient(http.DefaultClient, config)
	if err != nil {
		return nil, err
	}

	return &uaaOAuth2Connector{
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		client:       cli,
		uaaBaseURL:   uaaBaseURL,
	}, nil
}

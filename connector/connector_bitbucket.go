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
	BitbucketConnectorType = "bitbucket"
	bitbucketAuthURL       = "https://bitbucket.org/site/oauth2/authorize"
	bitbucketTokenURL      = "https://bitbucket.org/site/oauth2/access_token"
	bitbucketAPIUserURL    = "https://bitbucket.org/api/2.0/user"
	bitbucketAPIEmailURL   = "https://api.bitbucket.org/2.0/user/emails"
)

func init() {
	RegisterConnectorConfigType(BitbucketConnectorType, func() ConnectorConfig { return &BitbucketConnectorConfig{} })
}

type BitbucketConnectorConfig struct {
	ID           string `json:"id"`
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
}

func (cfg *BitbucketConnectorConfig) ConnectorID() string {
	return cfg.ID
}

func (cfg *BitbucketConnectorConfig) ConnectorType() string {
	return BitbucketConnectorType
}

func (cfg *BitbucketConnectorConfig) Connector(ns url.URL, lf oidc.LoginFunc, tpls *template.Template) (Connector, error) {
	ns.Path = path.Join(ns.Path, httpPathCallback)
	oauth2Conn, err := newBitbucketConnector(cfg.ClientID, cfg.ClientSecret, ns.String())
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

type bitbucketOAuth2Connector struct {
	clientID     string
	clientSecret string
	client       *oauth2.Client
}

func newBitbucketConnector(clientID, clientSecret, cbURL string) (oauth2Connector, error) {
	config := oauth2.Config{
		Credentials: oauth2.ClientCredentials{ID: clientID, Secret: clientSecret},
		AuthURL:     bitbucketAuthURL,
		TokenURL:    bitbucketTokenURL,
		AuthMethod:  oauth2.AuthMethodClientSecretPost,
		RedirectURL: cbURL,
	}

	cli, err := oauth2.NewClient(http.DefaultClient, config)
	if err != nil {
		return nil, err
	}

	return &bitbucketOAuth2Connector{
		clientID:     clientID,
		clientSecret: clientSecret,
		client:       cli,
	}, nil
}

func (c *bitbucketOAuth2Connector) Client() *oauth2.Client {
	return c.client
}

func (c *bitbucketOAuth2Connector) Identity(cli chttp.Client) (oidc.Identity, error) {
	var user struct {
		UUID        string `json:"uuid"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	}
	if err := getAndDecode(cli, bitbucketAPIUserURL, &user); err != nil {
		return oidc.Identity{}, fmt.Errorf("getting user info: %v", err)
	}

	name := user.DisplayName
	if name == "" {
		name = user.Username
	}

	var emails struct {
		Values []struct {
			Email     string `json:"email"`
			Confirmed bool   `json:"is_confirmed"`
			Primary   bool   `json:"is_primary"`
		} `json:"values"`
	}
	if err := getAndDecode(cli, bitbucketAPIEmailURL, &emails); err != nil {
		return oidc.Identity{}, fmt.Errorf("getting user email: %v", err)
	}
	email := ""
	for _, val := range emails.Values {
		if !val.Confirmed {
			continue
		}
		if email == "" || val.Primary {
			email = val.Email
		}
		if val.Primary {
			break
		}
	}

	return oidc.Identity{
		ID:    user.UUID,
		Name:  name,
		Email: email,
	}, nil
}

func getAndDecode(cli chttp.Client, url string, v interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("get: %v", err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		return oauth2.NewError(oauth2.ErrorAccessDenied)
	case resp.StatusCode == http.StatusOK:
	default:
		return fmt.Errorf("unexpected status from providor %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("decode body: %v", err)
	}
	return nil
}

func (c *bitbucketOAuth2Connector) Healthy() error {
	return nil
}

func (c *bitbucketOAuth2Connector) TrustedEmailProvider() bool {
	return false
}

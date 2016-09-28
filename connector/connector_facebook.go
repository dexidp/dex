package connector

import (
	"encoding/json"
	"fmt"
	chttp "github.com/coreos/go-oidc/http"
	"github.com/coreos/go-oidc/oauth2"
	"github.com/coreos/go-oidc/oidc"
	"html/template"
	"net/http"
	"net/url"
	"path"
)

const (
	FacebookConnectorType    = "facebook"
	facebookConnectorAuthURL = "https://www.facebook.com/dialog/oauth"
	facebookTokenURL         = "https://graph.facebook.com/v2.3/oauth/access_token"
	facebookGraphAPIURL      = "https://graph.facebook.com/me?fields=id,name,email"
)

type FacebookConnectorConfig struct {
	ID           string `json:"id"`
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
}

func init() {
	RegisterConnectorConfigType(FacebookConnectorType, func() ConnectorConfig { return &FacebookConnectorConfig{} })
}

func (cfg *FacebookConnectorConfig) ConnectorID() string {
	return cfg.ID
}

func (cfg *FacebookConnectorConfig) ConnectorType() string {
	return FacebookConnectorType
}

func (cfg *FacebookConnectorConfig) Connector(ns url.URL, lf oidc.LoginFunc, tpls *template.Template) (Connector, error) {
	ns.Path = path.Join(ns.Path, httpPathCallback)
	oauth2Conn, err := newFacebookConnector(cfg.ClientID, cfg.ClientSecret, ns.String())
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

type facebookOAuth2Connector struct {
	clientID     string
	clientSecret string
	client       *oauth2.Client
}

func newFacebookConnector(clientID, clientSecret, cbURL string) (oauth2Connector, error) {
	config := oauth2.Config{
		Credentials: oauth2.ClientCredentials{ID: clientID, Secret: clientSecret},
		AuthURL:     facebookConnectorAuthURL,
		TokenURL:    facebookTokenURL,
		AuthMethod:  oauth2.AuthMethodClientSecretPost,
		RedirectURL: cbURL,
		Scope:       []string{"email"},
	}

	cli, err := oauth2.NewClient(http.DefaultClient, config)
	if err != nil {
		return nil, err
	}

	return &facebookOAuth2Connector{
		clientID:     clientID,
		clientSecret: clientSecret,
		client:       cli,
	}, nil
}
func (c *facebookOAuth2Connector) Client() *oauth2.Client {
	return c.client
}

func (c *facebookOAuth2Connector) Healthy() error {
	return nil
}

func (c *facebookOAuth2Connector) TrustedEmailProvider() bool {
	return false
}

type ErrorMessage struct {
	Message        string `json:"message"`
	Type           string `json:"type"`
	Code           int    `json:"code"`
	ErrorSubCode   int    `json:"error_subcode"`
	ErrorUserTitle string `json:"error_user_title"`
	ErrorUserMsg   string `json:"error_user_msg"`
	FbTraceId      string `json:"fbtrace_id"`
}

type facebookErr struct {
	ErrorMessage ErrorMessage `json:"error"`
}

func (err facebookErr) Error() string {
	return fmt.Sprintf("facebook: %s", err.ErrorMessage.Message)
}

func (c *facebookOAuth2Connector) Identity(cli chttp.Client) (oidc.Identity, error) {
	var user struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	req, err := http.NewRequest("GET", facebookGraphAPIURL, nil)
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
		var authErr facebookErr
		if err := json.NewDecoder(resp.Body).Decode(&authErr); err != nil {
			return oidc.Identity{}, oauth2.NewError(oauth2.ErrorAccessDenied)
		}
		return oidc.Identity{}, authErr
	case resp.StatusCode == http.StatusOK:
	default:
		return oidc.Identity{}, fmt.Errorf("unexpected status from providor %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return oidc.Identity{}, fmt.Errorf("decode body: %v", err)
	}

	return oidc.Identity{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	}, nil
}

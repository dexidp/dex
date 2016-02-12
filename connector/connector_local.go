package connector

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"

	"github.com/coreos/dex/user"
	"github.com/coreos/go-oidc/oidc"
)

const (
	LocalConnectorType    = "local"
	LoginPageTemplateName = "local-login.html"
)

func init() {
	RegisterConnectorConfigType(LocalConnectorType, func() ConnectorConfig { return &LocalConnectorConfig{} })
}

type LocalConnectorConfig struct {
	ID            string              `json:"id"`
	PasswordInfos []user.PasswordInfo `json:"passwordInfos"`
}

func (cfg *LocalConnectorConfig) ConnectorID() string {
	return cfg.ID
}

func (cfg *LocalConnectorConfig) ConnectorType() string {
	return LocalConnectorType
}

func (cfg *LocalConnectorConfig) Connector(ns url.URL, lf oidc.LoginFunc, tpls *template.Template) (Connector, error) {
	tpl := tpls.Lookup(LoginPageTemplateName)
	if tpl == nil {
		return nil, fmt.Errorf("unable to find necessary HTML template")
	}

	idpc := &LocalConnector{
		id:        cfg.ID,
		namespace: ns,
		loginFunc: lf,
		loginTpl:  tpl,
	}

	return idpc, nil
}

type LocalConnector struct {
	id        string
	idp       *LocalIdentityProvider
	namespace url.URL
	loginFunc oidc.LoginFunc
	loginTpl  *template.Template
}

type Page struct {
	PostURL    string
	Name       string
	Error      bool
	Message    string
	SessionKey string
}

func (c *LocalConnector) ID() string {
	return c.id
}

func (c *LocalConnector) Healthy() error {
	return nil
}

func (c *LocalConnector) SetLocalIdentityProvider(idp *LocalIdentityProvider) {
	c.idp = idp
}

func (c *LocalConnector) LoginURL(sessionKey, prompt string) (string, error) {
	q := url.Values{}
	q.Set("session_key", sessionKey)
	q.Set("prompt", prompt)
	enc := q.Encode()

	return path.Join(c.namespace.Path, "login") + "?" + enc, nil
}

func (c *LocalConnector) Register(mux *http.ServeMux, errorURL url.URL) {
	route := c.namespace.Path + "/login"
	mux.Handle(route, handleLoginFunc(c.loginFunc, c.loginTpl, c.idp, route, errorURL))
}

func (c *LocalConnector) Sync() chan struct{} {
	return make(chan struct{})
}

func (c *LocalConnector) TrustedEmailProvider() bool {
	return false
}

type LocalIdentityProvider struct {
	PasswordInfoRepo user.PasswordInfoRepo
	UserRepo         user.UserRepo
}

func (m *LocalIdentityProvider) Identity(email, password string) (*oidc.Identity, error) {
	user, err := m.UserRepo.GetByEmail(nil, email)
	if err != nil {
		return nil, err
	}

	id := user.ID

	pi, err := m.PasswordInfoRepo.Get(nil, id)
	if err != nil {
		return nil, err
	}

	return pi.Authenticate(password)
}

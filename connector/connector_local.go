package connector

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"

	phttp "github.com/coreos/dex/pkg/http"
	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/user"
	"github.com/coreos/go-oidc/oauth2"
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

func redirectPostError(w http.ResponseWriter, errorURL url.URL, q url.Values) {
	redirectURL := phttp.MergeQuery(errorURL, q)
	w.Header().Set("Location", redirectURL.String())
	w.WriteHeader(http.StatusSeeOther)
}

func handleLoginFunc(lf oidc.LoginFunc, tpl *template.Template, idp *LocalIdentityProvider, localErrorPath string, errorURL url.URL) http.HandlerFunc {
	handleGET := func(w http.ResponseWriter, r *http.Request, errMsg string) {
		q := r.URL.Query()
		sessionKey := q.Get("session_key")

		p := &Page{PostURL: r.URL.String(), Name: "Local", SessionKey: sessionKey}
		if errMsg != "" {
			p.Error = true
			p.Message = errMsg
		}

		if err := tpl.Execute(w, p); err != nil {
			phttp.WriteError(w, http.StatusInternalServerError, err.Error())
		}
	}

	handlePOST := func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			msg := fmt.Sprintf("unable to parse form from body: %v", err)
			phttp.WriteError(w, http.StatusBadRequest, msg)
			return
		}

		userid := r.PostForm.Get("userid")
		if userid == "" {
			handleGET(w, r, "missing email address")
			return
		}

		password := r.PostForm.Get("password")
		if password == "" {
			handleGET(w, r, "missing password")
			return
		}

		ident, err := idp.Identity(userid, password)
		log.Errorf("IDENTITY: err: %v", err)

		if ident == nil || err != nil {
			handleGET(w, r, "invalid login")
			return
		}

		q := r.URL.Query()
		sessionKey := r.FormValue("session_key")
		if sessionKey == "" {
			q.Set("error", oauth2.ErrorInvalidRequest)
			q.Set("error_description", "missing session_key")
			redirectPostError(w, errorURL, q)
			return
		}

		redirectURL, err := lf(*ident, sessionKey)
		if err != nil {
			log.Errorf("Unable to log in %#v: %v", *ident, err)
			q.Set("error", oauth2.ErrorAccessDenied)
			q.Set("error_description", "login failed")
			redirectPostError(w, errorURL, q)
			return
		}

		w.Header().Set("Location", redirectURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			handlePOST(w, r)
		case "GET":
			handleGET(w, r, "")
		default:
			w.Header().Set("Allow", "GET, POST")
			phttp.WriteError(w, http.StatusMethodNotAllowed, "GET and POST only acceptable methods")
		}
	}
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

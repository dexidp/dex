package connector

import (
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"strings"

	phttp "github.com/coreos/dex/pkg/http"
	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/go-oidc/oauth2"
	"github.com/coreos/go-oidc/oidc"
)

const (
	OIDCConnectorType = "oidc"
	httpPathCallback  = "/callback"
)

func init() {
	RegisterConnectorConfigType(OIDCConnectorType, func() ConnectorConfig { return &OIDCConnectorConfig{} })
}

type OIDCConnectorConfig struct {
	ID                   string `json:"id"`
	IssuerURL            string `json:"issuerURL"`
	ClientID             string `json:"clientID"`
	ClientSecret         string `json:"clientSecret"`
	TrustedEmailProvider bool   `json:"trustedEmailProvider"`
	Domain               string `json:"domain"`
}

func (cfg *OIDCConnectorConfig) ConnectorID() string {
	return cfg.ID
}

func (cfg *OIDCConnectorConfig) ConnectorType() string {
	return OIDCConnectorType
}

type validateIdentityFunc func(id *oidc.Identity) error

func validateIdentityPass(id *oidc.Identity) error {
	return nil
}

func validateIdentityDomainFunc(want string) validateIdentityFunc {
	return func(ident *oidc.Identity) error {
		got, err := parseEmailDomain(ident)
		if err != nil {
			return err
		} else if want != got {
			return errors.New("domain mismatch")
		}
		return nil
	}
}

func parseEmailDomain(ident *oidc.Identity) (string, error) {
	// minimum viable email address is a@b
	if len(ident.Email) < 3 {
		return "", errors.New("invalid email address")
	}
	// assert that an @ is found, and it isn't the first or last character
	idx := strings.LastIndex(ident.Email, "@")
	if idx < 1 || idx == len(ident.Email)-1 {
		return "", errors.New("invalid email address")
	}
	return ident.Email[idx+1:], nil
}

type OIDCConnector struct {
	id                   string
	issuerURL            string
	cbURL                url.URL
	loginFunc            oidc.LoginFunc
	client               *oidc.Client
	trustedEmailProvider bool
	validateIdentity     validateIdentityFunc
}

func (cfg *OIDCConnectorConfig) Connector(ns url.URL, lf oidc.LoginFunc, tpls *template.Template) (Connector, error) {
	ns.Path = path.Join(ns.Path, httpPathCallback)

	ccfg := oidc.ClientConfig{
		RedirectURL: ns.String(),
		Credentials: oidc.ClientCredentials{
			ID:     cfg.ClientID,
			Secret: cfg.ClientSecret,
		},
	}

	cl, err := oidc.NewClient(ccfg)
	if err != nil {
		return nil, err
	}

	var vif validateIdentityFunc
	if cfg.Domain != "" {
		vif = validateIdentityDomainFunc(cfg.Domain)
	} else {
		vif = validateIdentityPass
	}

	idpc := &OIDCConnector{
		id:                   cfg.ID,
		issuerURL:            cfg.IssuerURL,
		cbURL:                ns,
		loginFunc:            lf,
		client:               cl,
		trustedEmailProvider: cfg.TrustedEmailProvider,
		validateIdentity:     vif,
	}

	return idpc, nil
}

func (c *OIDCConnector) ID() string {
	return c.id
}

func (c *OIDCConnector) Healthy() error {
	return c.client.Healthy()
}

func (c *OIDCConnector) LoginURL(sessionKey, prompt string) (string, error) {
	oac, err := c.client.OAuthClient()
	if err != nil {
		return "", err
	}

	return oac.AuthCodeURL(sessionKey, "", prompt), nil
}

func (c *OIDCConnector) Register(mux *http.ServeMux, errorURL url.URL) {
	mux.Handle(c.cbURL.Path, c.handleCallbackFunc(c.loginFunc, errorURL))
}

func (c *OIDCConnector) Sync() chan struct{} {
	return c.client.SyncProviderConfig(c.issuerURL)
}

func (c *OIDCConnector) TrustedEmailProvider() bool {
	return c.trustedEmailProvider
}

func redirectError(w http.ResponseWriter, errorURL url.URL, q url.Values) {
	redirectURL := phttp.MergeQuery(errorURL, q)
	w.Header().Set("Location", redirectURL.String())
	w.WriteHeader(http.StatusSeeOther)
}

func (c *OIDCConnector) handleCallbackFunc(lf oidc.LoginFunc, errorURL url.URL) http.HandlerFunc {
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

		tok, err := c.client.ExchangeAuthCode(code)
		if err != nil {
			log.Errorf("Unable to verify auth code with issuer: %v", err)
			q.Set("error", oauth2.ErrorUnsupportedResponseType)
			q.Set("error_description", "unable to verify auth code with issuer")
			redirectError(w, errorURL, q)
			return
		}

		claims, err := tok.Claims()
		if err != nil {
			log.Errorf("Unable to construct claims: %v", err)
			q.Set("error", oauth2.ErrorUnsupportedResponseType)
			q.Set("error_description", "unable to construct claims")
			redirectError(w, errorURL, q)
			return
		}

		ident, err := oidc.IdentityFromClaims(claims)
		if err != nil {
			log.Errorf("Failed parsing claims from remote provider: %v", err)
			q.Set("error", oauth2.ErrorUnsupportedResponseType)
			q.Set("error_description", "unable to convert claims to identity")
			redirectError(w, errorURL, q)
			return
		}

		if err := c.validateIdentity(ident); err != nil {
			log.Infof("Identity validation failed: %v", err)
			q.Set("error", oauth2.ErrorAccessDenied)
			q.Set("error_description", "invalid identity")
			redirectError(w, errorURL, q)
			return
		}

		sessionKey := q.Get("state")
		if sessionKey == "" {
			q.Set("error", oauth2.ErrorInvalidRequest)
			q.Set("error_description", "missing state query param")
			redirectError(w, errorURL, q)
			return
		}

		redirectURL, err := lf(*ident, sessionKey)
		if err != nil {
			log.Errorf("Unable to log in %#v: %v", *ident, err)
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

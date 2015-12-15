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

func parseEmailDomain(email string) (string, error) {
	// minimum viable email address is a@b
	if len(email) < 3 {
		return "", errors.New("invalid email address")
	}
	// assert that an @ is found, and it isn't the first or last character
	idx := strings.LastIndex(email, "@")
	if idx < 1 || idx == len(email)-1 {
		return "", errors.New("invalid email address")
	}
	return email[idx+1:], nil
}

type OIDCConnector struct {
	id                   string
	issuerURL            string
	cbURL                url.URL
	loginFunc            oidc.LoginFunc
	client               *oidc.Client
	trustedEmailProvider bool
	domain               string
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

	idpc := &OIDCConnector{
		id:                   cfg.ID,
		issuerURL:            cfg.IssuerURL,
		cbURL:                ns,
		loginFunc:            lf,
		client:               cl,
		trustedEmailProvider: cfg.TrustedEmailProvider,
		domain:               cfg.Domain,
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

		ok, err := c.validateRemoteIdentity(ident)
		if !ok {
			q.Set("error", oauth2.ErrorAccessDenied)
			q.Set("error_description", "invalid remote identity")
			redirectError(w, errorURL, q)
			return
		} else if err != nil {
			log.Errorf("Identity validation failed: %v", err)
			q.Set("error", oauth2.ErrorServerError)
			q.Set("error_description", "identity validation failed")
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

func (c *OIDCConnector) validateRemoteIdentity(ident *oidc.Identity) (bool, error) {
	got, err := parseEmailDomain(ident.Email)
	if err != nil {
		return false, err
	}

	if c.domain != "" && c.domain != got {
		log.Debug("Remote identity invalid: unrecognized domain")
		return false, nil
	}

	return true, nil
}

package connector

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/coreos/dex/pkg/log"
	chttp "github.com/coreos/go-oidc/http"
	"github.com/coreos/go-oidc/oauth2"
	"github.com/coreos/go-oidc/oidc"
)

type oauth2Connector interface {
	Client() *oauth2.Client

	// Identity uses a HTTP client authenticated as the end user to construct
	// an OIDC identity for that user.
	Identity(cli chttp.Client) (oidc.Identity, error)

	// Healthy it should attempt to determine if the connector's credientials
	// are valid.
	Healthy() error

	TrustedEmailProvider() bool
}

type OAuth2Connector struct {
	id        string
	loginFunc oidc.LoginFunc
	cbURL     url.URL
	conn      oauth2Connector
}

func (c *OAuth2Connector) ID() string {
	return c.id
}

func (c *OAuth2Connector) Healthy() error {
	return c.conn.Healthy()
}

func (c *OAuth2Connector) Sync() chan struct{} {
	stop := make(chan struct{}, 1)
	return stop
}

func (c *OAuth2Connector) TrustedEmailProvider() bool {
	return c.conn.TrustedEmailProvider()
}

func (c *OAuth2Connector) LoginURL(sessionKey, prompt string) (string, error) {
	return c.conn.Client().AuthCodeURL(sessionKey, oauth2.GrantTypeAuthCode, prompt), nil
}

func (c *OAuth2Connector) Register(mux *http.ServeMux, errorURL url.URL) {
	mux.Handle(c.cbURL.Path, c.handleCallbackFunc(c.loginFunc, errorURL))
}

func (c *OAuth2Connector) handleCallbackFunc(lf oidc.LoginFunc, errorURL url.URL) http.HandlerFunc {
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

		token, err := c.conn.Client().RequestToken(oauth2.GrantTypeAuthCode, code)
		if err != nil {
			log.Errorf("Unable to verify auth code with issuer: %v", err)
			q.Set("error", oauth2.ErrorUnsupportedResponseType)
			q.Set("error_description", "unable to verify auth code with issuer")
			redirectError(w, errorURL, q)
			return
		}
		ident, err := c.conn.Identity(newAuthenticatedClient(token, http.DefaultClient))
		if err != nil {
			log.Errorf("Unable to retrieve identity: %v", err)
			q.Set("error", oauth2.ErrorUnsupportedResponseType)
			q.Set("error_description", "unable to retrieve identity from issuer")
			redirectError(w, errorURL, q)
			return
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
		w.WriteHeader(http.StatusFound)
		return
	}
}

// authedClient authenticates all requests as the end user.
type authedClient struct {
	token oauth2.TokenResponse
	cli   chttp.Client
}

func newAuthenticatedClient(token oauth2.TokenResponse, cli chttp.Client) chttp.Client {
	return &authedClient{token, cli}
}

func (c *authedClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", tokenType(c.token)+" "+c.token.AccessToken)
	return c.cli.Do(req)
}

// Return the canonical name of the token type if non-empty, else "Bearer".
// Take from golang.org/x/oauth2
func tokenType(token oauth2.TokenResponse) string {
	if strings.EqualFold(token.TokenType, "bearer") {
		return "Bearer"
	}
	if strings.EqualFold(token.TokenType, "mac") {
		return "MAC"
	}
	if strings.EqualFold(token.TokenType, "basic") {
		return "Basic"
	}
	if token.TokenType != "" {
		return token.TokenType
	}
	return "Bearer"
}

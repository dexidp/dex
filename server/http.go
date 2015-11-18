package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oauth2"
	"github.com/coreos/go-oidc/oidc"
	"github.com/coreos/pkg/health"
	"github.com/jonboulle/clockwork"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/connector"
	phttp "github.com/coreos/dex/pkg/http"
	"github.com/coreos/dex/pkg/log"
)

const (
	lastSeenMaxAge  = time.Minute * 5
	discoveryMaxAge = time.Hour * 24
)

var (
	httpPathDiscovery         = "/.well-known/openid-configuration"
	httpPathToken             = "/token"
	httpPathKeys              = "/keys"
	httpPathAuth              = "/auth"
	httpPathHealth            = "/health"
	httpPathAPI               = "/api"
	httpPathRegister          = "/register"
	httpPathEmailVerify       = "/verify-email"
	httpPathVerifyEmailResend = "/resend-verify-email"
	httpPathSendResetPassword = "/send-reset-password"
	httpPathResetPassword     = "/reset-password"
	httpPathAcceptInvitation  = "/accept-invitation"
	httpPathDebugVars         = "/debug/vars"

	cookieLastSeen                 = "LastSeen"
	cookieShowEmailVerifiedMessage = "ShowEmailVerifiedMessage"
)

func handleDiscoveryFunc(cfg oidc.ProviderConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.Header().Set("Allow", "GET")
			phttp.WriteError(w, http.StatusMethodNotAllowed, "GET only acceptable method")
			return
		}

		b, err := json.Marshal(cfg)
		if err != nil {
			log.Errorf("Unable to marshal %#v to JSON: %v", cfg, err)
		}

		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(discoveryMaxAge.Seconds())))
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}
}

func handleKeysFunc(km key.PrivateKeyManager, clock clockwork.Clock) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.Header().Set("Allow", "GET")
			phttp.WriteError(w, http.StatusMethodNotAllowed, "GET only acceptable method")
			return
		}

		jwks, err := km.JWKs()
		if err != nil {
			log.Errorf("Failed to get JWKs while serving HTTP request: %v", err)
			phttp.WriteError(w, http.StatusInternalServerError, "")
			return
		}

		keys := struct {
			Keys []jose.JWK `json:"keys"`
		}{
			Keys: jwks,
		}

		b, err := json.Marshal(keys)
		if err != nil {
			log.Errorf("Unable to marshal signing key to JSON: %v", err)
		}

		exp := km.ExpiresAt()
		w.Header().Set("Expires", exp.Format(time.RFC1123))

		ttl := int(exp.Sub(clock.Now()).Seconds())
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", ttl))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}

type Link struct {
	URL         string
	ID          string
	DisplayName string
}

type templateData struct {
	Error                    bool
	Message                  string
	Instruction              string
	Detail                   string
	Register                 bool
	RegisterOrLoginURL       string
	MsgCode                  string
	ShowEmailVerifiedMessage bool
	Links                    []Link
}

// TODO(sym3tri): store this with the connector config
var connectorDisplayNameMap = map[string]string{
	"google": "Google",
	"local":  "Email",
}

func execTemplate(w http.ResponseWriter, tpl *template.Template, data interface{}) {
	execTemplateWithStatus(w, tpl, data, http.StatusOK)
}

func execTemplateWithStatus(w http.ResponseWriter, tpl *template.Template, data interface{}, status int) {
	w.WriteHeader(status)
	if err := tpl.Execute(w, data); err != nil {
		log.Errorf("Error loading page: %q", err)
		phttp.WriteError(w, http.StatusInternalServerError, "error loading page")
		return
	}
}

func renderLoginPage(w http.ResponseWriter, r *http.Request, srv OIDCServer, idpcs []connector.Connector, register bool, tpl *template.Template) {
	if tpl == nil {
		phttp.WriteError(w, http.StatusInternalServerError, "error loading login page")
		return
	}

	td := templateData{
		Message:                  "Error",
		Instruction:              "Please try again or contact the system administrator",
		Register:                 register,
		ShowEmailVerifiedMessage: consumeShowEmailVerifiedCookie(r, w),
	}

	// Render error if remote IdP connector errored and redirected here.
	q := r.URL.Query()
	e := q.Get("error")
	connectorID := q.Get("connector_id")
	if e != "" {
		td.Error = true
		td.Message = "Authentication Error"
		remoteMsg := q.Get("error_description")
		if remoteMsg == "" {
			remoteMsg = q.Get("error")
		}
		if connectorID == "" {
			td.Detail = remoteMsg
		} else {
			td.Detail = fmt.Sprintf("Error from %s: %s.", connectorID, remoteMsg)
		}
		execTemplate(w, tpl, td)
		return
	}

	if q.Get("msg_code") != "" {
		td.MsgCode = q.Get("msg_code")
	}

	// Render error message if client id is invalid.
	clientID := q.Get("client_id")
	cm, err := srv.ClientMetadata(clientID)
	if err != nil {
		log.Errorf("Failed fetching client %q from repo: %v", clientID, err)
		td.Error = true
		td.Message = "Server Error"
		execTemplate(w, tpl, td)
		return
	}
	if cm == nil {
		td.Error = true
		td.Message = "Authentication Error"
		td.Detail = "Invalid client ID"
		execTemplate(w, tpl, td)
		return
	}

	if len(idpcs) == 0 {
		td.Error = true
		td.Message = "Server Error"
		td.Instruction = "Unable to authenticate users at this time"
		td.Detail = "Authentication service may be misconfigured"
		execTemplate(w, tpl, td)
		return
	}

	link := *r.URL
	linkParams := link.Query()
	if !register {
		linkParams.Set("register", "1")
	} else {
		linkParams.Del("register")
	}
	linkParams.Del("msg_code")
	linkParams.Del("show_connectors")
	link.RawQuery = linkParams.Encode()
	td.RegisterOrLoginURL = link.String()

	var showConnectors map[string]struct{}

	// Only show the following connectors, if param is present
	if q.Get("show_connectors") != "" {
		conns := strings.Split(q.Get("show_connectors"), ",")
		if len(conns) != 0 {
			showConnectors = make(map[string]struct{})
			for _, connID := range conns {
				showConnectors[connID] = struct{}{}
			}
		}
	}

	for _, idpc := range idpcs {
		id := idpc.ID()
		if showConnectors != nil {
			if _, ok := showConnectors[id]; !ok {
				continue
			}
		}
		var link Link
		link.ID = id

		displayName, ok := connectorDisplayNameMap[id]
		if !ok {
			displayName = id
		}
		link.DisplayName = displayName

		v := r.URL.Query()
		v.Set("connector_id", idpc.ID())
		v.Set("response_type", "code")
		link.URL = httpPathAuth + "?" + v.Encode()
		td.Links = append(td.Links, link)
	}

	execTemplate(w, tpl, td)
}

func handleAuthFunc(srv OIDCServer, idpcs []connector.Connector, tpl *template.Template, registrationEnabled bool) http.HandlerFunc {
	idx := makeConnectorMap(idpcs)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.Header().Set("Allow", "GET")
			phttp.WriteError(w, http.StatusMethodNotAllowed, "GET only acceptable method")
			return
		}

		q := r.URL.Query()
		register := q.Get("register") == "1" && registrationEnabled
		e := q.Get("error")
		if e != "" {
			sessionKey := q.Get("state")
			if err := srv.KillSession(sessionKey); err != nil {
				log.Errorf("Failed killing sessionKey %q: %v", sessionKey, err)
			}
			renderLoginPage(w, r, srv, idpcs, register, tpl)
			return
		}

		connectorID := q.Get("connector_id")
		idpc, ok := idx[connectorID]
		if !ok {
			renderLoginPage(w, r, srv, idpcs, register, tpl)
			return
		}

		acr, err := oauth2.ParseAuthCodeRequest(q)
		if err != nil {
			log.Errorf("Invalid auth request: %v", err)
			writeAuthError(w, err, acr.State)
			return
		}

		cm, err := srv.ClientMetadata(acr.ClientID)
		if err != nil {
			log.Errorf("Failed fetching client %q from repo: %v", acr.ClientID, err)
			writeAuthError(w, oauth2.NewError(oauth2.ErrorServerError), acr.State)
			return
		}
		if cm == nil {
			log.Errorf("Client %q not found", acr.ClientID)
			writeAuthError(w, oauth2.NewError(oauth2.ErrorInvalidRequest), acr.State)
			return
		}

		if len(cm.RedirectURLs) == 0 {
			log.Errorf("Client %q has no redirect URLs", acr.ClientID)
			writeAuthError(w, oauth2.NewError(oauth2.ErrorServerError), acr.State)
			return
		}

		redirectURL, err := client.ValidRedirectURL(acr.RedirectURL, cm.RedirectURLs)
		if err != nil {
			switch err {
			case (client.ErrorCantChooseRedirectURL):
				log.Errorf("Request must provide redirect URL as client %q has registered many", acr.ClientID)
				writeAuthError(w, oauth2.NewError(oauth2.ErrorInvalidRequest), acr.State)
				return
			case (client.ErrorInvalidRedirectURL):
				log.Errorf("Request provided unregistered redirect URL: %s", acr.RedirectURL)
				writeAuthError(w, oauth2.NewError(oauth2.ErrorInvalidRequest), acr.State)
				return
			case (client.ErrorNoValidRedirectURLs):
				log.Errorf("There are no registered URLs for the requested client: %s", acr.RedirectURL)
				writeAuthError(w, oauth2.NewError(oauth2.ErrorInvalidRequest), acr.State)
				return
			}
		}

		if acr.ResponseType != oauth2.ResponseTypeCode {
			log.Errorf("unexpected ResponseType: %v: ", acr.ResponseType)
			redirectAuthError(w, oauth2.NewError(oauth2.ErrorUnsupportedResponseType), acr.State, redirectURL)
			return
		}

		// Check scopes.
		var scopes []string
		foundOpenIDScope := false
		for _, scope := range acr.Scope {
			switch scope {
			case "openid":
				foundOpenIDScope = true
				scopes = append(scopes, scope)
			case "offline_access":
				// According to the spec, for offline_access scope, the client must
				// use a response_type value that would result in an Authorization Code.
				// Currently oauth2.ResponseTypeCode is the only supported response type,
				// and it's been checked above, so we don't need to check it again here.
				//
				// TODO(yifan): Verify that 'consent' should be in 'prompt'.
				scopes = append(scopes, scope)
			default:
				// Pass all other scopes.
				scopes = append(scopes, scope)
			}
		}

		if !foundOpenIDScope {
			log.Errorf("Invalid auth request: missing 'openid' in 'scope'")
			writeAuthError(w, oauth2.NewError(oauth2.ErrorInvalidRequest), acr.State)
			return
		}

		nonce := q.Get("nonce")

		key, err := srv.NewSession(connectorID, acr.ClientID, acr.State, redirectURL, nonce, register, acr.Scope)
		if err != nil {
			log.Errorf("Error creating new session: %v: ", err)
			redirectAuthError(w, err, acr.State, redirectURL)
			return
		}

		if register {
			_, ok := idpc.(*connector.LocalConnector)
			if ok {
				q := url.Values{}
				q.Set("code", key)
				ru := httpPathRegister + "?" + q.Encode()
				w.Header().Set("Location", ru)
				w.WriteHeader(http.StatusTemporaryRedirect)
				return
			}
		}

		var p string
		if register {
			p = "select_account consent"
		}
		if shouldReprompt(r) || register {
			p = "select_account"
		}
		lu, err := idpc.LoginURL(key, p)
		if err != nil {
			log.Errorf("Connector.LoginURL failed: %v", err)
			redirectAuthError(w, err, acr.State, redirectURL)
			return
		}

		http.SetCookie(w, createLastSeenCookie())
		w.Header().Set("Location", lu)
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	}
}

func handleTokenFunc(srv OIDCServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.Header().Set("Allow", "POST")
			phttp.WriteError(w, http.StatusMethodNotAllowed, fmt.Sprintf("POST only acceptable method"))
			return
		}

		err := r.ParseForm()
		if err != nil {
			log.Errorf("error parsing request: %v", err)
			writeTokenError(w, oauth2.NewError(oauth2.ErrorInvalidRequest), "")
			return
		}

		state := r.PostForm.Get("state")

		user, password, ok := r.BasicAuth()
		if !ok {
			log.Errorf("error parsing basic auth")
			writeTokenError(w, oauth2.NewError(oauth2.ErrorInvalidClient), state)
			return
		}

		creds := oidc.ClientCredentials{ID: user, Secret: password}

		var jwt *jose.JWT
		var refreshToken string
		grantType := r.PostForm.Get("grant_type")

		switch grantType {
		case oauth2.GrantTypeAuthCode:
			code := r.PostForm.Get("code")
			if code == "" {
				log.Errorf("missing code param")
				writeTokenError(w, oauth2.NewError(oauth2.ErrorInvalidRequest), state)
				return
			}
			jwt, refreshToken, err = srv.CodeToken(creds, code)
			if err != nil {
				log.Errorf("couldn't exchange code for token: %v", err)
				writeTokenError(w, err, state)
				return
			}
		case oauth2.GrantTypeClientCreds:
			jwt, err = srv.ClientCredsToken(creds)
			if err != nil {
				log.Errorf("couldn't creds for token: %v", err)
				writeTokenError(w, err, state)
				return
			}
		case oauth2.GrantTypeRefreshToken:
			token := r.PostForm.Get("refresh_token")
			if token == "" {
				writeTokenError(w, oauth2.NewError(oauth2.ErrorInvalidRequest), state)
				return
			}
			jwt, err = srv.RefreshToken(creds, token)
			if err != nil {
				writeTokenError(w, err, state)
				return
			}
		default:
			log.Errorf("unsupported grant: %v", grantType)
			writeTokenError(w, oauth2.NewError(oauth2.ErrorUnsupportedGrantType), state)
			return
		}

		t := oAuth2Token{
			AccessToken:  jwt.Encode(),
			IDToken:      jwt.Encode(),
			TokenType:    "bearer",
			RefreshToken: refreshToken,
		}

		b, err := json.Marshal(t)
		if err != nil {
			log.Errorf("Failed marshaling %#v to JSON: %v", t, err)
			writeTokenError(w, oauth2.NewError(oauth2.ErrorServerError), state)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}

func makeHealthHandler(checks []health.Checkable) http.Handler {
	return health.Checker{
		Checks: checks,
	}
}

type oAuth2Token struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func createLastSeenCookie() *http.Cookie {
	now := time.Now()
	return &http.Cookie{
		HttpOnly: true,
		Name:     cookieLastSeen,
		MaxAge:   int(lastSeenMaxAge.Seconds()),
		// For old IE, ignored by most browsers.
		Expires: now.Add(lastSeenMaxAge),
	}
}

// shouldReprompt determines if user should be re-prompted for login based on existence of a cookie.
func shouldReprompt(r *http.Request) bool {
	_, err := r.Cookie(cookieLastSeen)
	if err == nil {
		return true
	}
	return false
}

func consumeShowEmailVerifiedCookie(r *http.Request, w http.ResponseWriter) bool {
	_, err := r.Cookie(cookieShowEmailVerifiedMessage)
	if err == nil {
		deleteCookie(w, cookieShowEmailVerifiedMessage)
		return true
	}
	return false
}

func deleteCookie(w http.ResponseWriter, name string) {
	now := time.Now()
	http.SetCookie(w, &http.Cookie{
		Name:    name,
		MaxAge:  -100,
		Expires: now.Add(time.Second * -100),
	})
}

func makeConnectorMap(idpcs []connector.Connector) map[string]connector.Connector {
	idx := make(map[string]connector.Connector, len(idpcs))
	for _, idpc := range idpcs {
		idx[idpc.ID()] = idpc
	}
	return idx
}

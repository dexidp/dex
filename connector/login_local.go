package connector

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	phttp "github.com/coreos/dex/pkg/http"
	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/go-oidc/oauth2"
	"github.com/coreos/go-oidc/oidc"
)

type JSONCredentials struct {
	ClientId string   `json:"client_id"`
	Scope    []string `json:"scope"`
	Nonce    string   `json:"nonce"`

	Login    string `json:"login"`
	Password string `json:"password"`
}

type JSONCodeResponse struct {
	Code string `json:"code"`
}

func redirectPostError(w http.ResponseWriter, errorURL url.URL, q url.Values) {
	redirectURL := phttp.MergeQuery(errorURL, q)
	w.Header().Set("Location", redirectURL.String())
	w.WriteHeader(http.StatusSeeOther)
}

// passwordLoginProvider is a provider which requires a username and password to identify the user.
type passwordLoginProvider interface {
	Identity(email, password string) (*oidc.Identity, error)
}

func handleRESTPasswordLogin(connectorType string, lf oidc.LoginFunc, nsf NewSessionFunc, idp passwordLoginProvider, localErrorPath string) http.HandlerFunc {
	handlePOST := func(w http.ResponseWriter, r *http.Request) {
		var credentials JSONCredentials
		err := json.NewDecoder(r.Body).Decode(&credentials)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if credentials.Login == "" || credentials.Password == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ident, err := idp.Identity(credentials.Login, credentials.Password)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		sessionKey, err := nsf(connectorType, credentials.ClientId, "NEW", url.URL{}, "", false, credentials.Scope)
		if err != nil {
			log.Errorf("Error creating new session key: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		rUrl, err := lf(*ident, sessionKey)
		if err != nil {
			log.Errorf("Error while logging in: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		p, err := url.Parse(rUrl)
		if err != nil {
			log.Errorf("Error while parsing url params: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		rawCode := p.Query().Get("code")

		writeResponseWithBody(w, http.StatusOK, JSONCodeResponse{Code: rawCode})
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			handlePOST(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func handlePasswordLogin(lf oidc.LoginFunc, tpl *template.Template, idp passwordLoginProvider, localErrorPath string, errorURL url.URL) http.HandlerFunc {
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
		if err != nil {
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
		w.WriteHeader(http.StatusFound)
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

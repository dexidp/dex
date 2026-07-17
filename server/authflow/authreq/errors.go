package authreq

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/dexidp/dex/storage"
)

// DisplayedErr is an error that should be displayed to the user as a web page.
// See RFC 6749 §4.1.2.1: an invalid client_id or redirect_uri is shown, not
// redirected.
type DisplayedErr struct {
	Status      int
	Description string
}

func (err *DisplayedErr) Error() string { return err.Description }

// NewDisplayedErr builds a DisplayedErr.
func NewDisplayedErr(status int, format string, a ...interface{}) *DisplayedErr {
	return &DisplayedErr{status, fmt.Sprintf(format, a...)}
}

// RedirectedErr is an error reported back to the client by 302 redirect.
type RedirectedErr struct {
	State       string
	RedirectURI string
	Type        string
	Description string
}

func (err *RedirectedErr) Error() string { return err.Description }

// Handler returns an http.Handler that redirects to the client with the error.
func (err *RedirectedErr) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := url.Values{}
		v.Add("state", err.State)
		v.Add("error", err.Type)
		if err.Description != "" {
			v.Add("error_description", err.Description)
		}

		// Parse the redirect URI to ensure it's valid before redirecting.
		u, parseErr := url.Parse(err.RedirectURI)
		if parseErr != nil {
			http.Error(w, "Invalid redirect URI", http.StatusBadRequest)
			return
		}

		query := u.Query()
		for key, values := range v {
			for _, value := range values {
				query.Add(key, value)
			}
		}
		u.RawQuery = query.Encode()

		http.Redirect(w, r, u.String(), http.StatusSeeOther)
	})
}

// RedirectWithError redirects back to the client with an OAuth2 error response.
// Used for prompt=none when login or consent is required.
func RedirectWithError(w http.ResponseWriter, r *http.Request, authReq *storage.AuthRequest, errType, description string) {
	err := &RedirectedErr{State: authReq.State, RedirectURI: authReq.RedirectURI, Type: errType, Description: description}
	err.Handler().ServeHTTP(w, r)
}

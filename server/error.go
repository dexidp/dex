package server

import (
	"net/http"
	"net/url"

	"github.com/coreos/go-oidc/oauth2"
)

const (
	errorInvalidClientMetadata = "invalid_client_metadata"
	errorInvalidRequest        = "invalid_request"
	errorServerError           = "server_error"
	errorAccessDenied          = "access_denied"
)

type apiError struct {
	Type        string `json:"error"`
	Description string `json:"error_description,omitempty"`
}

func (e *apiError) Error() string {
	return e.Type
}

func newAPIError(typ, desc string) *apiError {
	return &apiError{Type: typ, Description: desc}
}

func writeAPIError(w http.ResponseWriter, code int, err error) {
	aerr, ok := err.(*apiError)
	if !ok {
		aerr = newAPIError(errorServerError, "")
	}
	if aerr.Type == "" {
		aerr.Type = errorServerError
	}
	if code == 0 {
		code = http.StatusInternalServerError
	}
	writeResponseWithBody(w, code, aerr)
}

func writeTokenError(w http.ResponseWriter, err error, state string) {
	oerr, ok := err.(*oauth2.Error)
	if !ok {
		oerr = oauth2.NewError(oauth2.ErrorServerError)
	}
	oerr.State = state

	var status int
	switch oerr.Type {
	case oauth2.ErrorInvalidClient:
		status = http.StatusUnauthorized
		w.Header().Set("WWW-Authenticate", "Basic")
	default:
		status = http.StatusBadRequest
	}

	writeResponseWithBody(w, status, oerr)
}

func writeAuthError(w http.ResponseWriter, err error, state string) {
	oerr, ok := err.(*oauth2.Error)
	if !ok {
		oerr = oauth2.NewError(oauth2.ErrorServerError)
	}
	oerr.State = state
	writeResponseWithBody(w, http.StatusBadRequest, oerr)
}

func redirectAuthError(w http.ResponseWriter, err error, state string, redirectURL url.URL) {
	oerr, ok := err.(*oauth2.Error)
	if !ok {
		oerr = oauth2.NewError(oauth2.ErrorServerError)
	}

	q := redirectURL.Query()
	q.Set("error", oerr.Type)
	q.Set("state", state)
	redirectURL.RawQuery = q.Encode()

	w.Header().Set("Location", redirectURL.String())
	w.WriteHeader(http.StatusTemporaryRedirect)
}

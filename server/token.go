package server

// token.go implements the /token endpoint: it dispatches to the grants package
// and, for now, serves the device_code grant directly.

import (
	"net/http"
	"slices"

	"github.com/dexidp/dex/server/grants"
	"github.com/dexidp/dex/server/oauth2"
)

// newTokenEndpoint builds the token endpoint with its grants registered from the
// server's dependencies. It is the single construction point shared by the router
// wiring and the grant tests.
func (s *Server) newTokenEndpoint() *grants.Endpoint {
	return grants.NewEndpoint(s.issuer, s.storage, s.connectors, s.now, s.logger, s.passwordConnector, s.refreshTokenPolicy, s.sessionConfig != nil)
}

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request, endpoint *grants.Endpoint) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		s.tokenErrHelper(w, oauth2.InvalidRequest, "method not allowed", http.StatusBadRequest)
		return
	}

	err := r.ParseForm()
	if err != nil {
		s.logger.ErrorContext(r.Context(), "could not parse request body", "err", err)
		s.tokenErrHelper(w, oauth2.InvalidRequest, "", http.StatusBadRequest)
		return
	}

	grantType := r.PostFormValue("grant_type")
	if !slices.Contains(s.supportedGrantTypes, grantType) {
		s.logger.ErrorContext(r.Context(), "unsupported grant type", "grant_type", grantType)
		s.tokenErrHelper(w, oauth2.UnsupportedGrantType, "", http.StatusBadRequest)
		return
	}

	if !endpoint.Dispatch(w, r, grantType) {
		s.tokenErrHelper(w, oauth2.UnsupportedGrantType, "", http.StatusBadRequest)
	}
}

// handleDeviceTokenDeprecated serves the deprecated /device/token endpoint by
// dispatching the device_code grant through the token endpoint.
func (s *Server) handleDeviceTokenDeprecated(w http.ResponseWriter, r *http.Request, endpoint *grants.Endpoint) {
	s.logger.Warn(`the /device/token endpoint was called. It will be removed, use /token instead.`, "deprecated", true)

	if r.Method != http.MethodPost {
		s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := r.ParseForm(); err != nil {
		s.logger.Warn("could not parse Device Token Request body", "err", err)
		s.tokenErrHelper(w, oauth2.InvalidRequest, "", http.StatusBadRequest)
		return
	}
	if r.PostFormValue("grant_type") != oauth2.GrantTypeDeviceCode {
		s.tokenErrHelper(w, oauth2.InvalidGrant, "", http.StatusBadRequest)
		return
	}

	endpoint.Dispatch(w, r, oauth2.GrantTypeDeviceCode)
}

func (s *Server) tokenErrHelper(w http.ResponseWriter, typ string, description string, statusCode int) {
	if err := oauth2.WriteError(w, typ, description, statusCode); err != nil {
		// TODO(nabokihms): error with context
		s.logger.Error("token error response", "err", err)
	}
}

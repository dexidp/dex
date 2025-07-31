package server

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/otel/traces"
	"github.com/dexidp/dex/storage"
)

func (s *Server) handleConnectorCallback(w http.ResponseWriter, r *http.Request) {
	ctx, span := traces.InstrumentHandler(r)
	defer span.End()
	var authID string
	switch r.Method {
	case http.MethodGet: // OAuth2 callback
		if authID = r.URL.Query().Get("state"); authID == "" {
			s.renderError(r, w, http.StatusBadRequest, "User session error.")
			return
		}
	case http.MethodPost: // SAML POST binding
		if authID = r.PostFormValue("RelayState"); authID == "" {
			s.renderError(r, w, http.StatusBadRequest, "User session error.")
			return
		}
	default:
		s.renderError(r, w, http.StatusBadRequest, "Method not supported")
		return
	}

	authReq, err := s.storage.GetAuthRequest(ctx, authID)
	if err != nil {
		if err == storage.ErrNotFound {
			s.logger.ErrorContext(ctx, "invalid 'state' parameter provided", "err", err)
			s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
			return
		}
		s.logger.ErrorContext(ctx, "failed to get auth request", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}

	connID, err := url.PathUnescape(mux.Vars(r)["connector"])
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get connector", "connector_id", authReq.ConnectorID, "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	} else if connID != "" && connID != authReq.ConnectorID {
		s.logger.ErrorContext(ctx, "connector mismatch: callback triggered for different connector than authentication start", "authentication_start_connector_id", authReq.ConnectorID, "connector_id", connID)
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	conn, err := s.getConnector(ctx, authReq.ConnectorID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get connector", "connector_id", authReq.ConnectorID, "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	var identity connector.Identity
	switch conn := conn.Connector.(type) {
	case connector.CallbackConnector:
		if r.Method != http.MethodGet {
			s.logger.ErrorContext(ctx, "SAML request mapped to OAuth2 connector")
			s.renderError(r, w, http.StatusBadRequest, "Invalid request")
			return
		}
		identity, err = conn.HandleCallback(parseScopes(authReq.Scopes), r)
	case connector.SAMLConnector:
		if r.Method != http.MethodPost {
			s.logger.ErrorContext(ctx, "OAuth2 request mapped to SAML connector")
			s.renderError(r, w, http.StatusBadRequest, "Invalid request")
			return
		}
		identity, err = conn.HandlePOST(parseScopes(authReq.Scopes), r.PostFormValue("SAMLResponse"), authReq.ID)
	default:
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	if err != nil {
		s.logger.ErrorContext(ctx, "failed to authenticate", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, fmt.Sprintf("Failed to authenticate: %v", err))
		return
	}

	redirectURL, canSkipApproval, err := s.finalizeLogin(ctx, identity, authReq, conn.Connector)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to finalize login", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Login error.")
		return
	}

	if canSkipApproval {
		authReq, err = s.storage.GetAuthRequest(ctx, authReq.ID)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to get finalized auth request", "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "Login error.")
			return
		}
		s.sendCodeResponse(w, r, authReq)
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

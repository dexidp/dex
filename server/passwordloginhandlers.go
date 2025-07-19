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

func (s *Server) handlePasswordLogin(w http.ResponseWriter, r *http.Request) {
	ctx, span := traces.InstrumentHandler(r)
	defer span.End()
	connID, err := url.PathUnescape(mux.Vars(r)["connector"])
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to unescape connector ID", "err", err)
		s.renderError(r, w, http.StatusBadRequest, "Invalid connector ID.")
		return
	}

	authID := r.URL.Query().Get("state")
	if authID == "" {
		s.renderError(r, w, http.StatusBadRequest, "User session error.")
		return
	}

	backLink := r.URL.Query().Get("back")

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

	if connID != "" && connID != authReq.ConnectorID {
		s.logger.ErrorContext(ctx, "connector mismatch: password login triggered for different connector from authentication start", "start_connector_id", authReq.ConnectorID, "password_connector_id", connID)
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	conn, err := s.getConnector(ctx, authReq.ConnectorID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get connector", "connector_id", authReq.ConnectorID, "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Connector failed to initialize.")
		return
	}

	pwConn, ok := conn.Connector.(connector.PasswordConnector)
	if !ok {
		s.logger.ErrorContext(ctx, "expected password connector in handlePasswordLogin()", "password_connector", pwConn)
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	switch r.Method {
	case http.MethodGet:
		if err := s.templates.password(r, w, r.URL.String(), "", usernamePrompt(pwConn), false, backLink); err != nil {
			s.logger.ErrorContext(ctx, "server template error", "err", err)
		}
	case http.MethodPost:
		username := r.FormValue("login")
		password := r.FormValue("password")
		scopes := parseScopes(authReq.Scopes)

		identity, ok, err := pwConn.Login(ctx, scopes, username, password)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to login user", "err", err)
			s.renderError(r, w, http.StatusInternalServerError, fmt.Sprintf("Login error: %v", err))
			return
		}

		if !ok {
			if err := s.templates.password(r, w, r.URL.String(), username, usernamePrompt(pwConn), true, backLink); err != nil {
				s.logger.ErrorContext(ctx, "server template error", "err", err)
			}

			s.logger.ErrorContext(ctx, "failed login attempt: Invalid credentials.", "user", username)
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
	default:
		s.renderError(r, w, http.StatusBadRequest, "Unsupported request method.")
	}
}

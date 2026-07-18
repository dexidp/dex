package authflow

// callback.go implements the connector callback mechanism: the return leg of
// redirect-based connectors (OAuth2 callback and SAML POST binding).

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

func (h *Handler) handleConnectorCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var authID string
	switch r.Method {
	case http.MethodGet: // OAuth2 callback
		if authID = r.URL.Query().Get("state"); authID == "" {
			h.RenderError(r, w, http.StatusBadRequest, "User session error.")
			return
		}
	case http.MethodPost: // SAML POST binding
		if authID = r.PostFormValue("RelayState"); authID == "" {
			h.RenderError(r, w, http.StatusBadRequest, "User session error.")
			return
		}
	default:
		h.RenderError(r, w, http.StatusBadRequest, "Method not supported")
		return
	}

	authReq, err := h.storage.GetAuthRequest(ctx, authID)
	if err != nil {
		if err == storage.ErrNotFound {
			h.logger.ErrorContext(r.Context(), "invalid 'state' parameter provided", "err", err)
			h.RenderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
			return
		}
		h.logger.ErrorContext(r.Context(), "failed to get auth request", "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}

	connID, err := url.PathUnescape(mux.Vars(r)["connector"])
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to get connector", "connector_id", authReq.ConnectorID, "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	} else if connID != "" && connID != authReq.ConnectorID {
		h.logger.ErrorContext(r.Context(), "connector mismatch: callback triggered for different connector than authentication start", "authentication_start_connector_id", authReq.ConnectorID, "connector_id", connID)
		h.RenderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	conn, err := h.connectors.Get(ctx, authReq.ConnectorID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to get connector", "connector_id", authReq.ConnectorID, "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	var identity connector.Identity
	switch conn := conn.Connector.(type) {
	case connector.CallbackConnector:
		if r.Method != http.MethodGet {
			h.logger.ErrorContext(r.Context(), "SAML request mapped to OAuth2 connector")
			h.RenderError(r, w, http.StatusBadRequest, "Invalid request")
			return
		}
		identity, err = conn.HandleCallback(tokens.ParseScopes(authReq.Scopes), authReq.ConnectorData, r)
	case connector.SAMLConnector:
		if r.Method != http.MethodPost {
			h.logger.ErrorContext(r.Context(), "OAuth2 request mapped to SAML connector")
			h.RenderError(r, w, http.StatusBadRequest, "Invalid request")
			return
		}
		identity, err = conn.HandlePOST(tokens.ParseScopes(authReq.Scopes), r.PostFormValue("SAMLResponse"), authReq.ID)
	default:
		h.RenderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to authenticate", "err", err)
		var groupsErr *connector.UserNotInRequiredGroupsError
		if errors.As(err, &groupsErr) {
			h.RenderError(r, w, http.StatusForbidden, ErrMsgNotInRequiredGroups)
		} else {
			h.RenderError(r, w, http.StatusInternalServerError, ErrMsgAuthenticationFailed)
		}
		return
	}

	authReq, err = h.finalizeLogin(ctx, identity, authReq, conn.Connector)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to finalize login", "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, "Login error.")
		return
	}

	// Connector callbacks don't render the remember_me checkbox, so we use the server default.
	// The password login handler reads r.FormValue("remember_me") from the submitted form instead.
	if err := h.sessions.CreateOrUpdateAuthSession(ctx, r, w, authReq, h.sessions.DefaultRememberMe()); err != nil {
		h.logger.ErrorContext(ctx, "failed to create/update auth session", "err", err)
	}

	http.Redirect(w, r, h.buildContinueURL(authReq), http.StatusSeeOther)
}

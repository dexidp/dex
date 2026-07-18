package authflow

// authorize.go handles the /auth authorization endpoint: parsing the request,
// selecting a connector, and the browser-facing (HTML/redirect) error surface.

import (
	"context"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	conns "github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/storage"
)

// grantTypeFromAuthRequest determines the grant type from the authorization request parameters.
func (h *Handler) grantTypeFromAuthRequest(r *http.Request) string {
	redirectURI := r.Form.Get("redirect_uri")
	if redirectURI == oauth2.DeviceCallbackURI || strings.HasSuffix(redirectURI, oauth2.DeviceCallbackURI) {
		return oauth2.GrantTypeDeviceCode
	}
	responseType := r.Form.Get("response_type")
	for _, rt := range strings.Fields(responseType) {
		if rt == "token" || rt == "id_token" {
			return oauth2.GrantTypeImplicit
		}
	}
	return oauth2.GrantTypeAuthorizationCode
}

// handleAuthorization handles the OAuth2 auth endpoint. It is both the entry and
// the exit of the flow: a fresh request starts login, while a request carrying an
// auth-request id (req) is the consent step returning to issue the response —
// issuance is the authorize endpoint's own job.
func (h *Handler) handleAuthorization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Extract the arguments
	if err := r.ParseForm(); err != nil {
		h.logger.ErrorContext(r.Context(), "failed to parse arguments", "err", err)

		h.RenderError(r, w, http.StatusBadRequest, ErrMsgInvalidRequest)
		return
	}

	// A request with an auth-request id is the completed flow returning to issue.
	if r.Form.Get("req") != "" {
		h.handleIssue(w, r)
		return
	}

	connectorID := r.Form.Get("connector_id")
	allConnectors, err := h.storage.ListConnectors(ctx)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to get list of connectors", "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, "Failed to retrieve connector list.")
		return
	}

	// Determine the grant type from the authorization request to filter connectors.
	grantType := h.grantTypeFromAuthRequest(r)
	connectors := make([]storage.Connector, 0, len(allConnectors))
	for _, c := range allConnectors {
		if conns.GrantTypeAllowed(c.GrantTypes, grantType) {
			connectors = append(connectors, c)
		}
	}

	// Filter connectors based on the client's allowed connectors list.
	// client_id is required per RFC 6749 §4.1.1.
	client, authErr := h.getClientWithAuthError(ctx, r.Form.Get("client_id"))
	if authErr != nil {
		h.RenderError(r, w, authErr.Status, authErr.Error())
		return
	}
	connectors = conns.Filter(connectors, client.AllowedConnectors)

	if len(connectors) == 0 {
		h.RenderError(r, w, http.StatusBadRequest, "No connectors available for this client.")
		return
	}

	// We don't need connector_id any more
	r.Form.Del("connector_id")

	// Construct a URL with all of the arguments in its query
	connURL := url.URL{
		RawQuery: r.Form.Encode(),
	}

	// Redirect if a client chooses a specific connector_id
	if connectorID != "" {
		for _, c := range connectors {
			if c.ID == connectorID {
				connURL.Path = h.AbsPath("/auth", url.PathEscape(c.ID))
				http.Redirect(w, r, connURL.String(), http.StatusFound)
				return
			}
		}
		h.RenderError(r, w, http.StatusBadRequest, "Connector ID does not match a valid Connector")
		return
	}

	if len(connectors) == 1 && !h.alwaysShowLogin {
		connURL.Path = h.AbsPath("/auth", url.PathEscape(connectors[0].ID))
		http.Redirect(w, r, connURL.String(), http.StatusFound)
		return
	}

	// Skip connector selection if a valid session exists, unless prompt=select_account or alwaysShowLogin.
	if h.sessions.Enabled() {
		authReq, _, err := h.parseAuthorizationRequest(r)
		if err != nil {
			h.logger.ErrorContext(r.Context(), "failed to parse authorization request", "err", err)

			switch authErr := err.(type) {
			case *redirectedAuthErr:
				authErr.Handler().ServeHTTP(w, r)
			case *displayedAuthErr:
				h.RenderError(r, w, authErr.Status, err.Error())
			default:
				panic("unsupported error type")
			}
			return
		}
		prompt, err := oauth2.ParsePrompt(authReq.Prompt)
		if err != nil {
			// Server error because authReq was validated before saving it to database.
			h.redirectWithError(w, r, authReq, oauth2.ServerError, "Invalid authentication request")
			return
		}

		// Invalid prompts will be validated and properly redirected later
		if !h.alwaysShowLogin && !prompt.SelectAccount() {
			session := h.sessions.ValidSession(ctx, w, r)
			if session != nil {
				for _, c := range connectors {
					if c.ID != session.ConnectorID {
						continue
					}
					connURL.Path = h.AbsPath("/auth", url.PathEscape(session.ConnectorID))
					http.Redirect(w, r, connURL.String(), http.StatusFound)
					return
				}
			}
		}
		if prompt.None() {
			// Cannot authenticate silently with prompt=none.
			h.redirectWithError(w, r, authReq, oauth2.LoginRequired, "id_token_hint does not match authenticated user")
			return
		}
	}

	connectorInfos := make([]templates.ConnectorInfo, 0, len(connectors))
	for _, conn := range connectors {
		connURL.Path = h.AbsPath("/auth", url.PathEscape(conn.ID))
		connectorInfos = append(connectorInfos, templates.ConnectorInfo{
			ID:   conn.ID,
			Name: conn.Name,
			Type: conn.Type,
			URL:  template.URL(connURL.String()),
		})
	}

	if err := h.templates.Login(r, w, connectorInfos); err != nil {
		h.logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

// getClientWithAuthError retrieves a client by ID and returns a displayedAuthErr on failure.
// Invalid client_id is not treated as a redirect error per RFC 6749 §4.1.2.1.
// https://datatracker.ietf.org/doc/html/rfc6749#section-4.1.2.1
func (h *Handler) getClientWithAuthError(ctx context.Context, clientID string) (storage.Client, *displayedAuthErr) {
	client, err := h.storage.GetClient(ctx, clientID)
	if err != nil {
		if err == storage.ErrNotFound {
			h.logger.ErrorContext(ctx, "invalid client_id provided", "client_id", clientID)
			return storage.Client{}, newDisplayedErr(http.StatusBadRequest, "Invalid client_id provided.")
		}
		h.logger.ErrorContext(ctx, "failed to get client", "client_id", clientID, "err", err)
		return storage.Client{}, newDisplayedErr(http.StatusInternalServerError, "Database error.")
	}
	return client, nil
}

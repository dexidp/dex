package authflow

// login.go is the login entry point: it validates the chosen connector against
// the client and OIDC prompt/session rules, then kicks off that connector's
// mechanism (redirect for OAuth2/SAML, the password form for password
// connectors).

import (
	"fmt"
	"maps"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

func (h *Handler) handleConnectorLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authReq, hintSubject, err := h.parseAuthorizationRequest(r)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to parse authorization request", "err", err)

		switch authErr := err.(type) {
		case *redirectedAuthErr:
			authErr.Handler().ServeHTTP(w, r)
		case *displayedAuthErr:
			h.renderError(r, w, authErr.Status, err.Error())
		default:
			panic("unsupported error type")
		}

		return
	}

	connID, err := url.PathUnescape(mux.Vars(r)["connector"])
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to parse connector", "err", err)
		h.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist")
		return
	}

	// Validate that the connector is allowed for this client.
	client, authErr := h.getClientWithAuthError(ctx, authReq.ClientID)
	if authErr != nil {
		h.renderError(r, w, authErr.Status, authErr.Error())
		return
	}
	if !connectors.ConnectorAllowed(client.AllowedConnectors, connID) {
		h.logger.ErrorContext(r.Context(), "connector not allowed for client",
			"connector_id", connID, "client_id", authReq.ClientID)
		h.renderError(r, w, http.StatusForbidden, "Connector not allowed for this client.")
		return
	}

	conn, err := h.connectors.Get(ctx, connID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "Failed to get connector", "err", err)
		h.renderError(r, w, http.StatusBadRequest, "Connector failed to initialize")
		return
	}

	// Check if the connector allows the requested grant type.
	grantType := h.grantTypeFromAuthRequest(r)
	if !connectors.GrantTypeAllowed(conn.GrantTypes, grantType) {
		h.logger.ErrorContext(r.Context(), "connector does not allow requested grant type",
			"connector_id", connID, "grant_type", grantType)
		h.renderError(r, w, http.StatusBadRequest, "Requested connector does not support this grant type.")
		return
	}

	// Set the connector being used for the login.
	if authReq.ConnectorID != "" && authReq.ConnectorID != connID {
		h.logger.ErrorContext(r.Context(), "mismatched connector ID in auth request",
			"auth_request_connector_id", authReq.ConnectorID, "connector_id", connID)
		h.renderError(r, w, http.StatusBadRequest, "Bad connector ID")
		return
	}

	authReq.ConnectorID = connID

	// Actually create the auth request
	authReq.Expiry = h.now().Add(h.authRequestsValidFor)
	if err := h.storage.CreateAuthRequest(ctx, *authReq); err != nil {
		h.logger.ErrorContext(r.Context(), "failed to create authorization request", "err", err)
		h.renderError(r, w, http.StatusInternalServerError, "Failed to connect to the database.")
		return
	}

	// Handle OIDC prompt parameter and session-based login.
	prompt, err := oauth2.ParsePrompt(authReq.Prompt)
	if err != nil {
		// Server error because authReq was validated before saving it to database.
		h.redirectWithError(w, r, authReq, oauth2.ServerError, "Invalid authentication request")
		return
	}
	// handle prompt only if sessions are enabled
	if h.sessions.Enabled() {
		// Retrieve the session once for use in both hint and prompt logic.
		session := h.sessions.ValidAuthSession(ctx, w, r, authReq)

		// id_token_hint logic (OIDC Core 1.0 3.1.2.1):
		// When a hint is provided, verify that the session user matches.
		if hintSubject != "" {
			if !sessionMatchesHint(session, hintSubject) {
				// Clear the session if the user is different from the hint.
				session = nil
			}
			if session == nil && prompt.None() {
				// Cannot authenticate silently with prompt=none.
				h.redirectWithError(w, r, authReq, oauth2.LoginRequired, "id_token_hint does not match authenticated user")
				return
			}
		}

		// prompt=none: no UI allowed.
		if prompt.None() {
			// prompt=none: no UI allowed. advance reports interaction_required if the
			// session login can't complete silently; a missing session is login_required.
			if !h.trySessionLoginWithSession(ctx, r, w, authReq, session) {
				h.redirectWithError(w, r, authReq, oauth2.LoginRequired, "User not authenticated")
			}
			return
		}

		if !prompt.Login() {
			// Normal flow: try session-based login (skip if prompt=login forces re-auth).
			if h.trySessionLoginWithSession(ctx, r, w, authReq, session) {
				return
			}
		}
	}

	scopes := tokens.ParseScopes(authReq.Scopes)

	// Work out where the "Select another login method" link should go.
	// Include prompt=select_account so that handleAuthorization skips
	// session-based connector reuse and shows the connector list.
	backLink := ""
	if h.connectors.Len() > 1 {
		backLinkParams := make(url.Values)
		maps.Copy(backLinkParams, r.Form)
		if h.sessions.Enabled() {
			backLinkParams.Set("prompt", "select_account")
		}
		backLinkURL := url.URL{
			Path:     h.absPath("/auth"),
			RawQuery: backLinkParams.Encode(),
		}
		backLink = backLinkURL.String()
	}

	switch r.Method {
	case http.MethodGet:
		switch conn := conn.Connector.(type) {
		case connector.CallbackConnector:
			// Use the auth request ID as the "state" token.
			//
			// TODO(ericchiang): Is this appropriate or should we also be using a nonce?
			callbackURL, connData, err := conn.LoginURL(scopes, h.absURL("/callback"), authReq.ID)
			if err != nil {
				h.logger.ErrorContext(r.Context(), "connector returned error when creating callback", "connector_id", connID, "err", err)
				h.renderError(r, w, http.StatusInternalServerError, "Login error.")
				return
			}
			if len(connData) > 0 {
				updater := func(a storage.AuthRequest) (storage.AuthRequest, error) {
					a.ConnectorData = connData
					return a, nil
				}
				err := h.storage.UpdateAuthRequest(ctx, authReq.ID, updater)
				if err != nil {
					h.logger.ErrorContext(r.Context(), "Failed to set connector data on auth request", "connector_id", connID, "err", err)
					h.renderError(r, w, http.StatusInternalServerError, "Database error.")
					return
				}
			}
			http.Redirect(w, r, callbackURL, http.StatusFound)
		case connector.PasswordConnector:
			loginURL := url.URL{
				Path: h.absPath("/auth", connID, "login"),
			}
			q := loginURL.Query()
			q.Set("state", authReq.ID)
			q.Set("back", backLink)
			loginURL.RawQuery = q.Encode()

			http.Redirect(w, r, loginURL.String(), http.StatusFound)
		case connector.SAMLConnector:
			action, value, err := conn.POSTData(scopes, authReq.ID)
			if err != nil {
				h.logger.ErrorContext(r.Context(), "creating SAML data", "err", err)
				h.renderError(r, w, http.StatusInternalServerError, "Connector Login Error")
				return
			}

			// TODO(ericchiang): Don't inline this.
			fmt.Fprintf(w, `<!DOCTYPE html>
			  <html lang="en">
			  <head>
			    <meta http-equiv="content-type" content="text/html; charset=utf-8">
			    <title>SAML login</title>
			  </head>
			  <body>
			    <form method="post" action="%s" >
				    <input type="hidden" name="SAMLRequest" value="%s" />
				    <input type="hidden" name="RelayState" value="%s" />
			    </form>
				<script>
				    document.forms[0].submit();
				</script>
			  </body>
			  </html>`, action, value, authReq.ID)
		default:
			h.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
		}
	default:
		h.renderError(r, w, http.StatusBadRequest, "Unsupported request method.")
	}
}

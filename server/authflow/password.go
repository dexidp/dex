package authflow

// password.go implements the password-credential login mechanism: the login
// form and the credential check for password connectors.

import (
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

func (h *Handler) handlePasswordLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authID := r.URL.Query().Get("state")
	if authID == "" {
		h.RenderError(r, w, http.StatusBadRequest, "User session error.")
		return
	}

	backLink := r.URL.Query().Get("back")

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
		h.logger.ErrorContext(r.Context(), "failed to parse connector", "err", err)
		h.RenderError(r, w, http.StatusBadRequest, "Requested resource does not exist")
		return
	} else if connID != "" && connID != authReq.ConnectorID {
		h.logger.ErrorContext(r.Context(), "connector mismatch: password login triggered for different connector from authentication start", "start_connector_id", authReq.ConnectorID, "password_connector_id", connID)
		h.RenderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	conn, err := h.connectors.Get(ctx, authReq.ConnectorID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to get connector", "connector_id", authReq.ConnectorID, "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, "Connector failed to initialize.")
		return
	}

	pwConn, ok := conn.Connector.(connector.PasswordConnector)
	if !ok {
		h.logger.ErrorContext(r.Context(), "expected password connector in handlePasswordLogin()", "password_connector", pwConn)
		h.RenderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	rememberMe := h.sessions.RememberMeDefault()

	switch r.Method {
	case http.MethodGet:
		// Before rendering the password form, allow connectors that support SPNEGO to try Kerberos auth.
		if sp, ok := pwConn.(connector.SPNEGOAware); ok {
			scopes := tokens.ParseScopes(authReq.Scopes)
			if ident, handled, err := sp.TrySPNEGO(ctx, scopes, w, r); bool(handled) {
				if err != nil {
					// SPNEGO handled the request but reported an error (e.g., LDAP lookup failed
					// after successful Kerberos auth). Log error details, show generic message to user.
					h.logger.ErrorContext(ctx, "SPNEGO authentication error", "err", err)
					h.RenderError(r, w, http.StatusUnauthorized, ErrMsgAuthenticationFailed)
					return
				}
				if ident != nil {
					authReq, err = h.finalizeLogin(ctx, *ident, authReq, conn.Connector)
					if err != nil {
						h.logger.ErrorContext(ctx, "failed to finalize login", "err", err)
						h.RenderError(r, w, http.StatusInternalServerError, "Login error.")
						return
					}
					http.Redirect(w, r, h.BuildMFAURL(authReq), http.StatusSeeOther)
					return
				}
				// handled with no identity typically means the SPNEGO middleware
				// wrote its own 401 (bare challenge, continuation, or reject); do
				// not render the password form on top of it.
				return
			}
		}
		if err := h.templates.Password(r, w, r.URL.String(), "", usernamePrompt(pwConn), false, backLink, rememberMe); err != nil {
			h.logger.ErrorContext(r.Context(), "server template error", "err", err)
		}
	case http.MethodPost:
		username := r.FormValue("login")
		password := r.FormValue("password")
		scopes := tokens.ParseScopes(authReq.Scopes)

		identity, ok, err := pwConn.Login(r.Context(), scopes, username, password)
		if err != nil {
			h.logger.ErrorContext(r.Context(), "failed to login user", "err", err)
			h.RenderError(r, w, http.StatusInternalServerError, ErrMsgLoginError)
			return
		}
		if !ok {
			if err := h.templates.Password(r, w, r.URL.String(), username, usernamePrompt(pwConn), true, backLink, rememberMe); err != nil {
				h.logger.ErrorContext(r.Context(), "server template error", "err", err)
			}
			h.logger.ErrorContext(r.Context(), "failed login attempt: Invalid credentials.", "user", username)
			return
		}
		authReq, err = h.finalizeLogin(r.Context(), identity, authReq, conn.Connector)
		if err != nil {
			h.logger.ErrorContext(r.Context(), "failed to finalize login", "err", err)
			h.RenderError(r, w, http.StatusInternalServerError, "Login error.")
			return
		}

		rememberMe := r.FormValue("remember_me") == "on"
		if err := h.sessions.CreateOrUpdateAuthSession(ctx, r, w, authReq, rememberMe); err != nil {
			h.logger.ErrorContext(ctx, "failed to create/update auth session", "err", err)
		}

		http.Redirect(w, r, h.BuildMFAURL(authReq), http.StatusSeeOther)
	default:
		h.RenderError(r, w, http.StatusBadRequest, "Unsupported request method.")
	}
}

// Check for username prompt override from connector. Defaults to "Username".
func usernamePrompt(conn connector.PasswordConnector) string {
	if attr := conn.Prompt(); attr != "" {
		return attr
	}
	return "Username"
}

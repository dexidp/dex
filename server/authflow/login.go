package authflow

// login.go implements the browser login flow: connector login, the password
// login form, the connector callback, and login finalization.

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/mux"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/featureflags"
	"github.com/dexidp/dex/server/authflow/authreq"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

func (h *Handler) handleConnectorLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authReq, hintSubject, err := h.req.Parse(r)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to parse authorization request", "err", err)

		switch authErr := err.(type) {
		case *authreq.RedirectedErr:
			authErr.Handler().ServeHTTP(w, r)
		case *authreq.DisplayedErr:
			h.RenderError(r, w, authErr.Status, err.Error())
		default:
			panic("unsupported error type")
		}

		return
	}

	connID, err := url.PathUnescape(mux.Vars(r)["connector"])
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to parse connector", "err", err)
		h.RenderError(r, w, http.StatusBadRequest, "Requested resource does not exist")
		return
	}

	// Validate that the connector is allowed for this client.
	client, authErr := h.getClientWithAuthError(ctx, authReq.ClientID)
	if authErr != nil {
		h.RenderError(r, w, authErr.Status, authErr.Error())
		return
	}
	if !connectors.ConnectorAllowed(client.AllowedConnectors, connID) {
		h.logger.ErrorContext(r.Context(), "connector not allowed for client",
			"connector_id", connID, "client_id", authReq.ClientID)
		h.RenderError(r, w, http.StatusForbidden, "Connector not allowed for this client.")
		return
	}

	conn, err := h.connectors.Get(ctx, connID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "Failed to get connector", "err", err)
		h.RenderError(r, w, http.StatusBadRequest, "Connector failed to initialize")
		return
	}

	// Check if the connector allows the requested grant type.
	grantType := h.grantTypeFromAuthRequest(r)
	if !connectors.GrantTypeAllowed(conn.GrantTypes, grantType) {
		h.logger.ErrorContext(r.Context(), "connector does not allow requested grant type",
			"connector_id", connID, "grant_type", grantType)
		h.RenderError(r, w, http.StatusBadRequest, "Requested connector does not support this grant type.")
		return
	}

	// Set the connector being used for the login.
	if authReq.ConnectorID != "" && authReq.ConnectorID != connID {
		h.logger.ErrorContext(r.Context(), "mismatched connector ID in auth request",
			"auth_request_connector_id", authReq.ConnectorID, "connector_id", connID)
		h.RenderError(r, w, http.StatusBadRequest, "Bad connector ID")
		return
	}

	authReq.ConnectorID = connID

	// Actually create the auth request
	authReq.Expiry = h.now().Add(h.authRequestsValidFor)
	if err := h.storage.CreateAuthRequest(ctx, *authReq); err != nil {
		h.logger.ErrorContext(r.Context(), "failed to create authorization request", "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, "Failed to connect to the database.")
		return
	}

	// Handle OIDC prompt parameter and session-based login.
	prompt, err := oauth2.ParsePrompt(authReq.Prompt)
	if err != nil {
		// Server error because authReq was validated before saving it to database.
		authreq.RedirectWithError(w, r, authReq, oauth2.ServerError, "Invalid authentication request")
		return
	}
	// handle prompt only if sessions are enabled
	if h.sessions.Enabled() {
		// Retrieve the session once for use in both hint and prompt logic.
		session := h.sessions.ValidAuthSession(ctx, w, r, authReq)

		// id_token_hint logic (OIDC Core 1.0 3.1.2.1):
		// When a hint is provided, verify that the session user matches.
		if hintSubject != "" {
			if !authreq.SessionMatchesHint(session, hintSubject) {
				// Clear the session if the user is different from the hint.
				session = nil
			}
			if session == nil && prompt.None() {
				// Cannot authenticate silently with prompt=none.
				authreq.RedirectWithError(w, r, authReq, oauth2.LoginRequired, "id_token_hint does not match authenticated user")
				return
			}
		}

		// prompt=none: no UI allowed.
		if prompt.None() {
			redirectURL, ok := h.trySessionLoginWithSession(ctx, r, w, authReq, session)
			if !ok {
				authreq.RedirectWithError(w, r, authReq, oauth2.LoginRequired, "User not authenticated")
				return
			}
			if redirectURL != "" {
				// Session found but user interaction is needed (consent or MFA) — no UI allowed.
				authreq.RedirectWithError(w, r, authReq, oauth2.InteractionRequired, "User interaction required")
				return
			}
			return
		}

		if !prompt.Login() {
			// Normal flow: try session-based login (skip if prompt=login forces re-auth).
			if redirectURL, ok := h.trySessionLoginWithSession(ctx, r, w, authReq, session); ok {
				if redirectURL != "" {
					http.Redirect(w, r, redirectURL, http.StatusSeeOther)
				}
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
			Path:     h.AbsPath("/auth"),
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
			callbackURL, connData, err := conn.LoginURL(scopes, h.AbsURL("/callback"), authReq.ID)
			if err != nil {
				h.logger.ErrorContext(r.Context(), "connector returned error when creating callback", "connector_id", connID, "err", err)
				h.RenderError(r, w, http.StatusInternalServerError, "Login error.")
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
					h.RenderError(r, w, http.StatusInternalServerError, "Database error.")
					return
				}
			}
			http.Redirect(w, r, callbackURL, http.StatusFound)
		case connector.PasswordConnector:
			loginURL := url.URL{
				Path: h.AbsPath("/auth", connID, "login"),
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
				h.RenderError(r, w, http.StatusInternalServerError, "Connector Login Error")
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
			h.RenderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
		}
	default:
		h.RenderError(r, w, http.StatusBadRequest, "Unsupported request method.")
	}
}

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
					redirectURL, canSkipApproval, err := h.finalizeLogin(ctx, *ident, authReq, conn.Connector)
					if err != nil {
						h.logger.ErrorContext(ctx, "failed to finalize login", "err", err)
						h.RenderError(r, w, http.StatusInternalServerError, "Login error.")
						return
					}

					if canSkipApproval {
						authReq, err = h.storage.GetAuthRequest(ctx, authReq.ID)
						if err != nil {
							h.logger.ErrorContext(ctx, "failed to get finalized auth request", "err", err)
							h.RenderError(r, w, http.StatusInternalServerError, "Login error.")
							return
						}
						h.authcode.Send(w, r, authReq)
						return
					}

					http.Redirect(w, r, redirectURL, http.StatusSeeOther)
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
		redirectURL, canSkipApproval, err := h.finalizeLogin(r.Context(), identity, authReq, conn.Connector)
		if err != nil {
			h.logger.ErrorContext(r.Context(), "failed to finalize login", "err", err)
			h.RenderError(r, w, http.StatusInternalServerError, "Login error.")
			return
		}

		// Re-read auth request after finalizeLogin populated Claims.
		authReq, err = h.storage.GetAuthRequest(ctx, authReq.ID)
		if err != nil {
			h.logger.ErrorContext(r.Context(), "failed to get finalized auth request", "err", err)
			h.RenderError(r, w, http.StatusInternalServerError, "Login error.")
			return
		}

		rememberMe := r.FormValue("remember_me") == "on"
		if err := h.sessions.CreateOrUpdateAuthSession(ctx, r, w, authReq, rememberMe); err != nil {
			h.logger.ErrorContext(ctx, "failed to create/update auth session", "err", err)
		}

		if canSkipApproval {
			// authReq was already re-read after finalizeLogin above.
			h.authcode.Send(w, r, authReq)
			return
		}

		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	default:
		h.RenderError(r, w, http.StatusBadRequest, "Unsupported request method.")
	}
}

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

	redirectURL, canSkipApproval, err := h.finalizeLogin(ctx, identity, authReq, conn.Connector)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to finalize login", "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, "Login error.")
		return
	}

	// Re-read auth request after finalizeLogin populated Claims.
	authReq, err = h.storage.GetAuthRequest(ctx, authReq.ID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to get finalized auth request", "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, "Login error.")
		return
	}

	// Connector callbacks don't render the remember_me checkbox, so we use the server default.
	// The password login handler reads r.FormValue("remember_me") from the submitted form instead.
	if err := h.sessions.CreateOrUpdateAuthSession(ctx, r, w, authReq, h.sessions.DefaultRememberMe()); err != nil {
		h.logger.ErrorContext(ctx, "failed to create/update auth session", "err", err)
	}

	if canSkipApproval {
		// authReq was already re-read after finalizeLogin above.
		h.authcode.Send(w, r, authReq)
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// finalizeLogin associates the user's identity with the current AuthRequest, then returns
// the approval page's path.
func (h *Handler) finalizeLogin(ctx context.Context, identity connector.Identity, authReq storage.AuthRequest, conn connector.Connector) (string, bool, error) {
	// Refuse to complete login for a locked account. BlockedUntil lives on the
	// persisted UserIdentity, which only exists when the sessions feature is on;
	// a first-time login (no stored identity yet) cannot be blocked.
	if featureflags.SessionsEnabled.Enabled() {
		storedIdentity, err := h.storage.GetUserIdentity(ctx, identity.UserID, authReq.ConnectorID)
		switch {
		case err == nil:
			if !storedIdentity.BlockedUntil.IsZero() && h.now().Before(storedIdentity.BlockedUntil) {
				h.logger.WarnContext(ctx, "login rejected for locked account",
					"connector_id", authReq.ConnectorID, "user_id", identity.UserID, "blocked_until", storedIdentity.BlockedUntil)
				return "", false, fmt.Errorf("account is locked until %s", storedIdentity.BlockedUntil.Format(time.RFC3339))
			}
		case !errors.Is(err, storage.ErrNotFound):
			return "", false, fmt.Errorf("failed to look up user identity: %w", err)
		}
	}

	claims := storage.Claims{
		UserID:            identity.UserID,
		Username:          identity.Username,
		PreferredUsername: identity.PreferredUsername,
		Email:             identity.Email,
		EmailVerified:     identity.EmailVerified,
		Groups:            identity.Groups,
	}

	updater := func(a storage.AuthRequest) (storage.AuthRequest, error) {
		a.LoggedIn = true
		a.Claims = claims
		a.ConnectorData = identity.ConnectorData
		a.AuthTime = h.now()
		return a, nil
	}
	if err := h.storage.UpdateAuthRequest(ctx, authReq.ID, updater); err != nil {
		return "", false, fmt.Errorf("failed to update auth request: %v", err)
	}
	// Keep the in-memory copy in sync with what was persisted so later reads
	// (the next-step decision below) see the identity we just stored.
	authReq, _ = updater(authReq)

	email := claims.Email
	if !claims.EmailVerified {
		email += " (unverified)"
	}

	h.logger.InfoContext(ctx, "login successful",
		"connector_id", authReq.ConnectorID, "user_id", claims.UserID,
		"username", claims.Username, "preferred_username", claims.PreferredUsername,
		"email", email, "groups", claims.Groups)

	offlineAccessRequested := false
	for _, scope := range authReq.Scopes {
		if scope == tokens.ScopeOfflineAccess {
			offlineAccessRequested = true
			break
		}
	}
	_, canRefresh := conn.(connector.RefreshConnector)

	if offlineAccessRequested && canRefresh {
		// Try to retrieve an existing OfflineSession object for the corresponding user.
		session, err := h.storage.GetOfflineSessions(ctx, identity.UserID, authReq.ConnectorID)
		switch {
		case err != nil && err == storage.ErrNotFound:
			offlineSessions := storage.OfflineSessions{
				UserID:        identity.UserID,
				ConnID:        authReq.ConnectorID,
				Refresh:       make(map[string]*storage.RefreshTokenRef),
				ConnectorData: identity.ConnectorData,
			}

			// Create a new OfflineSession object for the user and add a reference object for
			// the newly received refreshtoken.
			if err := h.storage.CreateOfflineSessions(ctx, offlineSessions); err != nil {
				h.logger.ErrorContext(ctx, "failed to create offline session", "err", err)
				return "", false, err
			}
		case err == nil:
			// Update existing OfflineSession obj with new RefreshTokenRef.
			if err := h.storage.UpdateOfflineSessions(ctx, session.UserID, session.ConnID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
				if len(identity.ConnectorData) > 0 {
					old.ConnectorData = identity.ConnectorData
				}
				return old, nil
			}); err != nil {
				h.logger.ErrorContext(ctx, "failed to update offline session", "err", err)
				return "", false, err
			}
		default:
			h.logger.ErrorContext(ctx, "failed to get offline session", "err", err)
			return "", false, err
		}
	}

	// Create or update UserIdentity to persist user claims across sessions.
	if featureflags.SessionsEnabled.Enabled() {
		now := h.now()

		_, err := h.storage.GetUserIdentity(ctx, identity.UserID, authReq.ConnectorID)
		switch {
		case err != nil && errors.Is(err, storage.ErrNotFound):
			ui := storage.UserIdentity{
				UserID:      identity.UserID,
				ConnectorID: authReq.ConnectorID,
				Claims:      claims,
				Consents:    make(map[string][]string),
				CreatedAt:   now,
				LastLogin:   now,
			}
			if err := h.storage.CreateUserIdentity(ctx, ui); err != nil {
				h.logger.ErrorContext(ctx, "failed to create user identity", "err", err)
				return "", false, err
			}
		case err == nil:
			if err := h.storage.UpdateUserIdentity(ctx, identity.UserID, authReq.ConnectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
				old.Claims = claims
				old.LastLogin = now
				return old, nil
			}); err != nil {
				h.logger.ErrorContext(ctx, "failed to update user identity", "err", err)
				return "", false, err
			}
		default:
			h.logger.ErrorContext(ctx, "failed to get user identity", "err", err)
			return "", false, err
		}
	}

	step, err := h.nextAuthStep(ctx, &authReq)
	if err != nil {
		return "", false, fmt.Errorf("failed to determine next auth step: %v", err)
	}
	if mfa, ok := step.(mfaStep); ok {
		return h.mfa.BuildRedirectURL(authReq, mfa.authenticator), false, nil
	}

	// MFA is satisfied — record it so a later /approval re-entry doesn't re-check.
	if err := h.storage.UpdateAuthRequest(ctx, authReq.ID, func(a storage.AuthRequest) (storage.AuthRequest, error) {
		a.MFAValidated = true
		return a, nil
	}); err != nil {
		return "", false, fmt.Errorf("failed to update auth request MFA status: %v", err)
	}

	if _, ok := step.(issueStep); ok {
		return "", true, nil
	}
	return h.BuildApprovalURL(authReq), false, nil
}

// Check for username prompt override from connector. Defaults to "Username".
func usernamePrompt(conn connector.PasswordConnector) string {
	if attr := conn.Prompt(); attr != "" {
		return attr
	}
	return "Username"
}

package server

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/featureflags"
	"github.com/dexidp/dex/storage"
	"github.com/gorilla/mux"
)

func (s *Server) handleConnectorLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authReq, hintSubject, err := s.parseAuthorizationRequest(r)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to parse authorization request", "err", err)

		switch authErr := err.(type) {
		case *redirectedAuthErr:
			authErr.Handler().ServeHTTP(w, r)
		case *displayedAuthErr:
			s.renderError(r, w, authErr.Status, err.Error())
		default:
			panic("unsupported error type")
		}

		return
	}

	connID, err := url.PathUnescape(mux.Vars(r)["connector"])
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to parse connector", "err", err)
		s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist")
		return
	}

	// Validate that the connector is allowed for this client.
	client, authErr := s.getClientWithAuthError(ctx, authReq.ClientID)
	if authErr != nil {
		s.renderError(r, w, authErr.Status, authErr.Error())
		return
	}
	if !isConnectorAllowed(client.AllowedConnectors, connID) {
		s.logger.ErrorContext(r.Context(), "connector not allowed for client",
			"connector_id", connID, "client_id", authReq.ClientID)
		s.renderError(r, w, http.StatusForbidden, "Connector not allowed for this client.")
		return
	}

	conn, err := s.getConnector(ctx, connID)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "Failed to get connector", "err", err)
		s.renderError(r, w, http.StatusBadRequest, "Connector failed to initialize")
		return
	}

	// Check if the connector allows the requested grant type.
	grantType := s.grantTypeFromAuthRequest(r)
	if !GrantTypeAllowed(conn.GrantTypes, grantType) {
		s.logger.ErrorContext(r.Context(), "connector does not allow requested grant type",
			"connector_id", connID, "grant_type", grantType)
		s.renderError(r, w, http.StatusBadRequest, "Requested connector does not support this grant type.")
		return
	}

	// Set the connector being used for the login.
	if authReq.ConnectorID != "" && authReq.ConnectorID != connID {
		s.logger.ErrorContext(r.Context(), "mismatched connector ID in auth request",
			"auth_request_connector_id", authReq.ConnectorID, "connector_id", connID)
		s.renderError(r, w, http.StatusBadRequest, "Bad connector ID")
		return
	}

	authReq.ConnectorID = connID

	// Actually create the auth request
	authReq.Expiry = s.now().Add(s.authRequestsValidFor)
	if err := s.storage.CreateAuthRequest(ctx, *authReq); err != nil {
		s.logger.ErrorContext(r.Context(), "failed to create authorization request", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Failed to connect to the database.")
		return
	}

	// Handle OIDC prompt parameter and session-based login.
	prompt, err := ParsePrompt(authReq.Prompt)
	if err != nil {
		// Server error because authReq was validated before saving it to database.
		s.redirectWithError(w, r, authReq, errServerError, "Invalid authentication request")
		return
	}
	// handle prompt only if sessions are enabled
	if s.sessionConfig != nil {
		// Retrieve the session once for use in both hint and prompt logic.
		session := s.getValidAuthSession(ctx, w, r, authReq)

		// id_token_hint logic (OIDC Core 1.0 3.1.2.1):
		// When a hint is provided, verify that the session user matches.
		if hintSubject != "" {
			if !sessionMatchesHint(session, hintSubject) {
				// Clear the session if the user is different from the hint.
				session = nil
			}
			if session == nil && prompt.None() {
				// Cannot authenticate silently with prompt=none.
				s.redirectWithError(w, r, authReq, errLoginRequired, "id_token_hint does not match authenticated user")
				return
			}
		}

		// prompt=none: no UI allowed.
		if prompt.None() {
			redirectURL, ok := s.trySessionLoginWithSession(ctx, r, w, authReq, session)
			if !ok {
				s.redirectWithError(w, r, authReq, errLoginRequired, "User not authenticated")
				return
			}
			if redirectURL != "" {
				// Session found but user interaction is needed (consent or MFA) — no UI allowed.
				s.redirectWithError(w, r, authReq, errInteractionRequired, "User interaction required")
				return
			}
			return
		}

		if !prompt.Login() {
			// Normal flow: try session-based login (skip if prompt=login forces re-auth).
			if redirectURL, ok := s.trySessionLoginWithSession(ctx, r, w, authReq, session); ok {
				if redirectURL != "" {
					http.Redirect(w, r, redirectURL, http.StatusSeeOther)
				}
				return
			}
		}
	}

	scopes := parseScopes(authReq.Scopes)

	// Work out where the "Select another login method" link should go.
	// Include prompt=select_account so that handleAuthorization skips
	// session-based connector reuse and shows the connector list.
	backLink := ""
	if len(s.connectors) > 1 {
		backLinkParams := make(url.Values)
		maps.Copy(backLinkParams, r.Form)
		if s.sessionConfig != nil {
			backLinkParams.Set("prompt", "select_account")
		}
		backLinkURL := url.URL{
			Path:     s.absPath("/auth"),
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
			callbackURL, connData, err := conn.LoginURL(scopes, s.absURL("/callback"), authReq.ID)
			if err != nil {
				s.logger.ErrorContext(r.Context(), "connector returned error when creating callback", "connector_id", connID, "err", err)
				s.renderError(r, w, http.StatusInternalServerError, "Login error.")
				return
			}
			if len(connData) > 0 {
				updater := func(a storage.AuthRequest) (storage.AuthRequest, error) {
					a.ConnectorData = connData
					return a, nil
				}
				err := s.storage.UpdateAuthRequest(ctx, authReq.ID, updater)
				if err != nil {
					s.logger.ErrorContext(r.Context(), "Failed to set connector data on auth request", "connector_id", connID, "err", err)
					s.renderError(r, w, http.StatusInternalServerError, "Database error.")
					return
				}
			}
			http.Redirect(w, r, callbackURL, http.StatusFound)
		case connector.PasswordConnector:
			loginURL := url.URL{
				Path: s.absPath("/auth", connID, "login"),
			}
			q := loginURL.Query()
			q.Set("state", authReq.ID)
			q.Set("back", backLink)
			loginURL.RawQuery = q.Encode()

			http.Redirect(w, r, loginURL.String(), http.StatusFound)
		case connector.SAMLConnector:
			action, value, err := conn.POSTData(scopes, authReq.ID)
			if err != nil {
				s.logger.ErrorContext(r.Context(), "creating SAML data", "err", err)
				s.renderError(r, w, http.StatusInternalServerError, "Connector Login Error")
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
			s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
		}
	default:
		s.renderError(r, w, http.StatusBadRequest, "Unsupported request method.")
	}
}

func (s *Server) handlePasswordLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authID := r.URL.Query().Get("state")
	if authID == "" {
		s.renderError(r, w, http.StatusBadRequest, "User session error.")
		return
	}

	backLink := r.URL.Query().Get("back")

	authReq, err := s.storage.GetAuthRequest(ctx, authID)
	if err != nil {
		if err == storage.ErrNotFound {
			s.logger.ErrorContext(r.Context(), "invalid 'state' parameter provided", "err", err)
			s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
			return
		}
		s.logger.ErrorContext(r.Context(), "failed to get auth request", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}

	connID, err := url.PathUnescape(mux.Vars(r)["connector"])
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to parse connector", "err", err)
		s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist")
		return
	} else if connID != "" && connID != authReq.ConnectorID {
		s.logger.ErrorContext(r.Context(), "connector mismatch: password login triggered for different connector from authentication start", "start_connector_id", authReq.ConnectorID, "password_connector_id", connID)
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	conn, err := s.getConnector(ctx, authReq.ConnectorID)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to get connector", "connector_id", authReq.ConnectorID, "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Connector failed to initialize.")
		return
	}

	pwConn, ok := conn.Connector.(connector.PasswordConnector)
	if !ok {
		s.logger.ErrorContext(r.Context(), "expected password connector in handlePasswordLogin()", "password_connector", pwConn)
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	rememberMe := s.rememberMeDefault()

	switch r.Method {
	case http.MethodGet:
		// Before rendering the password form, allow connectors that support SPNEGO to try Kerberos auth.
		if sp, ok := pwConn.(connector.SPNEGOAware); ok {
			scopes := parseScopes(authReq.Scopes)
			if ident, handled, err := sp.TrySPNEGO(ctx, scopes, w, r); bool(handled) {
				if err != nil {
					// SPNEGO handled the request but reported an error (e.g., LDAP lookup failed
					// after successful Kerberos auth). Log error details, show generic message to user.
					s.logger.ErrorContext(ctx, "SPNEGO authentication error", "err", err)
					s.renderError(r, w, http.StatusUnauthorized, ErrMsgAuthenticationFailed)
					return
				}
				if ident != nil {
					redirectURL, canSkipApproval, err := s.finalizeLogin(ctx, *ident, authReq, conn.Connector)
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
					return
				}
				// handled with no identity typically means the SPNEGO middleware
				// wrote its own 401 (bare challenge, continuation, or reject); do
				// not render the password form on top of it.
				return
			}
		}
		if err := s.templates.password(r, w, r.URL.String(), "", usernamePrompt(pwConn), false, backLink, rememberMe); err != nil {
			s.logger.ErrorContext(r.Context(), "server template error", "err", err)
		}
	case http.MethodPost:
		username := r.FormValue("login")
		password := r.FormValue("password")
		scopes := parseScopes(authReq.Scopes)

		identity, ok, err := pwConn.Login(r.Context(), scopes, username, password)
		if err != nil {
			s.logger.ErrorContext(r.Context(), "failed to login user", "err", err)
			s.renderError(r, w, http.StatusInternalServerError, ErrMsgLoginError)
			return
		}
		if !ok {
			if err := s.templates.password(r, w, r.URL.String(), username, usernamePrompt(pwConn), true, backLink, rememberMe); err != nil {
				s.logger.ErrorContext(r.Context(), "server template error", "err", err)
			}
			s.logger.ErrorContext(r.Context(), "failed login attempt: Invalid credentials.", "user", username)
			return
		}
		redirectURL, canSkipApproval, err := s.finalizeLogin(r.Context(), identity, authReq, conn.Connector)
		if err != nil {
			s.logger.ErrorContext(r.Context(), "failed to finalize login", "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "Login error.")
			return
		}

		// Re-read auth request after finalizeLogin populated Claims.
		authReq, err = s.storage.GetAuthRequest(ctx, authReq.ID)
		if err != nil {
			s.logger.ErrorContext(r.Context(), "failed to get finalized auth request", "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "Login error.")
			return
		}

		rememberMe := r.FormValue("remember_me") == "on"
		if err := s.createOrUpdateAuthSession(ctx, r, w, authReq, rememberMe); err != nil {
			s.logger.ErrorContext(ctx, "failed to create/update auth session", "err", err)
		}

		if canSkipApproval {
			// authReq was already re-read after finalizeLogin above.
			s.sendCodeResponse(w, r, authReq)
			return
		}

		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	default:
		s.renderError(r, w, http.StatusBadRequest, "Unsupported request method.")
	}
}

func (s *Server) handleConnectorCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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
			s.logger.ErrorContext(r.Context(), "invalid 'state' parameter provided", "err", err)
			s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
			return
		}
		s.logger.ErrorContext(r.Context(), "failed to get auth request", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}

	connID, err := url.PathUnescape(mux.Vars(r)["connector"])
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to get connector", "connector_id", authReq.ConnectorID, "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	} else if connID != "" && connID != authReq.ConnectorID {
		s.logger.ErrorContext(r.Context(), "connector mismatch: callback triggered for different connector than authentication start", "authentication_start_connector_id", authReq.ConnectorID, "connector_id", connID)
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	conn, err := s.getConnector(ctx, authReq.ConnectorID)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to get connector", "connector_id", authReq.ConnectorID, "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	var identity connector.Identity
	switch conn := conn.Connector.(type) {
	case connector.CallbackConnector:
		if r.Method != http.MethodGet {
			s.logger.ErrorContext(r.Context(), "SAML request mapped to OAuth2 connector")
			s.renderError(r, w, http.StatusBadRequest, "Invalid request")
			return
		}
		identity, err = conn.HandleCallback(parseScopes(authReq.Scopes), authReq.ConnectorData, r)
	case connector.SAMLConnector:
		if r.Method != http.MethodPost {
			s.logger.ErrorContext(r.Context(), "OAuth2 request mapped to SAML connector")
			s.renderError(r, w, http.StatusBadRequest, "Invalid request")
			return
		}
		identity, err = conn.HandlePOST(parseScopes(authReq.Scopes), r.PostFormValue("SAMLResponse"), authReq.ID)
	default:
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to authenticate", "err", err)
		var groupsErr *connector.UserNotInRequiredGroupsError
		if errors.As(err, &groupsErr) {
			s.renderError(r, w, http.StatusForbidden, ErrMsgNotInRequiredGroups)
		} else {
			s.renderError(r, w, http.StatusInternalServerError, ErrMsgAuthenticationFailed)
		}
		return
	}

	redirectURL, canSkipApproval, err := s.finalizeLogin(ctx, identity, authReq, conn.Connector)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to finalize login", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Login error.")
		return
	}

	// Re-read auth request after finalizeLogin populated Claims.
	authReq, err = s.storage.GetAuthRequest(ctx, authReq.ID)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to get finalized auth request", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Login error.")
		return
	}

	// Connector callbacks don't render the remember_me checkbox, so we use the server default.
	// The password login handler reads r.FormValue("remember_me") from the submitted form instead.
	if err := s.createOrUpdateAuthSession(ctx, r, w, authReq, s.sessionConfig != nil && s.sessionConfig.RememberMeCheckedByDefault); err != nil {
		s.logger.ErrorContext(ctx, "failed to create/update auth session", "err", err)
	}

	if canSkipApproval {
		// authReq was already re-read after finalizeLogin above.
		s.sendCodeResponse(w, r, authReq)
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// finalizeLogin associates the user's identity with the current AuthRequest, then returns
// the approval page's path.
func (s *Server) finalizeLogin(ctx context.Context, identity connector.Identity, authReq storage.AuthRequest, conn connector.Connector) (string, bool, error) {
	// Refuse to complete login for a locked account. BlockedUntil lives on the
	// persisted UserIdentity, which only exists when the sessions feature is on;
	// a first-time login (no stored identity yet) cannot be blocked.
	if featureflags.SessionsEnabled.Enabled() {
		storedIdentity, err := s.storage.GetUserIdentity(ctx, identity.UserID, authReq.ConnectorID)
		switch {
		case err == nil:
			if !storedIdentity.BlockedUntil.IsZero() && s.now().Before(storedIdentity.BlockedUntil) {
				s.logger.WarnContext(ctx, "login rejected for locked account",
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
		a.AuthTime = s.now()
		return a, nil
	}
	if err := s.storage.UpdateAuthRequest(ctx, authReq.ID, updater); err != nil {
		return "", false, fmt.Errorf("failed to update auth request: %v", err)
	}

	email := claims.Email
	if !claims.EmailVerified {
		email += " (unverified)"
	}

	s.logger.InfoContext(ctx, "login successful",
		"connector_id", authReq.ConnectorID, "user_id", claims.UserID,
		"username", claims.Username, "preferred_username", claims.PreferredUsername,
		"email", email, "groups", claims.Groups)

	offlineAccessRequested := false
	for _, scope := range authReq.Scopes {
		if scope == scopeOfflineAccess {
			offlineAccessRequested = true
			break
		}
	}
	_, canRefresh := conn.(connector.RefreshConnector)

	if offlineAccessRequested && canRefresh {
		// Try to retrieve an existing OfflineSession object for the corresponding user.
		session, err := s.storage.GetOfflineSessions(ctx, identity.UserID, authReq.ConnectorID)
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
			if err := s.storage.CreateOfflineSessions(ctx, offlineSessions); err != nil {
				s.logger.ErrorContext(ctx, "failed to create offline session", "err", err)
				return "", false, err
			}
		case err == nil:
			// Update existing OfflineSession obj with new RefreshTokenRef.
			if err := s.storage.UpdateOfflineSessions(ctx, session.UserID, session.ConnID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
				if len(identity.ConnectorData) > 0 {
					old.ConnectorData = identity.ConnectorData
				}
				return old, nil
			}); err != nil {
				s.logger.ErrorContext(ctx, "failed to update offline session", "err", err)
				return "", false, err
			}
		default:
			s.logger.ErrorContext(ctx, "failed to get offline session", "err", err)
			return "", false, err
		}
	}

	// Create or update UserIdentity to persist user claims across sessions.
	var userIdentity *storage.UserIdentity
	if featureflags.SessionsEnabled.Enabled() {
		now := s.now()

		ui, err := s.storage.GetUserIdentity(ctx, identity.UserID, authReq.ConnectorID)
		switch {
		case err != nil && errors.Is(err, storage.ErrNotFound):
			ui = storage.UserIdentity{
				UserID:      identity.UserID,
				ConnectorID: authReq.ConnectorID,
				Claims:      claims,
				Consents:    make(map[string][]string),
				CreatedAt:   now,
				LastLogin:   now,
			}
			if err := s.storage.CreateUserIdentity(ctx, ui); err != nil {
				s.logger.ErrorContext(ctx, "failed to create user identity", "err", err)
				return "", false, err
			}
		case err == nil:
			if err := s.storage.UpdateUserIdentity(ctx, identity.UserID, authReq.ConnectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
				old.Claims = claims
				old.LastLogin = now
				return old, nil
			}); err != nil {
				s.logger.ErrorContext(ctx, "failed to update user identity", "err", err)
				return "", false, err
			}
			// Update the existing UserIdentity obj with new claims to use them later in the flow.
			ui.Claims = claims
			ui.LastLogin = now
		default:
			s.logger.ErrorContext(ctx, "failed to get user identity", "err", err)
			return "", false, err
		}
		userIdentity = &ui
	}

	// Check if the client requires MFA.
	mfaChain, err := s.mfaChainForClient(ctx, authReq.ClientID, authReq.ConnectorID)
	if err != nil {
		return "", false, fmt.Errorf("failed to get MFA chain for client: %v", err)
	}
	if len(mfaChain) > 0 {
		return s.buildMFARedirectURL(authReq, mfaChain[0]), false, nil
	}

	// No MFA required — mark as validated.
	if err := s.storage.UpdateAuthRequest(ctx, authReq.ID, func(a storage.AuthRequest) (storage.AuthRequest, error) {
		a.MFAValidated = true
		return a, nil
	}); err != nil {
		return "", false, fmt.Errorf("failed to update auth request MFA status: %v", err)
	}

	// Skip approval if globally configured.
	if s.skipApproval && !authReq.ForceApprovalPrompt {
		return "", true, nil
	}

	// Skip approval if user already consented to the requested scopes for this client.
	if !authReq.ForceApprovalPrompt && userIdentity != nil {
		if scopesCoveredByConsent(userIdentity.Consents[authReq.ClientID], authReq.Scopes) {
			return "", true, nil
		}
	}

	return s.buildApprovalURL(authReq), false, nil
}

// Check for username prompt override from connector. Defaults to "Username".
func usernamePrompt(conn connector.PasswordConnector) string {
	if attr := conn.Prompt(); attr != "" {
		return attr
	}
	return "Username"
}

package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"maps"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/gorilla/mux"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/featureflags"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

const (
	codeChallengeMethodPlain = "plain"
	codeChallengeMethodS256  = "S256"
)

func (s *Server) handlePublicKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// TODO(ericchiang): Cache this.
	keys, err := s.signer.ValidationKeys(ctx)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to get keys", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	if len(keys) == 0 {
		s.logger.ErrorContext(r.Context(), "no public keys found.")
		s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	jwks := jose.JSONWebKeySet{
		Keys: make([]jose.JSONWebKey, len(keys)),
	}
	for i, key := range keys {
		jwks.Keys[i] = *key
	}

	data, err := json.MarshalIndent(jwks, "", "  ")
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to marshal discovery data", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	// We don't have NextRotation info from Signer interface easily,
	// so we'll just set a reasonable default cache time.
	maxAge := time.Minute * 10

	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, must-revalidate", int(maxAge.Seconds())))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}

type discovery struct {
	Issuer            string   `json:"issuer"`
	Auth              string   `json:"authorization_endpoint"`
	Token             string   `json:"token_endpoint"`
	Keys              string   `json:"jwks_uri"`
	UserInfo          string   `json:"userinfo_endpoint"`
	DeviceEndpoint    string   `json:"device_authorization_endpoint"`
	Introspect        string   `json:"introspection_endpoint"`
	EndSession        string   `json:"end_session_endpoint,omitempty"`
	GrantTypes        []string `json:"grant_types_supported"`
	ResponseTypes     []string `json:"response_types_supported"`
	Subjects          []string `json:"subject_types_supported"`
	IDTokenAlgs       []string `json:"id_token_signing_alg_values_supported"`
	CodeChallengeAlgs []string `json:"code_challenge_methods_supported"`
	Scopes            []string `json:"scopes_supported"`
	AuthMethods       []string `json:"token_endpoint_auth_methods_supported"`
	Claims            []string `json:"claims_supported"`
}

func (s *Server) discoveryHandler(ctx context.Context) (http.HandlerFunc, error) {
	d := s.constructDiscovery(ctx)

	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal discovery data: %v", err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.Write(data)
	}), nil
}

func (s *Server) constructDiscovery(ctx context.Context) discovery {
	d := discovery{
		Issuer:            s.issuerURL.String(),
		Auth:              s.absURL("/auth"),
		Token:             s.absURL("/token"),
		Keys:              s.absURL("/keys"),
		UserInfo:          s.absURL("/userinfo"),
		DeviceEndpoint:    s.absURL("/device/code"),
		Introspect:        s.absURL("/token/introspect"),
		Subjects:          []string{"public"},
		IDTokenAlgs:       []string{string(jose.RS256)},
		CodeChallengeAlgs: s.pkce.CodeChallengeMethodsSupported,
		Scopes:            []string{"openid", "email", "groups", "profile", "offline_access"},
		AuthMethods:       []string{"client_secret_basic", "client_secret_post"},
		Claims: []string{
			"iss", "sub", "aud", "iat", "exp", "email", "email_verified",
			"locale", "name", "preferred_username", "at_hash",
		},
	}

	// Determine signing algorithm from signer
	signingAlg, err := s.signer.Algorithm(ctx)
	if err != nil {
		s.logger.Error("failed to get signing algorithm", "err", err)
	} else {
		d.IDTokenAlgs = []string{string(signingAlg)}
	}

	for responseType := range s.supportedResponseTypes {
		d.ResponseTypes = append(d.ResponseTypes, responseType)
	}
	sort.Strings(d.ResponseTypes)

	d.GrantTypes = s.supportedGrantTypes

	if s.sessionConfig != nil {
		d.EndSession = s.absURL("/logout")
	}

	return d
}

// grantTypeFromAuthRequest determines the grant type from the authorization request parameters.
func (s *Server) grantTypeFromAuthRequest(r *http.Request) string {
	redirectURI := r.Form.Get("redirect_uri")
	if redirectURI == deviceCallbackURI || strings.HasSuffix(redirectURI, deviceCallbackURI) {
		return grantTypeDeviceCode
	}
	responseType := r.Form.Get("response_type")
	for _, rt := range strings.Fields(responseType) {
		if rt == "token" || rt == "id_token" {
			return grantTypeImplicit
		}
	}
	return grantTypeAuthorizationCode
}

// handleAuthorization handles the OAuth2 auth endpoint.
func (s *Server) handleAuthorization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Extract the arguments
	if err := r.ParseForm(); err != nil {
		s.logger.ErrorContext(r.Context(), "failed to parse arguments", "err", err)

		s.renderError(r, w, http.StatusBadRequest, ErrMsgInvalidRequest)
		return
	}

	connectorID := r.Form.Get("connector_id")
	allConnectors, err := s.storage.ListConnectors(ctx)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to get list of connectors", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Failed to retrieve connector list.")
		return
	}

	// Determine the grant type from the authorization request to filter connectors.
	grantType := s.grantTypeFromAuthRequest(r)
	connectors := make([]storage.Connector, 0, len(allConnectors))
	for _, c := range allConnectors {
		if GrantTypeAllowed(c.GrantTypes, grantType) {
			connectors = append(connectors, c)
		}
	}

	// Filter connectors based on the client's allowed connectors list.
	// client_id is required per RFC 6749 §4.1.1.
	client, authErr := s.getClientWithAuthError(ctx, r.Form.Get("client_id"))
	if authErr != nil {
		s.renderError(r, w, authErr.Status, authErr.Error())
		return
	}
	connectors = filterConnectors(connectors, client.AllowedConnectors)

	if len(connectors) == 0 {
		s.renderError(r, w, http.StatusBadRequest, "No connectors available for this client.")
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
				connURL.Path = s.absPath("/auth", url.PathEscape(c.ID))
				http.Redirect(w, r, connURL.String(), http.StatusFound)
				return
			}
		}
		s.renderError(r, w, http.StatusBadRequest, "Connector ID does not match a valid Connector")
		return
	}

	if len(connectors) == 1 && !s.alwaysShowLogin {
		connURL.Path = s.absPath("/auth", url.PathEscape(connectors[0].ID))
		http.Redirect(w, r, connURL.String(), http.StatusFound)
		return
	}

	// Skip connector selection if a valid session exists, unless prompt=select_account or alwaysShowLogin.
	if s.sessionConfig != nil {
		authReq, _, err := s.parseAuthorizationRequest(r)
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
		prompt, err := ParsePrompt(authReq.Prompt)
		if err != nil {
			// Server error because authReq was validated before saving it to database.
			s.redirectWithError(w, r, authReq, errServerError, "Invalid authentication request")
			return
		}

		// Invalid prompts will be validated and properly redirected later
		if !s.alwaysShowLogin && !prompt.SelectAccount() {
			session := s.getValidSession(ctx, w, r)
			if session != nil {
				for _, c := range connectors {
					if c.ID != session.ConnectorID {
						continue
					}
					connURL.Path = s.absPath("/auth", url.PathEscape(session.ConnectorID))
					http.Redirect(w, r, connURL.String(), http.StatusFound)
					return
				}
			}
		}
		if prompt.None() {
			// Cannot authenticate silently with prompt=none.
			s.redirectWithError(w, r, authReq, errLoginRequired, "id_token_hint does not match authenticated user")
			return
		}
	}

	connectorInfos := make([]connectorInfo, 0, len(connectors))
	for _, conn := range connectors {
		connURL.Path = s.absPath("/auth", url.PathEscape(conn.ID))
		connectorInfos = append(connectorInfos, connectorInfo{
			ID:   conn.ID,
			Name: conn.Name,
			Type: conn.Type,
			URL:  template.URL(connURL.String()),
		})
	}

	if err := s.templates.login(r, w, connectorInfos); err != nil {
		s.logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

// filterConnectors filters the list of connectors by the allowed connector IDs.
// If allowedConnectors is empty, all connectors are returned (no filtering).
func filterConnectors(connectors []storage.Connector, allowedConnectors []string) []storage.Connector {
	if len(allowedConnectors) == 0 {
		return connectors
	}

	allowed := make(map[string]bool, len(allowedConnectors))
	for _, id := range allowedConnectors {
		allowed[id] = true
	}

	filtered := make([]storage.Connector, 0, len(connectors))
	for _, c := range connectors {
		if allowed[c.ID] {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// isConnectorAllowed checks if a connector ID is in the client's allowed connectors list.
// If allowedConnectors is empty, all connectors are allowed.
func isConnectorAllowed(allowedConnectors []string, connectorID string) bool {
	if len(allowedConnectors) == 0 {
		return true
	}
	for _, id := range allowedConnectors {
		if id == connectorID {
			return true
		}
	}
	return false
}

// getClientWithAuthError retrieves a client by ID and returns a displayedAuthErr on failure.
// Invalid client_id is not treated as a redirect error per RFC 6749 §4.1.2.1.
// https://datatracker.ietf.org/doc/html/rfc6749#section-4.1.2.1
func (s *Server) getClientWithAuthError(ctx context.Context, clientID string) (storage.Client, *displayedAuthErr) {
	client, err := s.storage.GetClient(ctx, clientID)
	if err != nil {
		if err == storage.ErrNotFound {
			s.logger.ErrorContext(ctx, "invalid client_id provided", "client_id", clientID)
			return storage.Client{}, newDisplayedErr(http.StatusBadRequest, "Invalid client_id provided.")
		}
		s.logger.ErrorContext(ctx, "failed to get client", "client_id", clientID, "err", err)
		return storage.Client{}, newDisplayedErr(http.StatusInternalServerError, "Database error.")
	}
	return client, nil
}

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

func (s *Server) handleApproval(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	macEncoded := r.FormValue("hmac")
	if macEncoded == "" {
		s.renderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}
	authReq, err := s.storage.GetAuthRequest(ctx, r.FormValue("req"))
	if err != nil {
		if err == storage.ErrNotFound {
			s.renderError(r, w, http.StatusBadRequest, "User session error.")
			return
		}
		s.logger.ErrorContext(r.Context(), "failed to get auth request", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}
	if !authReq.LoggedIn {
		s.logger.ErrorContext(r.Context(), "auth request does not have an identity for approval")
		s.renderError(r, w, http.StatusInternalServerError, "Login process not yet finalized.")
		return
	}

	if !authReq.MFAValidated {
		// Check if MFA is actually required — if so, redirect to TOTP instead of blocking.
		// This handles the case where MFA was enabled after the auth flow started.
		mfaChain, err := s.mfaChainForClient(ctx, authReq.ClientID, authReq.ConnectorID)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to get MFA chain", "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
			return
		}
		if len(mfaChain) > 0 {
			http.Redirect(w, r, s.buildMFARedirectURL(authReq, mfaChain[0]), http.StatusSeeOther)
			return
		}
		// No MFA required but flag not set — allow through (backward compat).
	}

	if !verifyHMAC(authReq.HMACKey, macEncoded, authReq.ID, "") {
		s.renderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Skip the approval page and issue the code directly if:
		// 1. The client didn't force the approval prompt, AND
		// 2. Either the server is configured to skip approval globally,
		//    or the user has already consented to all requested scopes for this client.
		// This handles the MFA redirect case: after MFA completion the user lands on
		// /approval via GET, and we don't want to show the consent screen again.
		if !authReq.ForceApprovalPrompt {
			if s.skipApproval {
				s.sendCodeResponse(w, r, authReq)
				return
			}
			ui, err := s.storage.GetUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID)
			if err == nil && scopesCoveredByConsent(ui.Consents[authReq.ClientID], authReq.Scopes) {
				s.sendCodeResponse(w, r, authReq)
				return
			}
		}

		client, err := s.storage.GetClient(ctx, authReq.ClientID)
		if err != nil {
			s.logger.ErrorContext(r.Context(), "Failed to get client", "client_id", authReq.ClientID, "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "Failed to retrieve client.")
			return
		}
		if err := s.templates.approval(r, w, authReq.ID, authReq.Claims.Username, client.Name, authReq.Scopes); err != nil {
			s.logger.ErrorContext(r.Context(), "server template error", "err", err)
		}
	case http.MethodPost:
		if r.FormValue("approval") != "approve" {
			s.renderError(r, w, http.StatusInternalServerError, "Approval rejected.")
			return
		}
		// Persist user-approved scopes as consent for this client.
		if featureflags.SessionsEnabled.Enabled() {
			if err := s.storage.UpdateUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
				if old.Consents == nil {
					old.Consents = make(map[string][]string)
				}
				old.Consents[authReq.ClientID] = authReq.Scopes
				return old, nil
			}); err != nil {
				s.logger.ErrorContext(ctx, "failed to update user identity consents", "err", err)
			}
		}
		s.sendCodeResponse(w, r, authReq)
	}
}

func (s *Server) sendCodeResponse(w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest) {
	s.updateSessionTokenIssuedAt(r, authReq.ClientID)

	ctx := r.Context()
	if s.now().After(authReq.Expiry) {
		s.renderError(r, w, http.StatusBadRequest, "User session has expired.")
		return
	}

	if err := s.storage.DeleteAuthRequest(ctx, authReq.ID); err != nil {
		if err != storage.ErrNotFound {
			s.logger.ErrorContext(r.Context(), "Failed to delete authorization request", "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
		} else {
			s.renderError(r, w, http.StatusBadRequest, "User session error.")
		}
		return
	}
	u, err := url.Parse(authReq.RedirectURI)
	if err != nil {
		s.renderError(r, w, http.StatusInternalServerError, "Invalid redirect URI.")
		return
	}

	var (
		// Was the initial request using the implicit or hybrid flow instead of
		// the "normal" code flow?
		implicitOrHybrid = false

		// Only present in hybrid or code flow. code.ID == "" if this is not set.
		code storage.AuthCode

		// ID token returned immediately if the response_type includes "id_token".
		// Only valid for implicit and hybrid flows.
		idToken       string
		idTokenExpiry time.Time

		// Access token
		accessToken string
	)

	for _, responseType := range authReq.ResponseTypes {
		switch responseType {
		case responseTypeCode:
			code = storage.AuthCode{
				ID:            storage.NewID(),
				ClientID:      authReq.ClientID,
				ConnectorID:   authReq.ConnectorID,
				Nonce:         authReq.Nonce,
				Scopes:        authReq.Scopes,
				Claims:        authReq.Claims,
				Expiry:        s.now().Add(time.Minute * 30),
				RedirectURI:   authReq.RedirectURI,
				ConnectorData: authReq.ConnectorData,
				PKCE:          authReq.PKCE,
				AuthTime:      authReq.AuthTime,
			}
			if err := s.storage.CreateAuthCode(ctx, code); err != nil {
				s.logger.ErrorContext(r.Context(), "Failed to create auth code", "err", err)
				s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
				return
			}

			// Implicit and hybrid flows that try to use the OOB redirect URI are
			// rejected earlier. If we got here we're using the code flow.
			if authReq.RedirectURI == redirectURIOOB {
				if err := s.templates.oob(r, w, code.ID); err != nil {
					s.logger.ErrorContext(r.Context(), "server template error", "err", err)
				}
				return
			}
		case responseTypeToken:
			implicitOrHybrid = true
			var err error

			accessToken, _, err = s.newAccessToken(r.Context(), authReq.ClientID, authReq.Claims, authReq.Scopes, authReq.Nonce, authReq.ConnectorID, authReq.AuthTime)
			if err != nil {
				s.logger.ErrorContext(r.Context(), "failed to create new access token", "err", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				return
			}
		case responseTypeIDToken:
			implicitOrHybrid = true
			var err error

			idToken, idTokenExpiry, err = s.newIDToken(r.Context(), authReq.ClientID, authReq.Claims, authReq.Scopes, authReq.Nonce, accessToken, code.ID, authReq.ConnectorID, authReq.AuthTime)
			if err != nil {
				s.logger.ErrorContext(r.Context(), "failed to create ID token", "err", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				return
			}
		}
	}

	if implicitOrHybrid {
		v := url.Values{}
		if accessToken != "" {
			v.Set("access_token", accessToken)
			v.Set("token_type", "bearer")
			// The hybrid flow with "code token" or "code id_token token" doesn't return an
			// "expires_in" value. If "code" wasn't provided, indicating the implicit flow,
			// don't add it.
			//
			// https://openid.net/specs/openid-connect-core-1_0.html#HybridAuthResponse
			if code.ID == "" {
				v.Set("expires_in", strconv.Itoa(int(idTokenExpiry.Sub(s.now()).Seconds())))
			}
		}
		v.Set("state", authReq.State)
		if idToken != "" {
			v.Set("id_token", idToken)
		}
		if code.ID != "" {
			v.Set("code", code.ID)
		}

		// Implicit and hybrid flows return their values as part of the fragment.
		//
		//   HTTP/1.1 303 See Other
		//   Location: https://client.example.org/cb#
		//     access_token=SlAV32hkKG
		//     &token_type=bearer
		//     &id_token=eyJ0 ... NiJ9.eyJ1c ... I6IjIifX0.DeWt4Qu ... ZXso
		//     &expires_in=3600
		//     &state=af0ifjsldkj
		//
		u.Fragment = v.Encode()
	} else {
		// The code flow add values to the URL query.
		//
		//   HTTP/1.1 303 See Other
		//   Location: https://client.example.org/cb?
		//     code=SplxlOBeZQQYbYS6WxSbIA
		//     &state=af0ifjsldkj
		//
		q := u.Query()
		q.Set("code", code.ID)
		q.Set("state", authReq.State)
		u.RawQuery = q.Encode()
	}

	http.Redirect(w, r, u.String(), http.StatusSeeOther)
}

// scopesCoveredByConsent checks whether the approved scopes cover all requested scopes.
// The openid scope is excluded from the comparison as it is a technical scope
// that does not require user consent.
func scopesCoveredByConsent(approved, requested []string) bool {
	approvedSet := make(map[string]struct{}, len(approved))
	for _, s := range approved {
		approvedSet[s] = struct{}{}
	}

	for _, scope := range requested {
		if scope == scopeOpenID {
			continue
		}
		if _, ok := approvedSet[scope]; !ok {
			return false
		}
	}

	return true
}

func (s *Server) withClientFromStorage(w http.ResponseWriter, r *http.Request, handler func(http.ResponseWriter, *http.Request, storage.Client)) {
	ctx := r.Context()
	clientID, clientSecret, ok := r.BasicAuth()
	if ok {
		var err error
		if clientID, err = url.QueryUnescape(clientID); err != nil {
			s.tokenErrHelper(w, errInvalidRequest, "client_id improperly encoded", http.StatusBadRequest)
			return
		}
		if clientSecret, err = url.QueryUnescape(clientSecret); err != nil {
			s.tokenErrHelper(w, errInvalidRequest, "client_secret improperly encoded", http.StatusBadRequest)
			return
		}
	} else {
		clientID = r.PostFormValue("client_id")
		clientSecret = r.PostFormValue("client_secret")
	}

	client, err := s.storage.GetClient(ctx, clientID)
	if err != nil {
		if err != storage.ErrNotFound {
			s.logger.ErrorContext(r.Context(), "failed to get client", "err", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		} else {
			s.tokenErrHelper(w, errInvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
		}
		return
	}

	if subtle.ConstantTimeCompare([]byte(client.Secret), []byte(clientSecret)) != 1 {
		if clientSecret == "" {
			s.logger.InfoContext(r.Context(), "missing client_secret on token request", "client_id", client.ID)
		} else {
			s.logger.InfoContext(r.Context(), "invalid client_secret on token request", "client_id", client.ID)
		}
		s.tokenErrHelper(w, errInvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
		return
	}

	handler(w, r, client)
}

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		s.tokenErrHelper(w, errInvalidRequest, "method not allowed", http.StatusBadRequest)
		return
	}

	err := r.ParseForm()
	if err != nil {
		s.logger.ErrorContext(r.Context(), "could not parse request body", "err", err)
		s.tokenErrHelper(w, errInvalidRequest, "", http.StatusBadRequest)
		return
	}

	grantType := r.PostFormValue("grant_type")
	if !contains(s.supportedGrantTypes, grantType) {
		s.logger.ErrorContext(r.Context(), "unsupported grant type", "grant_type", grantType)
		s.tokenErrHelper(w, errUnsupportedGrantType, "", http.StatusBadRequest)
		return
	}
	switch grantType {
	case grantTypeDeviceCode:
		s.handleDeviceToken(w, r)
	case grantTypeAuthorizationCode:
		s.withClientFromStorage(w, r, s.handleAuthCode)
	case grantTypeRefreshToken:
		s.withClientFromStorage(w, r, s.handleRefreshToken)
	case grantTypePassword:
		s.withClientFromStorage(w, r, s.handlePasswordGrant)
	case grantTypeTokenExchange:
		s.withClientFromStorage(w, r, s.handleTokenExchange)
	case grantTypeClientCredentials:
		s.withClientFromStorage(w, r, s.handleClientCredentialsGrant)
	default:
		s.tokenErrHelper(w, errUnsupportedGrantType, "", http.StatusBadRequest)
	}
}

func (s *Server) calculateCodeChallenge(codeVerifier, codeChallengeMethod string) (string, error) {
	switch codeChallengeMethod {
	case codeChallengeMethodPlain:
		return codeVerifier, nil
	case codeChallengeMethodS256:
		shaSum := sha256.Sum256([]byte(codeVerifier))
		return base64.RawURLEncoding.EncodeToString(shaSum[:]), nil
	default:
		return "", fmt.Errorf("unknown challenge method (%v)", codeChallengeMethod)
	}
}

// handle an access token request https://tools.ietf.org/html/rfc6749#section-4.1.3
func (s *Server) handleAuthCode(w http.ResponseWriter, r *http.Request, client storage.Client) {
	ctx := r.Context()
	code := r.PostFormValue("code")
	redirectURI, err := url.QueryUnescape(r.PostFormValue("redirect_uri"))
	if err != nil {
		s.tokenErrHelper(w, errInvalidRequest, "No redirect_uri provided.", http.StatusBadRequest)
		return
	}

	if code == "" {
		s.tokenErrHelper(w, errInvalidRequest, `Required param: code.`, http.StatusBadRequest)
		return
	}

	authCode, err := s.storage.GetAuthCode(ctx, code)
	if err != nil || s.now().After(authCode.Expiry) || authCode.ClientID != client.ID {
		if err != storage.ErrNotFound {
			s.logger.ErrorContext(r.Context(), "failed to get auth code", "err", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		} else {
			s.tokenErrHelper(w, errInvalidGrant, "Invalid or expired code parameter.", http.StatusBadRequest)
		}
		return
	}

	// RFC 7636 (PKCE)
	codeChallengeFromStorage := authCode.PKCE.CodeChallenge
	providedCodeVerifier := r.PostFormValue("code_verifier")

	switch {
	case providedCodeVerifier != "" && codeChallengeFromStorage != "":
		calculatedCodeChallenge, err := s.calculateCodeChallenge(providedCodeVerifier, authCode.PKCE.CodeChallengeMethod)
		if err != nil {
			s.logger.ErrorContext(r.Context(), "failed to calculate code challenge", "err", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
			return
		}
		if codeChallengeFromStorage != calculatedCodeChallenge {
			s.tokenErrHelper(w, errInvalidGrant, "Invalid code_verifier.", http.StatusBadRequest)
			return
		}
	case providedCodeVerifier != "":
		// Received no code_challenge on /auth, but a code_verifier on /token
		s.tokenErrHelper(w, errInvalidRequest, "No PKCE flow started. Cannot check code_verifier.", http.StatusBadRequest)
		return
	case codeChallengeFromStorage != "":
		// Received PKCE request on /auth, but no code_verifier on /token
		s.tokenErrHelper(w, errInvalidGrant, "Expecting parameter code_verifier in PKCE flow.", http.StatusBadRequest)
		return
	}

	if authCode.RedirectURI != redirectURI {
		s.tokenErrHelper(w, errInvalidRequest, "redirect_uri did not match URI from initial request.", http.StatusBadRequest)
		return
	}

	tokenResponse, err := s.exchangeAuthCode(ctx, w, authCode, client)
	if err != nil {
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}
	s.writeAccessToken(w, tokenResponse)
}

func (s *Server) exchangeAuthCode(ctx context.Context, w http.ResponseWriter, authCode storage.AuthCode, client storage.Client) (*accessTokenResponse, error) {
	accessToken, _, err := s.newAccessToken(ctx, client.ID, authCode.Claims, authCode.Scopes, authCode.Nonce, authCode.ConnectorID, authCode.AuthTime)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create new access token", "err", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return nil, err
	}

	idToken, expiry, err := s.newIDToken(ctx, client.ID, authCode.Claims, authCode.Scopes, authCode.Nonce, accessToken, authCode.ID, authCode.ConnectorID, authCode.AuthTime)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create ID token", "err", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return nil, err
	}

	if err := s.storage.DeleteAuthCode(ctx, authCode.ID); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete auth code", "err", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return nil, err
	}

	reqRefresh := func() bool {
		// Determine whether to issue a refresh token. A refresh token is only
		// issued when all of the following are true:
		//   1. The connector implements RefreshConnector.
		//   2. The connector's grantTypes config allows refresh_token.
		//   3. The client requested the offline_access scope.
		//
		// When any condition is not met, the refresh token is silently omitted
		// rather than returning an error. This matches the OAuth2 spec: the
		// server is never required to issue a refresh token (RFC 6749 §1.5).
		// https://datatracker.ietf.org/doc/html/rfc6749#section-1.5
		conn, err := s.getConnector(ctx, authCode.ConnectorID)
		if err != nil {
			s.logger.ErrorContext(ctx, "connector not found", "connector_id", authCode.ConnectorID, "err", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
			return false
		}

		_, ok := conn.Connector.(connector.RefreshConnector)
		if !ok {
			return false
		}

		if !GrantTypeAllowed(conn.GrantTypes, grantTypeRefreshToken) {
			return false
		}

		for _, scope := range authCode.Scopes {
			if scope == scopeOfflineAccess {
				return true
			}
		}
		return false
	}()
	var refreshToken string
	if reqRefresh {
		refresh := storage.RefreshToken{
			ID:            storage.NewID(),
			Token:         storage.NewID(),
			ClientID:      authCode.ClientID,
			ConnectorID:   authCode.ConnectorID,
			Scopes:        authCode.Scopes,
			Claims:        authCode.Claims,
			Nonce:         authCode.Nonce,
			ConnectorData: authCode.ConnectorData,
			CreatedAt:     s.now(),
			LastUsed:      s.now(),
		}
		token := &internal.RefreshToken{
			RefreshId: refresh.ID,
			Token:     refresh.Token,
		}
		if refreshToken, err = internal.Marshal(token); err != nil {
			s.logger.ErrorContext(ctx, "failed to marshal refresh token", "err", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
			return nil, err
		}

		if err := s.storage.CreateRefresh(ctx, refresh); err != nil {
			s.logger.ErrorContext(ctx, "failed to create refresh token", "err", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
			return nil, err
		}

		// deleteToken determines if we need to delete the newly created refresh token
		// due to a failure in updating/creating the OfflineSession object for the
		// corresponding user.
		var deleteToken bool
		defer func() {
			if deleteToken {
				// Delete newly created refresh token from storage.
				if err := s.storage.DeleteRefresh(ctx, refresh.ID); err != nil {
					s.logger.ErrorContext(ctx, "failed to delete refresh token", "err", err)
					s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
					return
				}
			}
		}()

		tokenRef := storage.RefreshTokenRef{
			ID:        refresh.ID,
			ClientID:  refresh.ClientID,
			CreatedAt: refresh.CreatedAt,
			LastUsed:  refresh.LastUsed,
		}

		// Try to retrieve an existing OfflineSession object for the corresponding user.
		if session, err := s.storage.GetOfflineSessions(ctx, refresh.Claims.UserID, refresh.ConnectorID); err != nil {
			if err != storage.ErrNotFound {
				s.logger.ErrorContext(ctx, "failed to get offline session", "err", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return nil, err
			}
			offlineSessions := storage.OfflineSessions{
				UserID:        refresh.Claims.UserID,
				ConnID:        refresh.ConnectorID,
				Refresh:       make(map[string]*storage.RefreshTokenRef),
				ConnectorData: refresh.ConnectorData,
			}
			offlineSessions.Refresh[tokenRef.ClientID] = &tokenRef

			// Create a new OfflineSession object for the user and add a reference object for
			// the newly received refreshtoken.
			if err := s.storage.CreateOfflineSessions(ctx, offlineSessions); err != nil {
				s.logger.ErrorContext(ctx, "failed to create offline session", "err", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return nil, err
			}
		} else {
			if oldTokenRef, ok := session.Refresh[tokenRef.ClientID]; ok {
				// Delete old refresh token from storage.
				if err := s.storage.DeleteRefresh(ctx, oldTokenRef.ID); err != nil && err != storage.ErrNotFound {
					s.logger.ErrorContext(ctx, "failed to delete refresh token", "err", err)
					s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
					deleteToken = true
					return nil, err
				}
			}

			// Update existing OfflineSession obj with new RefreshTokenRef.
			if err := s.storage.UpdateOfflineSessions(ctx, session.UserID, session.ConnID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
				old.Refresh[tokenRef.ClientID] = &tokenRef
				if len(refresh.ConnectorData) > 0 {
					old.ConnectorData = refresh.ConnectorData
				}
				return old, nil
			}); err != nil {
				s.logger.ErrorContext(ctx, "failed to update offline session", "err", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return nil, err
			}
		}
	}
	return s.toAccessTokenResponse(idToken, accessToken, refreshToken, expiry), nil
}

func (s *Server) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	const prefix = "Bearer "

	auth := r.Header.Get("authorization")
	if len(auth) < len(prefix) || !strings.EqualFold(prefix, auth[:len(prefix)]) {
		w.Header().Set("WWW-Authenticate", "Bearer")
		s.tokenErrHelper(w, errAccessDenied, "Invalid bearer token.", http.StatusUnauthorized)
		return
	}
	rawIDToken := auth[len(prefix):]

	verifier := oidc.NewVerifier(s.issuerURL.String(), &signerKeySet{s.signer}, &oidc.Config{SkipClientIDCheck: true})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to verify ID token", "err", err)
		s.tokenErrHelper(w, errAccessDenied, "Invalid bearer token.", http.StatusForbidden)
		return
	}

	var claims json.RawMessage
	if err := idToken.Claims(&claims); err != nil {
		s.logger.ErrorContext(r.Context(), "failed to decode ID token claims", "err", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(claims)
}

func (s *Server) handlePasswordGrant(w http.ResponseWriter, r *http.Request, client storage.Client) {
	ctx := r.Context()
	// Parse the fields
	if err := r.ParseForm(); err != nil {
		s.tokenErrHelper(w, errInvalidRequest, "Couldn't parse data", http.StatusBadRequest)
		return
	}
	q := r.Form

	nonce := q.Get("nonce")
	// Some clients, like the old go-oidc, provide extra whitespace. Tolerate this.
	scopes := strings.Fields(q.Get("scope"))

	// Parse the scopes if they are passed
	var (
		unrecognized  []string
		invalidScopes []string
	)
	hasOpenIDScope := false
	for _, scope := range scopes {
		switch scope {
		case scopeOpenID:
			hasOpenIDScope = true
		case scopeOfflineAccess, scopeEmail, scopeProfile, scopeGroups, scopeFederatedID:
		default:
			peerID, ok := parseCrossClientScope(scope)
			if !ok {
				unrecognized = append(unrecognized, scope)
				continue
			}

			isTrusted, err := s.validateCrossClientTrust(ctx, client.ID, peerID)
			if err != nil {
				s.logger.ErrorContext(r.Context(), "error validating cross client trust", "client_id", client.ID, "peer_id", peerID, "err", err)
				s.tokenErrHelper(w, errInvalidClient, "Error validating cross client trust.", http.StatusBadRequest)
				return
			}
			if !isTrusted {
				invalidScopes = append(invalidScopes, scope)
			}
		}
	}
	if !hasOpenIDScope {
		s.tokenErrHelper(w, errInvalidRequest, `Missing required scope(s) ["openid"].`, http.StatusBadRequest)
		return
	}
	if len(unrecognized) > 0 {
		s.tokenErrHelper(w, errInvalidRequest, fmt.Sprintf("Unrecognized scope(s) %q", unrecognized), http.StatusBadRequest)
		return
	}
	if len(invalidScopes) > 0 {
		s.tokenErrHelper(w, errInvalidRequest, fmt.Sprintf("Client can't request scope(s) %q", invalidScopes), http.StatusBadRequest)
		return
	}

	// Which connector
	connID := s.passwordConnector
	conn, err := s.getConnector(ctx, connID)
	if err != nil {
		s.tokenErrHelper(w, errInvalidRequest, "Requested connector does not exist.", http.StatusBadRequest)
		return
	}
	if !GrantTypeAllowed(conn.GrantTypes, grantTypePassword) {
		s.logger.ErrorContext(r.Context(), "connector does not allow password grant", "connector_id", connID)
		s.tokenErrHelper(w, errInvalidRequest, "Requested connector does not support password grant.", http.StatusBadRequest)
		return
	}

	passwordConnector, ok := conn.Connector.(connector.PasswordConnector)
	if !ok {
		s.tokenErrHelper(w, errInvalidRequest, "Requested password connector does not correct type.", http.StatusBadRequest)
		return
	}

	// Login
	username := q.Get("username")
	password := q.Get("password")
	identity, ok, err := passwordConnector.Login(ctx, parseScopes(scopes), username, password)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to login user", "err", err)
		s.tokenErrHelper(w, errInvalidRequest, "Could not login user", http.StatusBadRequest)
		return
	}
	if !ok {
		s.tokenErrHelper(w, errAccessDenied, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Build the claims to send the id token
	claims := storage.Claims{
		UserID:            identity.UserID,
		Username:          identity.Username,
		PreferredUsername: identity.PreferredUsername,
		Email:             identity.Email,
		EmailVerified:     identity.EmailVerified,
		Groups:            identity.Groups,
	}

	accessToken, _, err := s.newAccessToken(ctx, client.ID, claims, scopes, nonce, connID, time.Time{})
	if err != nil {
		s.logger.ErrorContext(r.Context(), "password grant failed to create new access token", "err", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	idToken, expiry, err := s.newIDToken(ctx, client.ID, claims, scopes, nonce, accessToken, "", connID, time.Time{})
	if err != nil {
		s.logger.ErrorContext(r.Context(), "password grant failed to create new ID token", "err", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	reqRefresh := func() bool {
		// Same logic as in exchangeAuthCode: silently omit refresh token
		// when the connector doesn't support it or grantTypes forbids it.
		// See RFC 6749 §1.5 — refresh tokens are never mandatory.
		// https://datatracker.ietf.org/doc/html/rfc6749#section-1.5
		if _, ok := conn.Connector.(connector.RefreshConnector); !ok {
			return false
		}

		if !GrantTypeAllowed(conn.GrantTypes, grantTypeRefreshToken) {
			return false
		}

		for _, scope := range scopes {
			if scope == scopeOfflineAccess {
				return true
			}
		}
		return false
	}()
	var refreshToken string
	if reqRefresh {
		refresh := storage.RefreshToken{
			ID:          storage.NewID(),
			Token:       storage.NewID(),
			ClientID:    client.ID,
			ConnectorID: connID,
			Scopes:      scopes,
			Claims:      claims,
			Nonce:       nonce,
			// ConnectorData: authCode.ConnectorData,
			CreatedAt: s.now(),
			LastUsed:  s.now(),
		}
		token := &internal.RefreshToken{
			RefreshId: refresh.ID,
			Token:     refresh.Token,
		}
		if refreshToken, err = internal.Marshal(token); err != nil {
			s.logger.ErrorContext(r.Context(), "failed to marshal refresh token", "err", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
			return
		}

		if err := s.storage.CreateRefresh(ctx, refresh); err != nil {
			s.logger.ErrorContext(r.Context(), "failed to create refresh token", "err", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
			return
		}

		// deleteToken determines if we need to delete the newly created refresh token
		// due to a failure in updating/creating the OfflineSession object for the
		// corresponding user.
		var deleteToken bool
		defer func() {
			if deleteToken {
				// Delete newly created refresh token from storage.
				if err := s.storage.DeleteRefresh(ctx, refresh.ID); err != nil {
					s.logger.ErrorContext(r.Context(), "failed to delete refresh token", "err", err)
					s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
					return
				}
			}
		}()

		tokenRef := storage.RefreshTokenRef{
			ID:        refresh.ID,
			ClientID:  refresh.ClientID,
			CreatedAt: refresh.CreatedAt,
			LastUsed:  refresh.LastUsed,
		}

		// Try to retrieve an existing OfflineSession object for the corresponding user.
		if session, err := s.storage.GetOfflineSessions(ctx, refresh.Claims.UserID, refresh.ConnectorID); err != nil {
			if err != storage.ErrNotFound {
				s.logger.ErrorContext(r.Context(), "failed to get offline session", "err", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return
			}
			offlineSessions := storage.OfflineSessions{
				UserID:        refresh.Claims.UserID,
				ConnID:        refresh.ConnectorID,
				Refresh:       make(map[string]*storage.RefreshTokenRef),
				ConnectorData: identity.ConnectorData,
			}
			offlineSessions.Refresh[tokenRef.ClientID] = &tokenRef

			// Create a new OfflineSession object for the user and add a reference object for
			// the newly received refreshtoken.
			if err := s.storage.CreateOfflineSessions(ctx, offlineSessions); err != nil {
				s.logger.ErrorContext(r.Context(), "failed to create offline session", "err", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return
			}
		} else {
			if oldTokenRef, ok := session.Refresh[tokenRef.ClientID]; ok {
				// Delete old refresh token from storage.
				if err := s.storage.DeleteRefresh(ctx, oldTokenRef.ID); err != nil {
					if err == storage.ErrNotFound {
						s.logger.Warn("database inconsistent, refresh token missing", "token_id", oldTokenRef.ID)
					} else {
						s.logger.ErrorContext(r.Context(), "failed to delete refresh token", "err", err)
						s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
						deleteToken = true
						return
					}
				}
			}

			// Update existing OfflineSession obj with new RefreshTokenRef.
			if err := s.storage.UpdateOfflineSessions(ctx, session.UserID, session.ConnID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
				old.Refresh[tokenRef.ClientID] = &tokenRef
				old.ConnectorData = identity.ConnectorData
				return old, nil
			}); err != nil {
				s.logger.ErrorContext(r.Context(), "failed to update offline session", "err", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return
			}
		}
	}

	resp := s.toAccessTokenResponse(idToken, accessToken, refreshToken, expiry)
	s.writeAccessToken(w, resp)
}

func (s *Server) handleTokenExchange(w http.ResponseWriter, r *http.Request, client storage.Client) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		s.logger.ErrorContext(r.Context(), "could not parse request body", "err", err)
		s.tokenErrHelper(w, errInvalidRequest, "", http.StatusBadRequest)
		return
	}
	q := r.Form

	scopes := strings.Fields(q.Get("scope"))            // OPTIONAL, map to issued token scope
	requestedTokenType := q.Get("requested_token_type") // OPTIONAL, default to access token
	if requestedTokenType == "" {
		requestedTokenType = tokenTypeAccess
	}
	subjectToken := q.Get("subject_token")          // REQUIRED
	subjectTokenType := q.Get("subject_token_type") // REQUIRED
	connID := q.Get("connector_id")                 // REQUIRED, not in RFC

	switch subjectTokenType {
	case tokenTypeID, tokenTypeAccess: // ok, continue
	default:
		s.tokenErrHelper(w, errRequestNotSupported, "Invalid subject_token_type.", http.StatusBadRequest)
		return
	}

	if subjectToken == "" {
		s.tokenErrHelper(w, errInvalidRequest, "Missing subject_token", http.StatusBadRequest)
		return
	}

	conn, err := s.getConnector(ctx, connID)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to get connector", "err", err)
		s.tokenErrHelper(w, errInvalidRequest, "Requested connector does not exist.", http.StatusBadRequest)
		return
	}
	if !GrantTypeAllowed(conn.GrantTypes, grantTypeTokenExchange) {
		s.logger.ErrorContext(r.Context(), "connector does not allow token exchange", "connector_id", connID)
		s.tokenErrHelper(w, errInvalidRequest, "Requested connector does not support token exchange.", http.StatusBadRequest)
		return
	}
	teConn, ok := conn.Connector.(connector.TokenIdentityConnector)
	if !ok {
		s.logger.ErrorContext(r.Context(), "connector doesn't implement token exchange", "connector_id", connID)
		s.tokenErrHelper(w, errInvalidRequest, "Requested connector does not exist.", http.StatusBadRequest)
		return
	}
	identity, err := teConn.TokenIdentity(ctx, subjectTokenType, subjectToken)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to verify subject token", "err", err)
		s.tokenErrHelper(w, errAccessDenied, "", http.StatusUnauthorized)
		return
	}

	claims := storage.Claims{
		UserID:            identity.UserID,
		Username:          identity.Username,
		PreferredUsername: identity.PreferredUsername,
		Email:             identity.Email,
		EmailVerified:     identity.EmailVerified,
		Groups:            identity.Groups,
	}
	resp := accessTokenResponse{
		IssuedTokenType: requestedTokenType,
		TokenType:       "bearer",
	}
	var expiry time.Time
	switch requestedTokenType {
	case tokenTypeID:
		resp.AccessToken, expiry, err = s.newIDToken(r.Context(), client.ID, claims, scopes, "", "", "", connID, time.Time{})
	case tokenTypeAccess:
		resp.AccessToken, expiry, err = s.newAccessToken(r.Context(), client.ID, claims, scopes, "", connID, time.Time{})
	default:
		s.tokenErrHelper(w, errRequestNotSupported, "Invalid requested_token_type.", http.StatusBadRequest)
		return
	}
	if err != nil {
		s.logger.ErrorContext(r.Context(), "token exchange failed to create new token", "requested_token_type", requestedTokenType, "err", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}
	resp.ExpiresIn = int(time.Until(expiry).Seconds())

	// Token response must include cache headers https://tools.ietf.org/html/rfc6749#section-5.1
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleClientCredentialsGrant(w http.ResponseWriter, r *http.Request, client storage.Client) {
	ctx := r.Context()

	// client_credentials requires a confidential client.
	if client.Public {
		s.tokenErrHelper(w, errUnauthorizedClient, "Public clients cannot use client_credentials grant.", http.StatusBadRequest)
		return
	}

	// Parse scopes from request.
	if err := r.ParseForm(); err != nil {
		s.tokenErrHelper(w, errInvalidRequest, "Couldn't parse data", http.StatusBadRequest)
		return
	}
	scopes := strings.Fields(r.Form.Get("scope"))

	// Validate scopes.
	var (
		unrecognized  []string
		invalidScopes []string
	)
	hasOpenIDScope := false
	for _, scope := range scopes {
		switch scope {
		case scopeOpenID:
			hasOpenIDScope = true
		case scopeEmail, scopeProfile, scopeGroups:
			// allowed
		case scopeOfflineAccess:
			s.tokenErrHelper(w, errInvalidScope, "client_credentials grant does not support offline_access scope.", http.StatusBadRequest)
			return
		case scopeFederatedID:
			s.tokenErrHelper(w, errInvalidScope, "client_credentials grant does not support federated:id scope.", http.StatusBadRequest)
			return
		default:
			peerID, ok := parseCrossClientScope(scope)
			if !ok {
				unrecognized = append(unrecognized, scope)
				continue
			}

			isTrusted, err := s.validateCrossClientTrust(ctx, client.ID, peerID)
			if err != nil {
				s.logger.ErrorContext(ctx, "error validating cross client trust", "client_id", client.ID, "peer_id", peerID, "err", err)
				s.tokenErrHelper(w, errInvalidClient, "Error validating cross client trust.", http.StatusBadRequest)
				return
			}
			if !isTrusted {
				invalidScopes = append(invalidScopes, scope)
			}
		}
	}
	if len(unrecognized) > 0 {
		s.tokenErrHelper(w, errInvalidScope, fmt.Sprintf("Unrecognized scope(s) %q", unrecognized), http.StatusBadRequest)
		return
	}
	if len(invalidScopes) > 0 {
		s.tokenErrHelper(w, errInvalidScope, fmt.Sprintf("Client can't request scope(s) %q", invalidScopes), http.StatusBadRequest)
		return
	}

	// Build claims from the client itself — no user involved.
	claims := storage.Claims{
		UserID: client.ID,
	}

	// Only populate Username/PreferredUsername when the profile scope is requested.
	for _, scope := range scopes {
		if scope == scopeProfile {
			claims.Username = client.Name
			claims.PreferredUsername = client.Name
			break
		}
	}

	nonce := r.Form.Get("nonce")

	// Empty connector ID is unique for cluster credentials grant
	// Creating connectors with an empty ID with the config and API is prohibited
	connID := ""

	accessToken, expiry, err := s.newAccessToken(ctx, client.ID, claims, scopes, nonce, connID, time.Time{})
	if err != nil {
		s.logger.ErrorContext(ctx, "client_credentials grant failed to create new access token", "err", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	var idToken string
	if hasOpenIDScope {
		idToken, expiry, err = s.newIDToken(ctx, client.ID, claims, scopes, nonce, accessToken, "", connID, time.Time{})
		if err != nil {
			s.logger.ErrorContext(ctx, "client_credentials grant failed to create new ID token", "err", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
			return
		}
	}

	resp := s.toAccessTokenResponse(idToken, accessToken, "", expiry)
	s.writeAccessToken(w, resp)
}

type accessTokenResponse struct {
	AccessToken     string `json:"access_token"`
	IssuedTokenType string `json:"issued_token_type,omitempty"`
	TokenType       string `json:"token_type"`
	ExpiresIn       int    `json:"expires_in,omitempty"`
	RefreshToken    string `json:"refresh_token,omitempty"`
	IDToken         string `json:"id_token,omitempty"`
	Scope           string `json:"scope,omitempty"`
}

func (s *Server) toAccessTokenResponse(idToken, accessToken, refreshToken string, expiry time.Time) *accessTokenResponse {
	return &accessTokenResponse{
		AccessToken:  accessToken,
		TokenType:    "bearer",
		ExpiresIn:    int(expiry.Sub(s.now()).Seconds()),
		RefreshToken: refreshToken,
		IDToken:      idToken,
	}
}

func (s *Server) writeAccessToken(w http.ResponseWriter, resp *accessTokenResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		// TODO(nabokihms): error with context
		s.logger.Error("failed to marshal access token response", "err", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))

	// Token response must include cache headers https://tools.ietf.org/html/rfc6749#section-5.1
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Write(data)
}

func (s *Server) renderError(r *http.Request, w http.ResponseWriter, status int, description string) {
	if err := s.templates.err(r, w, status, description); err != nil {
		s.logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

func (s *Server) tokenErrHelper(w http.ResponseWriter, typ string, description string, statusCode int) {
	if err := tokenErr(w, typ, description, statusCode); err != nil {
		// TODO(nabokihms): error with context
		s.logger.Error("token error response", "err", err)
	}
}

// Check for username prompt override from connector. Defaults to "Username".
func usernamePrompt(conn connector.PasswordConnector) string {
	if attr := conn.Prompt(); attr != "" {
		return attr
	}
	return "Username"
}

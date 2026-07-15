package server

// authorize.go handles the /auth authorization endpoint: parsing the request,
// selecting a connector, and the browser-facing (HTML/redirect) error surface.

import (
	"context"
	"crypto"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

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

	connectorInfos := make([]templates.ConnectorInfo, 0, len(connectors))
	for _, conn := range connectors {
		connURL.Path = s.absPath("/auth", url.PathEscape(conn.ID))
		connectorInfos = append(connectorInfos, templates.ConnectorInfo{
			ID:   conn.ID,
			Name: conn.Name,
			Type: conn.Type,
			URL:  template.URL(connURL.String()),
		})
	}

	if err := s.templates.Login(r, w, connectorInfos); err != nil {
		s.logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
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

func (s *Server) renderError(r *http.Request, w http.ResponseWriter, status int, description string) {
	if err := s.templates.Err(r, w, status, description); err != nil {
		s.logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

type displayedAuthErr struct {
	Status      int
	Description string
}

func (err *displayedAuthErr) Error() string {
	return err.Description
}

func newDisplayedErr(status int, format string, a ...interface{}) *displayedAuthErr {
	return &displayedAuthErr{status, fmt.Sprintf(format, a...)}
}

// redirectWithError redirects back to the client with an OAuth2 error response.
// Used for prompt=none when login or consent is required.
func (s *Server) redirectWithError(w http.ResponseWriter, r *http.Request, authReq *storage.AuthRequest, errType, description string) {
	err := &redirectedAuthErr{
		State:       authReq.State,
		RedirectURI: authReq.RedirectURI,
		Type:        errType,
		Description: description,
	}
	err.Handler().ServeHTTP(w, r)
}

// redirectedAuthErr is an error that should be reported back to the client by 302 redirect
type redirectedAuthErr struct {
	State       string
	RedirectURI string
	Type        string
	Description string
}

func (err *redirectedAuthErr) Error() string {
	return err.Description
}

func (err *redirectedAuthErr) Handler() http.Handler {
	hf := func(w http.ResponseWriter, r *http.Request) {
		v := url.Values{}
		v.Add("state", err.State)
		v.Add("error", err.Type)
		if err.Description != "" {
			v.Add("error_description", err.Description)
		}

		// Parse the redirect URI to ensure it's valid before redirecting
		u, parseErr := url.Parse(err.RedirectURI)
		if parseErr != nil {
			// If URI parsing fails, respond with an error instead of redirecting
			http.Error(w, "Invalid redirect URI", http.StatusBadRequest)
			return
		}

		// Add error parameters to the URL
		query := u.Query()
		for key, values := range v {
			for _, value := range values {
				query.Add(key, value)
			}
		}
		u.RawQuery = query.Encode()

		http.Redirect(w, r, u.String(), http.StatusSeeOther)
	}
	return http.HandlerFunc(hf)
}

// parseAuthorizationRequest parses the initial request from the OAuth2 client.
// Returns the auth request, the raw subject from id_token_hint (empty if not provided), and any error.
func (s *Server) parseAuthorizationRequest(r *http.Request) (*storage.AuthRequest, string, error) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		return nil, "", newDisplayedErr(http.StatusBadRequest, "Failed to parse request.")
	}
	q := r.Form
	// r.ParseForm already URL-decodes query values once; decoding redirect_uri a
	// second time created a normalization differential with the token endpoint.
	redirectURI := q.Get("redirect_uri")

	clientID := q.Get("client_id")
	state := q.Get("state")
	nonce := q.Get("nonce")
	connectorID := q.Get("connector_id")
	// Some clients, like the old go-oidc, provide extra whitespace. Tolerate this.
	scopes := strings.Fields(q.Get("scope"))
	responseTypes := strings.Fields(q.Get("response_type"))

	codeChallenge := q.Get("code_challenge")
	codeChallengeMethod := q.Get("code_challenge_method")

	if codeChallengeMethod == "" {
		codeChallengeMethod = codeChallengeMethodPlain
	}

	client, err := s.storage.GetClient(ctx, clientID)
	if err != nil {
		if err == storage.ErrNotFound {
			s.logger.ErrorContext(r.Context(), "invalid client_id provided", "client_id", clientID)
			return nil, "", newDisplayedErr(http.StatusNotFound, "Invalid client_id.")
		}
		s.logger.ErrorContext(r.Context(), "failed to get client", "err", err)
		return nil, "", newDisplayedErr(http.StatusInternalServerError, "Database error.")
	}

	if !validateRedirectURI(client, redirectURI) {
		s.logger.ErrorContext(r.Context(), "unregistered redirect_uri", "redirect_uri", redirectURI, "client_id", clientID)
		return nil, "", newDisplayedErr(http.StatusBadRequest, "Unregistered redirect_uri.")
	}
	if redirectURI == deviceCallbackURI && client.Public {
		redirectURI = s.absPath(deviceCallbackURI)
	}

	// From here on out, we want to redirect back to the client with an error.
	newRedirectedErr := func(typ, format string, a ...interface{}) *redirectedAuthErr {
		return &redirectedAuthErr{state, redirectURI, typ, fmt.Sprintf(format, a...)}
	}

	if connectorID != "" {
		connectors, err := s.storage.ListConnectors(ctx)
		if err != nil {
			s.logger.ErrorContext(r.Context(), "failed to list connectors", "err", err)
			return nil, "", newRedirectedErr(errServerError, "Unable to retrieve connectors")
		}
		if !validateConnectorID(connectors, connectorID) {
			return nil, "", newRedirectedErr(errInvalidRequest, "Invalid ConnectorID")
		}
		if !isConnectorAllowed(client.AllowedConnectors, connectorID) {
			return nil, "", newRedirectedErr(errInvalidRequest, "Connector not allowed for this client")
		}
	}

	// dex doesn't support request parameter and must return request_not_supported error
	// https://openid.net/specs/openid-connect-core-1_0.html#6.1
	if q.Get("request") != "" {
		return nil, "", newRedirectedErr(errRequestNotSupported, "Server does not support request parameter.")
	}

	if codeChallenge != "" && !slices.Contains(s.pkce.CodeChallengeMethodsSupported, codeChallengeMethod) {
		return nil, "", newRedirectedErr(errInvalidRequest, "Unsupported PKCE challenge method (%q).", codeChallengeMethod)
	}

	// Enforce PKCE if configured.
	// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-12#section-4.1.1
	if s.pkce.Enforce && codeChallenge == "" {
		return nil, "", newRedirectedErr(errInvalidRequest, "PKCE is required. The code_challenge parameter must be provided.")
	}

	var (
		unrecognized  []string
		invalidScopes []string
	)
	hasOpenIDScope := false
	for _, scope := range scopes {
		switch scope {
		case tokens.ScopeOpenID:
			hasOpenIDScope = true
		case tokens.ScopeOfflineAccess, tokens.ScopeEmail, tokens.ScopeProfile, tokens.ScopeGroups, tokens.ScopeFederatedID:
		default:
			peerID, ok := tokens.ParseCrossClientScope(scope)
			if !ok {
				unrecognized = append(unrecognized, scope)
				continue
			}

			isTrusted, err := s.validateCrossClientTrust(r.Context(), clientID, peerID)
			if err != nil {
				return nil, "", newRedirectedErr(errServerError, "Internal server error.")
			}
			if !isTrusted {
				invalidScopes = append(invalidScopes, scope)
			}
		}
	}
	if !hasOpenIDScope {
		return nil, "", newRedirectedErr(errInvalidScope, `Missing required scope(s) ["openid"].`)
	}
	if len(unrecognized) > 0 {
		return nil, "", newRedirectedErr(errInvalidScope, "Unrecognized scope(s) %q", unrecognized)
	}
	if len(invalidScopes) > 0 {
		return nil, "", newRedirectedErr(errInvalidScope, "Client can't request scope(s) %q", invalidScopes)
	}

	var rt struct {
		code    bool
		idToken bool
		token   bool
	}

	for _, responseType := range responseTypes {
		switch responseType {
		case responseTypeCode:
			rt.code = true
		case responseTypeIDToken:
			rt.idToken = true
		case responseTypeToken:
			rt.token = true
		default:
			return nil, "", newRedirectedErr(errInvalidRequest, "Invalid response type %q", responseType)
		}

		if !s.supportedResponseTypes[responseType] {
			return nil, "", newRedirectedErr(errUnsupportedResponseType, "Unsupported response type %q", responseType)
		}
	}

	if len(responseTypes) == 0 {
		return nil, "", newRedirectedErr(errInvalidRequest, "No response_type provided")
	}

	if rt.token && !rt.code && !rt.idToken {
		// "token" can't be provided by its own.
		//
		// https://openid.net/specs/openid-connect-core-1_0.html#Authentication
		return nil, "", newRedirectedErr(errInvalidRequest, "Response type 'token' must be provided with type 'id_token' and/or 'code'")
	}
	if !rt.code {
		// Either "id_token token" or "id_token" has been provided which implies the
		// implicit flow. Implicit flow requires a nonce value.
		//
		// https://openid.net/specs/openid-connect-core-1_0.html#ImplicitAuthRequest
		if nonce == "" {
			return nil, "", newRedirectedErr(errInvalidRequest, "Response type 'token' requires a 'nonce' value.")
		}
	}
	if rt.token {
		if redirectURI == redirectURIOOB {
			return nil, "", newRedirectedErr(errInvalidRequest, "Cannot use response type 'token' with redirect_uri '%s'.", redirectURIOOB)
		}
	}

	prompt, err := ParsePrompt(q.Get("prompt"))
	if err != nil {
		return nil, "", newRedirectedErr(errInvalidRequest, "Invalid prompt parameter: %v", err)
	}

	// Parse max_age: -1 means not specified.
	maxAge := -1
	if maxAgeStr := q.Get("max_age"); maxAgeStr != "" {
		v, err := strconv.Atoi(maxAgeStr)
		if err != nil || v < 0 {
			return nil, "", newRedirectedErr(errInvalidRequest, "Invalid max_age value %q", maxAgeStr)
		}
		maxAge = v
	}

	// OIDC prompt=consent implies force approval.
	forceApproval := q.Get("approval_prompt") == "force" || prompt.Consent()

	// Validate id_token_hint if provided (OIDC Core 1.0 §3.1.2.1).
	var idTokenHintSubject string
	if hint := q.Get("id_token_hint"); hint != "" {
		idToken, err := s.validateIDTokenHint(ctx, hint)
		if err != nil {
			return nil, "", newRedirectedErr(errInvalidRequest, "Invalid id_token_hint.")
		}
		idTokenHintSubject = idToken.Subject
	}

	return &storage.AuthRequest{
		ID:                  storage.NewID(),
		ClientID:            client.ID,
		State:               state,
		Nonce:               nonce,
		ForceApprovalPrompt: forceApproval,
		Prompt:              prompt.String(),
		MaxAge:              maxAge,
		Scopes:              scopes,
		RedirectURI:         redirectURI,
		ResponseTypes:       responseTypes,
		ConnectorID:         connectorID,
		PKCE: storage.PKCE{
			CodeChallenge:       codeChallenge,
			CodeChallengeMethod: codeChallengeMethod,
		},
		HMACKey: storage.NewHMACKey(crypto.SHA256),
	}, idTokenHintSubject, nil
}

func validateRedirectURI(client storage.Client, redirectURI string) bool {
	// Allow named RedirectURIs for both public and non-public clients.
	// This is required make PKCE-enabled web apps work, when configured as public clients.
	for _, uri := range client.RedirectURIs {
		if redirectURI == uri {
			return true
		}
	}
	// For non-public clients or when RedirectURIs is set, we allow only explicitly named RedirectURIs.
	// Otherwise, we check below for special URIs used for desktop or mobile apps.
	if !client.Public || len(client.RedirectURIs) > 0 {
		return false
	}

	if redirectURI == redirectURIOOB || redirectURI == deviceCallbackURI {
		return true
	}

	// verify that the host is of form "http://localhost:(port)(path)", "http://localhost(path)" or numeric form like
	// "http://127.0.0.1:(port)(path)"
	u, err := url.Parse(redirectURI)
	if err != nil {
		return false
	}
	if u.Scheme != "http" {
		return false
	}
	return isHostLocal(u.Host)
}

func isHostLocal(host string) bool {
	if host == "localhost" || net.ParseIP(host).IsLoopback() {
		return true
	}

	host, _, err := net.SplitHostPort(host)
	if err != nil {
		return false
	}

	return host == "localhost" || net.ParseIP(host).IsLoopback()
}

func validateConnectorID(connectors []storage.Connector, connectorID string) bool {
	for _, c := range connectors {
		if c.ID == connectorID {
			return true
		}
	}
	return false
}

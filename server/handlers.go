package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/gorilla/mux"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

const (
	codeChallengeMethodPlain = "plain"
	codeChallengeMethodS256  = "S256"
)

func (s *Server) handlePublicKeys(w http.ResponseWriter, r *http.Request) {
	// TODO(ericchiang): Cache this.
	keys, err := s.storage.GetKeys()
	if err != nil {
		s.logger.Errorf("failed to get keys: %v", err)
		s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	if keys.SigningKeyPub == nil {
		s.logger.Errorf("No public keys found.")
		s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	jwks := jose.JSONWebKeySet{
		Keys: make([]jose.JSONWebKey, len(keys.VerificationKeys)+1),
	}
	jwks.Keys[0] = *keys.SigningKeyPub
	for i, verificationKey := range keys.VerificationKeys {
		jwks.Keys[i+1] = *verificationKey.PublicKey
	}

	data, err := json.MarshalIndent(jwks, "", "  ")
	if err != nil {
		s.logger.Errorf("failed to marshal discovery data: %v", err)
		s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}
	maxAge := keys.NextRotation.Sub(s.now())
	if maxAge < (time.Minute * 2) {
		maxAge = time.Minute * 2
	}

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
	GrantTypes        []string `json:"grant_types_supported"`
	ResponseTypes     []string `json:"response_types_supported"`
	Subjects          []string `json:"subject_types_supported"`
	IDTokenAlgs       []string `json:"id_token_signing_alg_values_supported"`
	CodeChallengeAlgs []string `json:"code_challenge_methods_supported"`
	Scopes            []string `json:"scopes_supported"`
	AuthMethods       []string `json:"token_endpoint_auth_methods_supported"`
	Claims            []string `json:"claims_supported"`
}

func (s *Server) discoveryHandler() (http.HandlerFunc, error) {
	d := discovery{
		Issuer:            s.issuerURL.String(),
		Auth:              s.absURL("/auth"),
		Token:             s.absURL("/token"),
		Keys:              s.absURL("/keys"),
		UserInfo:          s.absURL("/userinfo"),
		DeviceEndpoint:    s.absURL("/device/code"),
		Subjects:          []string{"public"},
		IDTokenAlgs:       []string{string(jose.RS256)},
		CodeChallengeAlgs: []string{codeChallengeMethodS256, codeChallengeMethodPlain},
		Scopes:            []string{"openid", "email", "groups", "profile", "offline_access"},
		AuthMethods:       []string{"client_secret_basic", "client_secret_post"},
		Claims: []string{
			"iss", "sub", "aud", "iat", "exp", "email", "email_verified",
			"locale", "name", "preferred_username", "at_hash",
		},
	}

	for responseType := range s.supportedResponseTypes {
		d.ResponseTypes = append(d.ResponseTypes, responseType)
	}
	sort.Strings(d.ResponseTypes)

	d.GrantTypes = s.supportedGrantTypes

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

// handleAuthorization handles the OAuth2 auth endpoint.
func (s *Server) handleAuthorization(w http.ResponseWriter, r *http.Request) {
	// Extract the arguments
	if err := r.ParseForm(); err != nil {
		s.logger.Errorf("Failed to parse arguments: %v", err)

		s.renderError(r, w, http.StatusBadRequest, err.Error())
		return
	}

	connectorID := r.Form.Get("connector_id")

	connectors, err := s.storage.ListConnectors()
	if err != nil {
		s.logger.Errorf("Failed to get list of connectors: %v", err)
		s.renderError(r, w, http.StatusInternalServerError, "Failed to retrieve connector list.")
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
	}

	connectorInfos := make([]connectorInfo, len(connectors))
	for index, conn := range connectors {
		connURL.Path = s.absPath("/auth", url.PathEscape(conn.ID))
		connectorInfos[index] = connectorInfo{
			ID:   conn.ID,
			Name: conn.Name,
			Type: conn.Type,
			URL:  template.URL(connURL.String()),
		}
	}

	if err := s.templates.login(r, w, connectorInfos); err != nil {
		s.logger.Errorf("Server template error: %v", err)
	}
}

func (s *Server) handleConnectorLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authReq, err := s.parseAuthorizationRequest(r)
	if err != nil {
		s.logger.Errorf("Failed to parse authorization request: %v", err)

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
		s.logger.Errorf("Failed to parse connector: %v", err)
		s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist")
		return
	}

	conn, err := s.getConnector(connID)
	if err != nil {
		s.logger.Errorf("Failed to get connector: %v", err)
		s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist")
		return
	}

	// Set the connector being used for the login.
	if authReq.ConnectorID != "" && authReq.ConnectorID != connID {
		s.logger.Errorf("Mismatched connector ID in auth request: %s vs %s",
			authReq.ConnectorID, connID)
		s.renderError(r, w, http.StatusBadRequest, "Bad connector ID")
		return
	}

	authReq.ConnectorID = connID

	// Actually create the auth request
	authReq.Expiry = s.now().Add(s.authRequestsValidFor)
	if err := s.storage.CreateAuthRequest(ctx, *authReq); err != nil {
		s.logger.Errorf("Failed to create authorization request: %v", err)
		s.renderError(r, w, http.StatusInternalServerError, "Failed to connect to the database.")
		return
	}

	scopes := parseScopes(authReq.Scopes)

	// Work out where the "Select another login method" link should go.
	backLink := ""
	if len(s.connectors) > 1 {
		backLinkURL := url.URL{
			Path:     s.absPath("/auth"),
			RawQuery: r.Form.Encode(),
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
			callbackURL, err := conn.LoginURL(scopes, s.absURL("/callback"), authReq.ID)
			if err != nil {
				s.logger.Errorf("Connector %q returned error when creating callback: %v", connID, err)
				s.renderError(r, w, http.StatusInternalServerError, "Login error.")
				return
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
				s.logger.Errorf("Creating SAML data: %v", err)
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

	authReq, err := s.storage.GetAuthRequest(authID)
	if err != nil {
		if err == storage.ErrNotFound {
			s.logger.Errorf("Invalid 'state' parameter provided: %v", err)
			s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
			return
		}
		s.logger.Errorf("Failed to get auth request: %v", err)
		s.renderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}

	connID, err := url.PathUnescape(mux.Vars(r)["connector"])
	if err != nil {
		s.logger.Errorf("Failed to parse connector: %v", err)
		s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist")
		return
	} else if connID != "" && connID != authReq.ConnectorID {
		s.logger.Errorf("Connector mismatch: authentication started with id %q, but password login for id %q was triggered", authReq.ConnectorID, connID)
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	conn, err := s.getConnector(authReq.ConnectorID)
	if err != nil {
		s.logger.Errorf("Failed to get connector with id %q : %v", authReq.ConnectorID, err)
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	pwConn, ok := conn.Connector.(connector.PasswordConnector)
	if !ok {
		s.logger.Errorf("Expected password connector in handlePasswordLogin(), but got %v", pwConn)
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	switch r.Method {
	case http.MethodGet:
		if err := s.templates.password(r, w, r.URL.String(), "", usernamePrompt(pwConn), false, backLink); err != nil {
			s.logger.Errorf("Server template error: %v", err)
		}
	case http.MethodPost:
		username := r.FormValue("login")
		password := r.FormValue("password")
		scopes := parseScopes(authReq.Scopes)

		identity, ok, err := pwConn.Login(ctx, scopes, username, password)
		if err != nil {
			s.logger.Errorf("Failed to login user: %v", err)
			s.renderError(r, w, http.StatusInternalServerError, fmt.Sprintf("Login error: %v", err))
			return
		}
		if !ok {
			if err := s.templates.password(r, w, r.URL.String(), username, usernamePrompt(pwConn), true, backLink); err != nil {
				s.logger.Errorf("Server template error: %v", err)
			}
			s.logger.Errorf("Failed login attempt for user: %q. Invalid credentials.", username)
			return
		}
		redirectURL, canSkipApproval, err := s.finalizeLogin(ctx, identity, authReq, conn.Connector)
		if err != nil {
			s.logger.Errorf("Failed to finalize login: %v", err)
			s.renderError(r, w, http.StatusInternalServerError, "Login error.")
			return
		}

		if canSkipApproval {
			authReq, err = s.storage.GetAuthRequest(authReq.ID)
			if err != nil {
				s.logger.Errorf("Failed to get finalized auth request: %v", err)
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

	authReq, err := s.storage.GetAuthRequest(authID)
	if err != nil {
		if err == storage.ErrNotFound {
			s.logger.Errorf("Invalid 'state' parameter provided: %v", err)
			s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
			return
		}
		s.logger.Errorf("Failed to get auth request: %v", err)
		s.renderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}

	connID, err := url.PathUnescape(mux.Vars(r)["connector"])
	if err != nil {
		s.logger.Errorf("Failed to get connector with id %q : %v", authReq.ConnectorID, err)
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	} else if connID != "" && connID != authReq.ConnectorID {
		s.logger.Errorf("Connector mismatch: authentication started with id %q, but callback for id %q was triggered", authReq.ConnectorID, connID)
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	conn, err := s.getConnector(authReq.ConnectorID)
	if err != nil {
		s.logger.Errorf("Failed to get connector with id %q : %v", authReq.ConnectorID, err)
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	var identity connector.Identity
	switch conn := conn.Connector.(type) {
	case connector.CallbackConnector:
		if r.Method != http.MethodGet {
			s.logger.Errorf("SAML request mapped to OAuth2 connector")
			s.renderError(r, w, http.StatusBadRequest, "Invalid request")
			return
		}
		identity, err = conn.HandleCallback(parseScopes(authReq.Scopes), r)
	case connector.SAMLConnector:
		if r.Method != http.MethodPost {
			s.logger.Errorf("OAuth2 request mapped to SAML connector")
			s.renderError(r, w, http.StatusBadRequest, "Invalid request")
			return
		}
		identity, err = conn.HandlePOST(parseScopes(authReq.Scopes), r.PostFormValue("SAMLResponse"), authReq.ID)
	default:
		s.renderError(r, w, http.StatusInternalServerError, "Requested resource does not exist.")
		return
	}

	if err != nil {
		s.logger.Errorf("Failed to authenticate: %v", err)
		s.renderError(r, w, http.StatusInternalServerError, fmt.Sprintf("Failed to authenticate: %v", err))
		return
	}

	redirectURL, canSkipApproval, err := s.finalizeLogin(ctx, identity, authReq, conn.Connector)
	if err != nil {
		s.logger.Errorf("Failed to finalize login: %v", err)
		s.renderError(r, w, http.StatusInternalServerError, "Login error.")
		return
	}

	if canSkipApproval {
		authReq, err = s.storage.GetAuthRequest(authReq.ID)
		if err != nil {
			s.logger.Errorf("Failed to get finalized auth request: %v", err)
			s.renderError(r, w, http.StatusInternalServerError, "Login error.")
			return
		}
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
		return a, nil
	}
	if err := s.storage.UpdateAuthRequest(authReq.ID, updater); err != nil {
		return "", false, fmt.Errorf("failed to update auth request: %v", err)
	}

	email := claims.Email
	if !claims.EmailVerified {
		email += " (unverified)"
	}

	s.logger.Infof("login successful: connector %q, username=%q, preferred_username=%q, email=%q, groups=%q",
		authReq.ConnectorID, claims.Username, claims.PreferredUsername, email, claims.Groups)

	// we can skip the redirect to /approval and go ahead and send code if it's not required
	if s.skipApproval && !authReq.ForceApprovalPrompt {
		return "", true, nil
	}

	// an HMAC is used here to ensure that the request ID is unpredictable, ensuring that an attacker who intercepted the original
	// flow would be unable to poll for the result at the /approval endpoint
	h := hmac.New(sha256.New, authReq.HMACKey)
	h.Write([]byte(authReq.ID))
	mac := h.Sum(nil)

	returnURL := path.Join(s.issuerURL.Path, "/approval") + "?req=" + authReq.ID + "&hmac=" + base64.RawURLEncoding.EncodeToString(mac)
	_, ok := conn.(connector.RefreshConnector)
	if !ok {
		return returnURL, false, nil
	}

	offlineAccessRequested := false
	for _, scope := range authReq.Scopes {
		if scope == scopeOfflineAccess {
			offlineAccessRequested = true
			break
		}
	}
	if !offlineAccessRequested {
		return returnURL, false, nil
	}

	// Try to retrieve an existing OfflineSession object for the corresponding user.
	session, err := s.storage.GetOfflineSessions(identity.UserID, authReq.ConnectorID)
	if err != nil {
		if err != storage.ErrNotFound {
			s.logger.Errorf("failed to get offline session: %v", err)
			return "", false, err
		}
		offlineSessions := storage.OfflineSessions{
			UserID:        identity.UserID,
			ConnID:        authReq.ConnectorID,
			Refresh:       make(map[string]*storage.RefreshTokenRef),
			ConnectorData: identity.ConnectorData,
		}

		// Create a new OfflineSession object for the user and add a reference object for
		// the newly received refreshtoken.
		if err := s.storage.CreateOfflineSessions(ctx, offlineSessions); err != nil {
			s.logger.Errorf("failed to create offline session: %v", err)
			return "", false, err
		}

		return returnURL, false, nil
	}

	// Update existing OfflineSession obj with new RefreshTokenRef.
	if err := s.storage.UpdateOfflineSessions(session.UserID, session.ConnID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
		if len(identity.ConnectorData) > 0 {
			old.ConnectorData = identity.ConnectorData
		}
		return old, nil
	}); err != nil {
		s.logger.Errorf("failed to update offline session: %v", err)
		return "", false, err
	}

	return returnURL, false, nil
}

func (s *Server) handleApproval(w http.ResponseWriter, r *http.Request) {
	macEncoded := r.FormValue("hmac")
	if macEncoded == "" {
		s.renderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}
	mac, err := base64.RawURLEncoding.DecodeString(macEncoded)
	if err != nil {
		s.renderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}

	authReq, err := s.storage.GetAuthRequest(r.FormValue("req"))
	if err != nil {
		s.logger.Errorf("Failed to get auth request: %v", err)
		s.renderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}
	if !authReq.LoggedIn {
		s.logger.Errorf("Auth request does not have an identity for approval")
		s.renderError(r, w, http.StatusInternalServerError, "Login process not yet finalized.")
		return
	}

	// build expected hmac with secret key
	h := hmac.New(sha256.New, authReq.HMACKey)
	h.Write([]byte(authReq.ID))
	expectedMAC := h.Sum(nil)
	// constant time comparison
	if !hmac.Equal(mac, expectedMAC) {
		s.renderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}

	switch r.Method {
	case http.MethodGet:
		client, err := s.storage.GetClient(authReq.ClientID)
		if err != nil {
			s.logger.Errorf("Failed to get client %q: %v", authReq.ClientID, err)
			s.renderError(r, w, http.StatusInternalServerError, "Failed to retrieve client.")
			return
		}
		if err := s.templates.approval(r, w, authReq.ID, authReq.Claims.Username, client.Name, authReq.Scopes); err != nil {
			s.logger.Errorf("Server template error: %v", err)
		}
	case http.MethodPost:
		if r.FormValue("approval") != "approve" {
			s.renderError(r, w, http.StatusInternalServerError, "Approval rejected.")
			return
		}
		s.sendCodeResponse(w, r, authReq)
	}
}

func (s *Server) sendCodeResponse(w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest) {
	ctx := r.Context()
	if s.now().After(authReq.Expiry) {
		s.renderError(r, w, http.StatusBadRequest, "User session has expired.")
		return
	}

	if err := s.storage.DeleteAuthRequest(authReq.ID); err != nil {
		if err != storage.ErrNotFound {
			s.logger.Errorf("Failed to delete authorization request: %v", err)
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
			}
			if err := s.storage.CreateAuthCode(ctx, code); err != nil {
				s.logger.Errorf("Failed to create auth code: %v", err)
				s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
				return
			}

			// Implicit and hybrid flows that try to use the OOB redirect URI are
			// rejected earlier. If we got here we're using the code flow.
			if authReq.RedirectURI == redirectURIOOB {
				if err := s.templates.oob(r, w, code.ID); err != nil {
					s.logger.Errorf("Server template error: %v", err)
				}
				return
			}
		case responseTypeToken:
			implicitOrHybrid = true
		case responseTypeIDToken:
			implicitOrHybrid = true
			var err error

			accessToken, _, err = s.newAccessToken(authReq.ClientID, authReq.Claims, authReq.Scopes, authReq.Nonce, authReq.ConnectorID)
			if err != nil {
				s.logger.Errorf("failed to create new access token: %v", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				return
			}

			idToken, idTokenExpiry, err = s.newIDToken(authReq.ClientID, authReq.Claims, authReq.Scopes, authReq.Nonce, accessToken, code.ID, authReq.ConnectorID)
			if err != nil {
				s.logger.Errorf("failed to create ID token: %v", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				return
			}
		}
	}

	if implicitOrHybrid {
		v := url.Values{}
		v.Set("access_token", accessToken)
		v.Set("token_type", "bearer")
		v.Set("state", authReq.State)
		if idToken != "" {
			v.Set("id_token", idToken)
			// The hybrid flow with only "code token" or "code id_token" doesn't return an
			// "expires_in" value. If "code" wasn't provided, indicating the implicit flow,
			// don't add it.
			//
			// https://openid.net/specs/openid-connect-core-1_0.html#HybridAuthResponse
			if code.ID == "" {
				v.Set("expires_in", strconv.Itoa(int(idTokenExpiry.Sub(s.now()).Seconds())))
			}
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

func (s *Server) withClientFromStorage(w http.ResponseWriter, r *http.Request, handler func(http.ResponseWriter, *http.Request, storage.Client)) {
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

	client, err := s.storage.GetClient(clientID)
	if err != nil {
		if err != storage.ErrNotFound {
			s.logger.Errorf("failed to get client: %v", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		} else {
			s.tokenErrHelper(w, errInvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
		}
		return
	}

	if subtle.ConstantTimeCompare([]byte(client.Secret), []byte(clientSecret)) != 1 {
		if clientSecret == "" {
			s.logger.Infof("missing client_secret on token request for client: %s", client.ID)
		} else {
			s.logger.Infof("invalid client_secret on token request for client: %s", client.ID)
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
		s.logger.Errorf("Could not parse request body: %v", err)
		s.tokenErrHelper(w, errInvalidRequest, "", http.StatusBadRequest)
		return
	}

	grantType := r.PostFormValue("grant_type")
	if !contains(s.supportedGrantTypes, grantType) {
		s.logger.Errorf("unsupported grant type: %v", grantType)
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
	redirectURI := r.PostFormValue("redirect_uri")

	if code == "" {
		s.tokenErrHelper(w, errInvalidRequest, `Required param: code.`, http.StatusBadRequest)
		return
	}

	authCode, err := s.storage.GetAuthCode(code)
	if err != nil || s.now().After(authCode.Expiry) || authCode.ClientID != client.ID {
		if err != storage.ErrNotFound {
			s.logger.Errorf("failed to get auth code: %v", err)
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
			s.logger.Error(err)
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
	accessToken, _, err := s.newAccessToken(client.ID, authCode.Claims, authCode.Scopes, authCode.Nonce, authCode.ConnectorID)
	if err != nil {
		s.logger.Errorf("failed to create new access token: %v", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return nil, err
	}

	idToken, expiry, err := s.newIDToken(client.ID, authCode.Claims, authCode.Scopes, authCode.Nonce, accessToken, authCode.ID, authCode.ConnectorID)
	if err != nil {
		s.logger.Errorf("failed to create ID token: %v", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return nil, err
	}

	if err := s.storage.DeleteAuthCode(authCode.ID); err != nil {
		s.logger.Errorf("failed to delete auth code: %v", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return nil, err
	}

	reqRefresh := func() bool {
		// Ensure the connector supports refresh tokens.
		//
		// Connectors like `saml` do not implement RefreshConnector.
		conn, err := s.getConnector(authCode.ConnectorID)
		if err != nil {
			s.logger.Errorf("connector with ID %q not found: %v", authCode.ConnectorID, err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
			return false
		}

		_, ok := conn.Connector.(connector.RefreshConnector)
		if !ok {
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
			s.logger.Errorf("failed to marshal refresh token: %v", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
			return nil, err
		}

		if err := s.storage.CreateRefresh(ctx, refresh); err != nil {
			s.logger.Errorf("failed to create refresh token: %v", err)
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
				if err := s.storage.DeleteRefresh(refresh.ID); err != nil {
					s.logger.Errorf("failed to delete refresh token: %v", err)
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
		if session, err := s.storage.GetOfflineSessions(refresh.Claims.UserID, refresh.ConnectorID); err != nil {
			if err != storage.ErrNotFound {
				s.logger.Errorf("failed to get offline session: %v", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return nil, err
			}
			offlineSessions := storage.OfflineSessions{
				UserID:  refresh.Claims.UserID,
				ConnID:  refresh.ConnectorID,
				Refresh: make(map[string]*storage.RefreshTokenRef),
			}
			offlineSessions.Refresh[tokenRef.ClientID] = &tokenRef

			// Create a new OfflineSession object for the user and add a reference object for
			// the newly received refreshtoken.
			if err := s.storage.CreateOfflineSessions(ctx, offlineSessions); err != nil {
				s.logger.Errorf("failed to create offline session: %v", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return nil, err
			}
		} else {
			if oldTokenRef, ok := session.Refresh[tokenRef.ClientID]; ok {
				// Delete old refresh token from storage.
				if err := s.storage.DeleteRefresh(oldTokenRef.ID); err != nil && err != storage.ErrNotFound {
					s.logger.Errorf("failed to delete refresh token: %v", err)
					s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
					deleteToken = true
					return nil, err
				}
			}

			// Update existing OfflineSession obj with new RefreshTokenRef.
			if err := s.storage.UpdateOfflineSessions(session.UserID, session.ConnID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
				old.Refresh[tokenRef.ClientID] = &tokenRef
				return old, nil
			}); err != nil {
				s.logger.Errorf("failed to update offline session: %v", err)
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

	verifier := oidc.NewVerifier(s.issuerURL.String(), &storageKeySet{s.storage}, &oidc.Config{SkipClientIDCheck: true})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		s.tokenErrHelper(w, errAccessDenied, err.Error(), http.StatusForbidden)
		return
	}

	var claims json.RawMessage
	if err := idToken.Claims(&claims); err != nil {
		s.tokenErrHelper(w, errServerError, err.Error(), http.StatusInternalServerError)
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

			isTrusted, err := s.validateCrossClientTrust(client.ID, peerID)
			if err != nil {
				s.tokenErrHelper(w, errInvalidClient, fmt.Sprintf("Error validating cross client trust %v.", err), http.StatusBadRequest)
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
	conn, err := s.getConnector(connID)
	if err != nil {
		s.tokenErrHelper(w, errInvalidRequest, "Requested connector does not exist.", http.StatusBadRequest)
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
		s.logger.Errorf("Failed to login user: %v", err)
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

	accessToken, _, err := s.newAccessToken(client.ID, claims, scopes, nonce, connID)
	if err != nil {
		s.logger.Errorf("password grant failed to create new access token: %v", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	idToken, expiry, err := s.newIDToken(client.ID, claims, scopes, nonce, accessToken, "", connID)
	if err != nil {
		s.logger.Errorf("password grant failed to create new ID token: %v", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	reqRefresh := func() bool {
		// Ensure the connector supports refresh tokens.
		//
		// Connectors like `saml` do not implement RefreshConnector.
		_, ok := conn.Connector.(connector.RefreshConnector)
		if !ok {
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
			s.logger.Errorf("failed to marshal refresh token: %v", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
			return
		}

		if err := s.storage.CreateRefresh(ctx, refresh); err != nil {
			s.logger.Errorf("failed to create refresh token: %v", err)
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
				if err := s.storage.DeleteRefresh(refresh.ID); err != nil {
					s.logger.Errorf("failed to delete refresh token: %v", err)
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
		if session, err := s.storage.GetOfflineSessions(refresh.Claims.UserID, refresh.ConnectorID); err != nil {
			if err != storage.ErrNotFound {
				s.logger.Errorf("failed to get offline session: %v", err)
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
				s.logger.Errorf("failed to create offline session: %v", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return
			}
		} else {
			if oldTokenRef, ok := session.Refresh[tokenRef.ClientID]; ok {
				// Delete old refresh token from storage.
				if err := s.storage.DeleteRefresh(oldTokenRef.ID); err != nil {
					if err == storage.ErrNotFound {
						s.logger.Warnf("database inconsistent, refresh token missing: %v", oldTokenRef.ID)
					} else {
						s.logger.Errorf("failed to delete refresh token: %v", err)
						s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
						deleteToken = true
						return
					}
				}
			}

			// Update existing OfflineSession obj with new RefreshTokenRef.
			if err := s.storage.UpdateOfflineSessions(session.UserID, session.ConnID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
				old.Refresh[tokenRef.ClientID] = &tokenRef
				old.ConnectorData = identity.ConnectorData
				return old, nil
			}); err != nil {
				s.logger.Errorf("failed to update offline session: %v", err)
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
		s.logger.Errorf("could not parse request body: %v", err)
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

	conn, err := s.getConnector(connID)
	if err != nil {
		s.logger.Errorf("failed to get connector: %v", err)
		s.tokenErrHelper(w, errInvalidRequest, "Requested connector does not exist.", http.StatusBadRequest)
		return
	}
	teConn, ok := conn.Connector.(connector.TokenIdentityConnector)
	if !ok {
		s.logger.Errorf("connector doesn't implement token exchange: %v", connID)
		s.tokenErrHelper(w, errInvalidRequest, "Requested connector does not exist.", http.StatusBadRequest)
		return
	}
	identity, err := teConn.TokenIdentity(ctx, subjectTokenType, subjectToken)
	if err != nil {
		s.logger.Errorf("failed to verify subject token: %v", err)
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
		resp.AccessToken, expiry, err = s.newIDToken(client.ID, claims, scopes, "", "", "", connID)
	case tokenTypeAccess:
		resp.AccessToken, expiry, err = s.newAccessToken(client.ID, claims, scopes, "", connID)
	default:
		s.tokenErrHelper(w, errRequestNotSupported, "Invalid requested_token_type.", http.StatusBadRequest)
		return
	}
	if err != nil {
		s.logger.Errorf("token exchange failed to create new %v token: %v", requestedTokenType, err)
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
		s.logger.Errorf("failed to marshal access token response: %v", err)
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
		s.logger.Errorf("server template error: %v", err)
	}
}

func (s *Server) tokenErrHelper(w http.ResponseWriter, typ string, description string, statusCode int) {
	if err := tokenErr(w, typ, description, statusCode); err != nil {
		s.logger.Errorf("token error response: %v", err)
	}
}

// Check for username prompt override from connector. Defaults to "Username".
func usernamePrompt(conn connector.PasswordConnector) string {
	if attr := conn.Prompt(); attr != "" {
		return attr
	}
	return "Username"
}

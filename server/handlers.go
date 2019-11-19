package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	oidc "github.com/coreos/go-oidc"
	"github.com/gorilla/mux"
	jose "gopkg.in/square/go-jose.v2"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

// newHealthChecker returns the healthz handler. The handler runs until the
// provided context is canceled.
func (s *Server) newHealthChecker(ctx context.Context) http.Handler {
	h := &healthChecker{s: s}

	// Perform one health check synchronously so the returned handler returns
	// valid data immediately.
	h.runHealthCheck()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second * 15):
			}
			h.runHealthCheck()
		}
	}()
	return h
}

// healthChecker periodically performs health checks on server dependenices.
// Currently, it only checks that the storage layer is avialable.
type healthChecker struct {
	s *Server

	// Result of the last health check: any error and the amount of time it took
	// to query the storage.
	mu sync.RWMutex
	// Guarded by the mutex
	err    error
	passed time.Duration
}

// runHealthCheck performs a single health check and makes the result available
// for any clients performing and HTTP request against the healthChecker.
func (h *healthChecker) runHealthCheck() {
	t := h.s.now()
	err := checkStorageHealth(h.s.storage, h.s.now)
	passed := h.s.now().Sub(t)
	if err != nil {
		h.s.logger.Errorf("Storage health check failed: %v", err)
	}

	// Make sure to only hold the mutex to access the fields, and not while
	// we're querying the storage object.
	h.mu.Lock()
	h.err = err
	h.passed = passed
	h.mu.Unlock()
}

func checkStorageHealth(s storage.Storage, now func() time.Time) error {
	a := storage.AuthRequest{
		ID:       storage.NewID(),
		ClientID: storage.NewID(),

		// Set a short expiry so if the delete fails this will be cleaned up quickly by garbage collection.
		Expiry: now().Add(time.Minute),
	}

	if err := s.CreateAuthRequest(a); err != nil {
		return fmt.Errorf("create auth request: %v", err)
	}
	if err := s.DeleteAuthRequest(a.ID); err != nil {
		return fmt.Errorf("delete auth request: %v", err)
	}
	return nil
}

func (h *healthChecker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	err := h.err
	t := h.passed
	h.mu.RUnlock()

	if err != nil {
		h.s.renderError(r, w, http.StatusInternalServerError, "Health check failed.")
		return
	}
	fmt.Fprintf(w, "Health check passed in %s", t)
}

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
	Issuer        string   `json:"issuer"`
	Auth          string   `json:"authorization_endpoint"`
	Token         string   `json:"token_endpoint"`
	Keys          string   `json:"jwks_uri"`
	UserInfo      string   `json:"userinfo_endpoint"`
	ResponseTypes []string `json:"response_types_supported"`
	Subjects      []string `json:"subject_types_supported"`
	IDTokenAlgs   []string `json:"id_token_signing_alg_values_supported"`
	Scopes        []string `json:"scopes_supported"`
	AuthMethods   []string `json:"token_endpoint_auth_methods_supported"`
	Claims        []string `json:"claims_supported"`
}

func (s *Server) discoveryHandler() (http.HandlerFunc, error) {
	d := discovery{
		Issuer:      s.issuerURL.String(),
		Auth:        s.absURL("/auth"),
		Token:       s.absURL("/token"),
		Keys:        s.absURL("/keys"),
		UserInfo:    s.absURL("/userinfo"),
		Subjects:    []string{"public"},
		IDTokenAlgs: []string{string(jose.RS256)},
		Scopes:      []string{"openid", "email", "groups", "profile", "offline_access"},
		AuthMethods: []string{"client_secret_basic"},
		Claims: []string{
			"aud", "email", "email_verified", "exp",
			"iat", "iss", "locale", "name", "sub",
		},
	}

	for responseType := range s.supportedResponseTypes {
		d.ResponseTypes = append(d.ResponseTypes, responseType)
	}
	sort.Strings(d.ResponseTypes)

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
	authReq, err := s.parseAuthorizationRequest(r)
	if err != nil {
		s.logger.Errorf("Failed to parse authorization request: %v", err)
		status := http.StatusInternalServerError

		// If this is an authErr, let's let it handle the error, or update the HTTP
		// status code
		if err, ok := err.(*authErr); ok {
			if handler, ok := err.Handle(); ok {
				// client_id and redirect_uri checked out and we can redirect back to
				// the client with the error.
				handler.ServeHTTP(w, r)
				return
			}
			status = err.Status()
		}

		s.renderError(r, w, status, err.Error())
		return
	}

	// TODO(ericchiang): Create this authorization request later in the login flow
	// so users don't hit "not found" database errors if they wait at the login
	// screen too long.
	//
	// See: https://github.com/dexidp/dex/issues/646
	authReq.Expiry = s.now().Add(s.authRequestsValidFor)
	if err := s.storage.CreateAuthRequest(*authReq); err != nil {
		s.logger.Errorf("Failed to create authorization request: %v", err)
		s.renderError(r, w, http.StatusInternalServerError, "Failed to connect to the database.")
		return
	}

	connectors, err := s.storage.ListConnectors()
	if err != nil {
		s.logger.Errorf("Failed to get list of connectors: %v", err)
		s.renderError(r, w, http.StatusInternalServerError, "Failed to retrieve connector list.")
		return
	}

	// Redirect if a client chooses a specific connector_id
	if authReq.ConnectorID != "" {
		for _, c := range connectors {
			if c.ID == authReq.ConnectorID {
				http.Redirect(w, r, s.absPath("/auth", c.ID)+"?req="+authReq.ID, http.StatusFound)
				return
			}
		}
		s.tokenErrHelper(w, errInvalidConnectorID, "Connector ID does not match a valid Connector", http.StatusNotFound)
		return
	}

	if len(connectors) == 1 && !s.alwaysShowLogin {
		for _, c := range connectors {
			// TODO(ericchiang): Make this pass on r.URL.RawQuery and let something latter
			// on create the auth request.
			http.Redirect(w, r, s.absPath("/auth", c.ID)+"?req="+authReq.ID, http.StatusFound)
			return
		}
	}

	connectorInfos := make([]connectorInfo, len(connectors))
	i := 0
	for _, conn := range connectors {
		connectorInfos[i] = connectorInfo{
			ID:   conn.ID,
			Name: conn.Name,
			// TODO(ericchiang): Make this pass on r.URL.RawQuery and let something latter
			// on create the auth request.
			URL: s.absPath("/auth", conn.ID) + "?req=" + authReq.ID,
		}
		i++
	}

	if err := s.templates.login(r, w, connectorInfos, r.URL.Path); err != nil {
		s.logger.Errorf("Server template error: %v", err)
	}
}

func (s *Server) handleConnectorLogin(w http.ResponseWriter, r *http.Request) {
	connID := mux.Vars(r)["connector"]
	conn, err := s.getConnector(connID)
	if err != nil {
		s.logger.Errorf("Failed to create authorization request: %v", err)
		s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist")
		return
	}

	authReqID := r.FormValue("req")

	authReq, err := s.storage.GetAuthRequest(authReqID)
	if err != nil {
		s.logger.Errorf("Failed to get auth request: %v", err)
		if err == storage.ErrNotFound {
			s.renderError(r, w, http.StatusBadRequest, "Login session expired.")
		} else {
			s.renderError(r, w, http.StatusInternalServerError, "Database error.")
		}
		return
	}

	// Set the connector being used for the login.
	if authReq.ConnectorID != connID {
		updater := func(a storage.AuthRequest) (storage.AuthRequest, error) {
			a.ConnectorID = connID
			return a, nil
		}
		if err := s.storage.UpdateAuthRequest(authReqID, updater); err != nil {
			s.logger.Errorf("Failed to set connector ID on auth request: %v", err)
			s.renderError(r, w, http.StatusInternalServerError, "Database error.")
			return
		}
	}

	scopes := parseScopes(authReq.Scopes)
	showBacklink := len(s.connectors) > 1

	switch r.Method {
	case http.MethodGet:
		switch conn := conn.Connector.(type) {
		case connector.CallbackConnector:
			// Use the auth request ID as the "state" token.
			//
			// TODO(ericchiang): Is this appropriate or should we also be using a nonce?
			callbackURL, err := conn.LoginURL(scopes, s.absURL("/callback"), authReqID)
			if err != nil {
				s.logger.Errorf("Connector %q returned error when creating callback: %v", connID, err)
				s.renderError(r, w, http.StatusInternalServerError, "Login error.")
				return
			}
			http.Redirect(w, r, callbackURL, http.StatusFound)
		case connector.PasswordConnector:
			if err := s.templates.password(r, w, r.URL.String(), "", usernamePrompt(conn), false, showBacklink, r.URL.Path); err != nil {
				s.logger.Errorf("Server template error: %v", err)
			}
		case connector.SAMLConnector:
			action, value, err := conn.POSTData(scopes, authReqID)
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
			  </html>`, action, value, authReqID)
		default:
			s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
		}
	case http.MethodPost:
		passwordConnector, ok := conn.Connector.(connector.PasswordConnector)
		if !ok {
			s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
			return
		}

		username := r.FormValue("login")
		password := r.FormValue("password")

		identity, ok, err := passwordConnector.Login(r.Context(), scopes, username, password)
		if err != nil {
			s.logger.Errorf("Failed to login user: %v", err)
			s.renderError(r, w, http.StatusInternalServerError, fmt.Sprintf("Login error: %v", err))
			return
		}
		if !ok {
			if err := s.templates.password(r, w, r.URL.String(), username, usernamePrompt(passwordConnector), true, showBacklink, r.URL.Path); err != nil {
				s.logger.Errorf("Server template error: %v", err)
			}
			return
		}
		redirectURL, err := s.finalizeLogin(identity, authReq, conn.Connector)
		if err != nil {
			s.logger.Errorf("Failed to finalize login: %v", err)
			s.renderError(r, w, http.StatusInternalServerError, "Login error.")
			return
		}

		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	default:
		s.renderError(r, w, http.StatusBadRequest, "Unsupported request method.")
	}
}

func (s *Server) handleConnectorCallback(w http.ResponseWriter, r *http.Request) {
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

	if connID := mux.Vars(r)["connector"]; connID != "" && connID != authReq.ConnectorID {
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

	redirectURL, err := s.finalizeLogin(identity, authReq, conn.Connector)
	if err != nil {
		s.logger.Errorf("Failed to finalize login: %v", err)
		s.renderError(r, w, http.StatusInternalServerError, "Login error.")
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// finalizeLogin associates the user's identity with the current AuthRequest, then returns
// the approval page's path.
func (s *Server) finalizeLogin(identity connector.Identity, authReq storage.AuthRequest, conn connector.Connector) (string, error) {
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
		return "", fmt.Errorf("failed to update auth request: %v", err)
	}

	email := claims.Email
	if !claims.EmailVerified {
		email = email + " (unverified)"
	}

	s.logger.Infof("login successful: connector %q, username=%q, preferred_username=%q, email=%q, groups=%q",
		authReq.ConnectorID, claims.Username, claims.PreferredUsername, email, claims.Groups)

	returnURL := path.Join(s.issuerURL.Path, "/approval") + "?req=" + authReq.ID
	_, ok := conn.(connector.RefreshConnector)
	if !ok {
		return returnURL, nil
	}

	// Try to retrieve an existing OfflineSession object for the corresponding user.
	if session, err := s.storage.GetOfflineSessions(identity.UserID, authReq.ConnectorID); err != nil {
		if err != storage.ErrNotFound {
			s.logger.Errorf("failed to get offline session: %v", err)
			return "", err
		}
		offlineSessions := storage.OfflineSessions{
			UserID:        identity.UserID,
			ConnID:        authReq.ConnectorID,
			Refresh:       make(map[string]*storage.RefreshTokenRef),
			ConnectorData: identity.ConnectorData,
		}

		// Create a new OfflineSession object for the user and add a reference object for
		// the newly received refreshtoken.
		if err := s.storage.CreateOfflineSessions(offlineSessions); err != nil {
			s.logger.Errorf("failed to create offline session: %v", err)
			return "", err
		}
	} else {
		// Update existing OfflineSession obj with new RefreshTokenRef.
		if err := s.storage.UpdateOfflineSessions(session.UserID, session.ConnID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
			if len(identity.ConnectorData) > 0 {
				old.ConnectorData = identity.ConnectorData
			}
			return old, nil
		}); err != nil {
			s.logger.Errorf("failed to update offline session: %v", err)
			return "", err
		}
	}

	return returnURL, nil
}

func (s *Server) handleApproval(w http.ResponseWriter, r *http.Request) {
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

	switch r.Method {
	case http.MethodGet:
		if s.skipApproval {
			s.sendCodeResponse(w, r, authReq)
			return
		}
		client, err := s.storage.GetClient(authReq.ClientID)
		if err != nil {
			s.logger.Errorf("Failed to get client %q: %v", authReq.ClientID, err)
			s.renderError(r, w, http.StatusInternalServerError, "Failed to retrieve client.")
			return
		}
		if err := s.templates.approval(r, w, authReq.ID, authReq.Claims.Username, client.Name, authReq.Scopes, r.URL.Path); err != nil {
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
			}
			if err := s.storage.CreateAuthCode(code); err != nil {
				s.logger.Errorf("Failed to create auth code: %v", err)
				s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
				return
			}

			// Implicit and hybrid flows that try to use the OOB redirect URI are
			// rejected earlier. If we got here we're using the code flow.
			if authReq.RedirectURI == redirectURIOOB {
				if err := s.templates.oob(r, w, code.ID, r.URL.Path); err != nil {
					s.logger.Errorf("Server template error: %v", err)
				}
				return
			}
		case responseTypeToken:
			implicitOrHybrid = true
		case responseTypeIDToken:
			implicitOrHybrid = true
			var err error

			accessToken, err = s.newAccessToken(authReq.ClientID, authReq.Claims, authReq.Scopes, authReq.Nonce, authReq.ConnectorID)
			if err != nil {
				s.logger.Errorf("failed to create new access token: %v", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				return
			}

			idToken, idTokenExpiry, err = s.newIDToken(authReq.ClientID, authReq.Claims, authReq.Scopes, authReq.Nonce, accessToken, authReq.ConnectorID)
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

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
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
	if client.Secret != clientSecret {
		s.tokenErrHelper(w, errInvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
		return
	}

	grantType := r.PostFormValue("grant_type")
	switch grantType {
	case grantTypeAuthorizationCode:
		s.handleAuthCode(w, r, client)
	case grantTypeRefreshToken:
		s.handleRefreshToken(w, r, client)
	default:
		s.tokenErrHelper(w, errInvalidGrant, "", http.StatusBadRequest)
	}
}

// handle an access token request https://tools.ietf.org/html/rfc6749#section-4.1.3
func (s *Server) handleAuthCode(w http.ResponseWriter, r *http.Request, client storage.Client) {
	code := r.PostFormValue("code")
	redirectURI := r.PostFormValue("redirect_uri")

	authCode, err := s.storage.GetAuthCode(code)
	if err != nil || s.now().After(authCode.Expiry) || authCode.ClientID != client.ID {
		if err != storage.ErrNotFound {
			s.logger.Errorf("failed to get auth code: %v", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		} else {
			s.tokenErrHelper(w, errInvalidRequest, "Invalid or expired code parameter.", http.StatusBadRequest)
		}
		return
	}

	if authCode.RedirectURI != redirectURI {
		s.tokenErrHelper(w, errInvalidRequest, "redirect_uri did not match URI from initial request.", http.StatusBadRequest)
		return
	}

	accessToken, err := s.newAccessToken(client.ID, authCode.Claims, authCode.Scopes, authCode.Nonce, authCode.ConnectorID)
	if err != nil {
		s.logger.Errorf("failed to create new access token: %v", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	idToken, expiry, err := s.newIDToken(client.ID, authCode.Claims, authCode.Scopes, authCode.Nonce, accessToken, authCode.ConnectorID)
	if err != nil {
		s.logger.Errorf("failed to create ID token: %v", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	if err := s.storage.DeleteAuthCode(code); err != nil {
		s.logger.Errorf("failed to delete auth code: %v", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
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
			return
		}

		if err := s.storage.CreateRefresh(refresh); err != nil {
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
				UserID:  refresh.Claims.UserID,
				ConnID:  refresh.ConnectorID,
				Refresh: make(map[string]*storage.RefreshTokenRef),
			}
			offlineSessions.Refresh[tokenRef.ClientID] = &tokenRef

			// Create a new OfflineSession object for the user and add a reference object for
			// the newly received refreshtoken.
			if err := s.storage.CreateOfflineSessions(offlineSessions); err != nil {
				s.logger.Errorf("failed to create offline session: %v", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return
			}
		} else {
			if oldTokenRef, ok := session.Refresh[tokenRef.ClientID]; ok {
				// Delete old refresh token from storage.
				if err := s.storage.DeleteRefresh(oldTokenRef.ID); err != nil {
					s.logger.Errorf("failed to delete refresh token: %v", err)
					s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
					deleteToken = true
					return
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
				return
			}

		}
	}
	s.writeAccessToken(w, idToken, accessToken, refreshToken, expiry)
}

// handle a refresh token request https://tools.ietf.org/html/rfc6749#section-6
func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request, client storage.Client) {
	code := r.PostFormValue("refresh_token")
	scope := r.PostFormValue("scope")
	if code == "" {
		s.tokenErrHelper(w, errInvalidRequest, "No refresh token in request.", http.StatusBadRequest)
		return
	}

	token := new(internal.RefreshToken)
	if err := internal.Unmarshal(code, token); err != nil {
		// For backward compatibility, assume the refresh_token is a raw refresh token ID
		// if it fails to decode.
		//
		// Because refresh_token values that aren't unmarshable were generated by servers
		// that don't have a Token value, we'll still reject any attempts to claim a
		// refresh_token twice.
		token = &internal.RefreshToken{RefreshId: code, Token: ""}
	}

	refresh, err := s.storage.GetRefresh(token.RefreshId)
	if err != nil {
		s.logger.Errorf("failed to get refresh token: %v", err)
		if err == storage.ErrNotFound {
			s.tokenErrHelper(w, errInvalidRequest, "Refresh token is invalid or has already been claimed by another client.", http.StatusBadRequest)
		} else {
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		}
		return
	}
	if refresh.ClientID != client.ID {
		s.logger.Errorf("client %s trying to claim token for client %s", client.ID, refresh.ClientID)
		s.tokenErrHelper(w, errInvalidRequest, "Refresh token is invalid or has already been claimed by another client.", http.StatusBadRequest)
		return
	}
	if refresh.Token != token.Token {
		s.logger.Errorf("refresh token with id %s claimed twice", refresh.ID)
		s.tokenErrHelper(w, errInvalidRequest, "Refresh token is invalid or has already been claimed by another client.", http.StatusBadRequest)
		return
	}

	// Per the OAuth2 spec, if the client has omitted the scopes, default to the original
	// authorized scopes.
	//
	// https://tools.ietf.org/html/rfc6749#section-6
	scopes := refresh.Scopes
	if scope != "" {
		requestedScopes := strings.Fields(scope)
		var unauthorizedScopes []string

		for _, s := range requestedScopes {
			contains := func() bool {
				for _, scope := range refresh.Scopes {
					if s == scope {
						return true
					}
				}
				return false
			}()
			if !contains {
				unauthorizedScopes = append(unauthorizedScopes, s)
			}
		}

		if len(unauthorizedScopes) > 0 {
			msg := fmt.Sprintf("Requested scopes contain unauthorized scope(s): %q.", unauthorizedScopes)
			s.tokenErrHelper(w, errInvalidRequest, msg, http.StatusBadRequest)
			return
		}
		scopes = requestedScopes
	}

	var connectorData []byte
	if session, err := s.storage.GetOfflineSessions(refresh.Claims.UserID, refresh.ConnectorID); err != nil {
		if err != storage.ErrNotFound {
			s.logger.Errorf("failed to get offline session: %v", err)
			return
		}
	} else if len(refresh.ConnectorData) > 0 {
		// Use the old connector data if it exists, should be deleted once used
		connectorData = refresh.ConnectorData
	} else {
		connectorData = session.ConnectorData
	}

	conn, err := s.getConnector(refresh.ConnectorID)
	if err != nil {
		s.logger.Errorf("connector with ID %q not found: %v", refresh.ConnectorID, err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}
	ident := connector.Identity{
		UserID:            refresh.Claims.UserID,
		Username:          refresh.Claims.Username,
		PreferredUsername: refresh.Claims.PreferredUsername,
		Email:             refresh.Claims.Email,
		EmailVerified:     refresh.Claims.EmailVerified,
		Groups:            refresh.Claims.Groups,
		ConnectorData:     connectorData,
	}

	// Can the connector refresh the identity? If so, attempt to refresh the data
	// in the connector.
	//
	// TODO(ericchiang): We may want a strict mode where connectors that don't implement
	// this interface can't perform refreshing.
	if refreshConn, ok := conn.Connector.(connector.RefreshConnector); ok {
		newIdent, err := refreshConn.Refresh(r.Context(), parseScopes(scopes), ident)
		if err != nil {
			s.logger.Errorf("failed to refresh identity: %v", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
			return
		}
		ident = newIdent
	}

	claims := storage.Claims{
		UserID:            ident.UserID,
		Username:          ident.Username,
		PreferredUsername: ident.PreferredUsername,
		Email:             ident.Email,
		EmailVerified:     ident.EmailVerified,
		Groups:            ident.Groups,
	}

	accessToken, err := s.newAccessToken(client.ID, claims, scopes, refresh.Nonce, refresh.ConnectorID)
	if err != nil {
		s.logger.Errorf("failed to create new access token: %v", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	idToken, expiry, err := s.newIDToken(client.ID, claims, scopes, refresh.Nonce, accessToken, refresh.ConnectorID)
	if err != nil {
		s.logger.Errorf("failed to create ID token: %v", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	newToken := &internal.RefreshToken{
		RefreshId: refresh.ID,
		Token:     storage.NewID(),
	}
	rawNewToken, err := internal.Marshal(newToken)
	if err != nil {
		s.logger.Errorf("failed to marshal refresh token: %v", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	lastUsed := s.now()
	updater := func(old storage.RefreshToken) (storage.RefreshToken, error) {
		if old.Token != refresh.Token {
			return old, errors.New("refresh token claimed twice")
		}
		old.Token = newToken.Token
		// Update the claims of the refresh token.
		//
		// UserID intentionally ignored for now.
		old.Claims.Username = ident.Username
		old.Claims.PreferredUsername = ident.PreferredUsername
		old.Claims.Email = ident.Email
		old.Claims.EmailVerified = ident.EmailVerified
		old.Claims.Groups = ident.Groups
		old.LastUsed = lastUsed

		// ConnectorData has been moved to OfflineSession
		old.ConnectorData = []byte{}
		return old, nil
	}

	// Update LastUsed time stamp in refresh token reference object
	// in offline session for the user.
	if err := s.storage.UpdateOfflineSessions(refresh.Claims.UserID, refresh.ConnectorID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
		if old.Refresh[refresh.ClientID].ID != refresh.ID {
			return old, errors.New("refresh token invalid")
		}
		old.Refresh[refresh.ClientID].LastUsed = lastUsed
		old.ConnectorData = ident.ConnectorData
		return old, nil
	}); err != nil {
		s.logger.Errorf("failed to update offline session: %v", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	// Update refresh token in the storage.
	if err := s.storage.UpdateRefreshToken(refresh.ID, updater); err != nil {
		s.logger.Errorf("failed to update refresh token: %v", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	s.writeAccessToken(w, idToken, accessToken, rawNewToken, expiry)
}

func (s *Server) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	const prefix = "Bearer "

	auth := r.Header.Get("authorization")
	if len(auth) < len(prefix) || !strings.EqualFold(prefix, auth[:len(prefix)]) {
		w.Header().Set("WWW-Authenticate", "Bearer")
		s.tokenErrHelper(w, errAccessDenied, "Invalid bearer token.", http.StatusUnauthorized)
		return
	}
	rawIDToken := auth[len(prefix):]

	verifier := oidc.NewVerifier(s.issuerURL.String(), &storageKeySet{s.storage}, &oidc.Config{SkipClientIDCheck: true})
	idToken, err := verifier.Verify(r.Context(), rawIDToken)
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

func (s *Server) writeAccessToken(w http.ResponseWriter, idToken, accessToken, refreshToken string, expiry time.Time) {
	resp := struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token,omitempty"`
		IDToken      string `json:"id_token"`
	}{
		accessToken,
		"bearer",
		int(expiry.Sub(s.now()).Seconds()),
		refreshToken,
		idToken,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		s.logger.Errorf("failed to marshal access token response: %v", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}

func (s *Server) renderError(r *http.Request, w http.ResponseWriter, status int, description string) {
	if err := s.templates.err(r, w, status, description); err != nil {
		s.logger.Errorf("Server template error: %v", err)
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

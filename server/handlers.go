package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	jose "gopkg.in/square/go-jose.v2"

	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/storage"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	start := s.now()
	err := func() error {
		// Instead of trying to introspect health, just try to use the underlying storage.
		a := storage.AuthRequest{
			ID:       storage.NewID(),
			ClientID: storage.NewID(),

			// Set a short expiry so if the delete fails this will be cleaned up quickly by garbage collection.
			Expiry: s.now().Add(time.Minute),
		}

		if err := s.storage.CreateAuthRequest(a); err != nil {
			return fmt.Errorf("create auth request: %v", err)
		}
		if err := s.storage.DeleteAuthRequest(a.ID); err != nil {
			return fmt.Errorf("delete auth request: %v", err)
		}
		return nil
	}()

	t := s.now().Sub(start)
	if err != nil {
		log.Printf("Storage health check failed: %v", err)
		http.Error(w, "Health check failed", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "Health check passed in %s", t)
}

func (s *Server) handlePublicKeys(w http.ResponseWriter, r *http.Request) {
	// TODO(ericchiang): Cache this.
	keys, err := s.storage.GetKeys()
	if err != nil {
		log.Printf("failed to get keys: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if keys.SigningKeyPub == nil {
		log.Printf("No public keys found.")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
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
		log.Printf("failed to marshal discovery data: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	maxAge := keys.NextRotation.Sub(s.now())
	if maxAge < (time.Minute * 2) {
		maxAge = time.Minute * 2
	}

	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, must-revalidate", maxAge))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}

type discovery struct {
	Issuer        string   `json:"issuer"`
	Auth          string   `json:"authorization_endpoint"`
	Token         string   `json:"token_endpoint"`
	Keys          string   `json:"jwks_uri"`
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
		Subjects:    []string{"public"},
		IDTokenAlgs: []string{string(jose.RS256)},
		Scopes:      []string{"openid", "email", "profile", "offline_access"},
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

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.Write(data)
	}, nil
}

// handleAuthorization handles the OAuth2 auth endpoint.
func (s *Server) handleAuthorization(w http.ResponseWriter, r *http.Request) {
	authReq, err := parseAuthorizationRequest(s.storage, s.supportedResponseTypes, r)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err.Type, err.Description)
		return
	}
	if err := s.storage.CreateAuthRequest(authReq); err != nil {
		log.Printf("Failed to create authorization request: %v", err)
		s.renderError(w, http.StatusInternalServerError, errServerError, "")
		return
	}
	state := authReq.ID

	if len(s.connectors) == 1 {
		for id := range s.connectors {
			http.Redirect(w, r, s.absPath("/auth", id)+"?state="+state, http.StatusFound)
			return
		}
	}

	connectorInfos := make([]connectorInfo, len(s.connectors))
	i := 0
	for id, conn := range s.connectors {
		connectorInfos[i] = connectorInfo{
			ID:   id,
			Name: conn.DisplayName,
			URL:  s.absPath("/auth", id),
		}
		i++
	}

	s.templates.login(w, connectorInfos, state)
}

func (s *Server) handleConnectorLogin(w http.ResponseWriter, r *http.Request) {
	connID := mux.Vars(r)["connector"]
	conn, ok := s.connectors[connID]
	if !ok {
		s.notFound(w, r)
		return
	}

	// TODO(ericchiang): cache user identity.

	state := r.FormValue("state")
	switch r.Method {
	case "GET":
		switch conn := conn.Connector.(type) {
		case connector.CallbackConnector:
			callbackURL, err := conn.LoginURL(s.absURL("/callback", connID), state)
			if err != nil {
				log.Printf("Connector %q returned error when creating callback: %v", connID, err)
				s.renderError(w, http.StatusInternalServerError, errServerError, "")
				return
			}
			http.Redirect(w, r, callbackURL, http.StatusFound)
		case connector.PasswordConnector:
			s.templates.password(w, state, r.URL.String(), "", false)
		default:
			s.notFound(w, r)
		}
	case "POST":
		passwordConnector, ok := conn.Connector.(connector.PasswordConnector)
		if !ok {
			s.notFound(w, r)
			return
		}

		username := r.FormValue("login")
		password := r.FormValue("password")

		identity, ok, err := passwordConnector.Login(username, password)
		if err != nil {
			log.Printf("Failed to login user: %v", err)
			s.renderError(w, http.StatusInternalServerError, errServerError, "")
			return
		}
		if !ok {
			s.templates.password(w, state, r.URL.String(), username, true)
			return
		}
		redirectURL, err := s.finalizeLogin(identity, state, connID, conn.Connector)
		if err != nil {
			log.Printf("Failed to finalize login: %v", err)
			s.renderError(w, http.StatusInternalServerError, errServerError, "")
			return
		}

		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	default:
		s.notFound(w, r)
	}
}

func (s *Server) handleConnectorCallback(w http.ResponseWriter, r *http.Request) {
	connID := mux.Vars(r)["connector"]
	conn, ok := s.connectors[connID]
	if !ok {
		s.notFound(w, r)
		return
	}
	callbackConnector, ok := conn.Connector.(connector.CallbackConnector)
	if !ok {
		s.notFound(w, r)
		return
	}

	identity, state, err := callbackConnector.HandleCallback(r)
	if err != nil {
		log.Printf("Failed to authenticate: %v", err)
		s.renderError(w, http.StatusInternalServerError, errServerError, "")
		return
	}

	redirectURL, err := s.finalizeLogin(identity, state, connID, conn.Connector)
	if err != nil {
		log.Printf("Failed to finalize login: %v", err)
		s.renderError(w, http.StatusInternalServerError, errServerError, "")
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (s *Server) finalizeLogin(identity connector.Identity, authReqID, connectorID string, conn connector.Connector) (string, error) {
	if authReqID == "" {
		return "", errors.New("no auth request ID passed")
	}
	claims := storage.Claims{
		UserID:        identity.UserID,
		Username:      identity.Username,
		Email:         identity.Email,
		EmailVerified: identity.EmailVerified,
	}

	groupsConn, ok := conn.(connector.GroupsConnector)
	if ok {
		authReq, err := s.storage.GetAuthRequest(authReqID)
		if err != nil {
			return "", fmt.Errorf("get auth request: %v", err)
		}
		reqGroups := func() bool {
			for _, scope := range authReq.Scopes {
				if scope == scopeGroups {
					return true
				}
			}
			return false
		}()
		if reqGroups {
			if claims.Groups, err = groupsConn.Groups(identity); err != nil {
				return "", fmt.Errorf("getting groups: %v", err)
			}
		}
	}

	updater := func(a storage.AuthRequest) (storage.AuthRequest, error) {
		a.LoggedIn = true
		a.Claims = claims
		a.ConnectorID = connectorID
		a.ConnectorData = identity.ConnectorData
		return a, nil
	}
	if err := s.storage.UpdateAuthRequest(authReqID, updater); err != nil {
		return "", fmt.Errorf("failed to update auth request: %v", err)
	}
	return path.Join(s.issuerURL.Path, "/approval") + "?state=" + authReqID, nil
}

func (s *Server) handleApproval(w http.ResponseWriter, r *http.Request) {
	authReq, err := s.storage.GetAuthRequest(r.FormValue("state"))
	if err != nil {
		log.Printf("Failed to get auth request: %v", err)
		s.renderError(w, http.StatusInternalServerError, errServerError, "")
		return
	}
	if !authReq.LoggedIn {
		log.Printf("Auth request does not have an identity for approval")
		s.renderError(w, http.StatusInternalServerError, errServerError, "")
		return
	}

	switch r.Method {
	case "GET":
		if s.skipApproval {
			s.sendCodeResponse(w, r, authReq)
			return
		}
		client, err := s.storage.GetClient(authReq.ClientID)
		if err != nil {
			log.Printf("Failed to get client %q: %v", authReq.ClientID, err)
			s.renderError(w, http.StatusInternalServerError, errServerError, "")
			return
		}
		s.templates.approval(w, authReq.ID, authReq.Claims.Username, client.Name, authReq.Scopes)
	case "POST":
		if r.FormValue("approval") != "approve" {
			s.renderError(w, http.StatusInternalServerError, "approval rejected", "")
			return
		}
		s.sendCodeResponse(w, r, authReq)
	}
}

func (s *Server) sendCodeResponse(w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest) {
	if authReq.Expiry.After(s.now()) {
		s.renderError(w, http.StatusBadRequest, errInvalidRequest, "Authorization request period has expired.")
		return
	}

	if err := s.storage.DeleteAuthRequest(authReq.ID); err != nil {
		if err != storage.ErrNotFound {
			log.Printf("Failed to delete authorization request: %v", err)
			s.renderError(w, http.StatusInternalServerError, errServerError, "")
		} else {
			s.renderError(w, http.StatusBadRequest, errInvalidRequest, "Authorization request has already been completed.")
		}
		return
	}
	u, err := url.Parse(authReq.RedirectURI)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, errServerError, "Invalid redirect URI.")
		return
	}
	q := u.Query()

	for _, responseType := range authReq.ResponseTypes {
		switch responseType {
		case responseTypeCode:
			code := storage.AuthCode{
				ID:          storage.NewID(),
				ClientID:    authReq.ClientID,
				ConnectorID: authReq.ConnectorID,
				Nonce:       authReq.Nonce,
				Scopes:      authReq.Scopes,
				Claims:      authReq.Claims,
				Expiry:      s.now().Add(time.Minute * 5),
				RedirectURI: authReq.RedirectURI,
			}
			if err := s.storage.CreateAuthCode(code); err != nil {
				log.Printf("Failed to create auth code: %v", err)
				s.renderError(w, http.StatusInternalServerError, errServerError, "")
				return
			}

			if authReq.RedirectURI == redirectURIOOB {
				// TODO(ericchiang): Add a proper template.
				fmt.Fprintf(w, "Code: %s", code.ID)
				return
			}
			q.Set("code", code.ID)
		case responseTypeToken:
			idToken, expiry, err := s.newIDToken(authReq.ClientID, authReq.Claims, authReq.Scopes, authReq.Nonce)
			if err != nil {
				log.Printf("failed to create ID token: %v", err)
				tokenErr(w, errServerError, "", http.StatusInternalServerError)
				return
			}
			v := url.Values{}
			v.Set("access_token", storage.NewID())
			v.Set("token_type", "bearer")
			v.Set("id_token", idToken)
			v.Set("state", authReq.State)
			v.Set("expires_in", strconv.Itoa(int(expiry.Sub(s.now()))))
			u.Fragment = v.Encode()
		}
	}

	q.Set("state", authReq.State)
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusSeeOther)
}

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	clientID, clientSecret, ok := r.BasicAuth()
	if ok {
		var err error
		if clientID, err = url.QueryUnescape(clientID); err != nil {
			tokenErr(w, errInvalidRequest, "client_id improperly encoded", http.StatusBadRequest)
			return
		}
		if clientSecret, err = url.QueryUnescape(clientSecret); err != nil {
			tokenErr(w, errInvalidRequest, "client_secret improperly encoded", http.StatusBadRequest)
			return
		}
	} else {
		clientID = r.PostFormValue("client_id")
		clientSecret = r.PostFormValue("client_secret")
	}

	client, err := s.storage.GetClient(clientID)
	if err != nil {
		if err != storage.ErrNotFound {
			log.Printf("failed to get client: %v", err)
			tokenErr(w, errServerError, "", http.StatusInternalServerError)
		} else {
			tokenErr(w, errInvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
		}
		return
	}
	if client.Secret != clientSecret {
		tokenErr(w, errInvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
		return
	}

	grantType := r.PostFormValue("grant_type")
	switch grantType {
	case grantTypeAuthorizationCode:
		s.handleAuthCode(w, r, client)
	case grantTypeRefreshToken:
		s.handleRefreshToken(w, r, client)
	default:
		tokenErr(w, errInvalidGrant, "", http.StatusBadRequest)
	}
}

// handle an access token request https://tools.ietf.org/html/rfc6749#section-4.1.3
func (s *Server) handleAuthCode(w http.ResponseWriter, r *http.Request, client storage.Client) {
	code := r.PostFormValue("code")
	redirectURI := r.PostFormValue("redirect_uri")

	authCode, err := s.storage.GetAuthCode(code)
	if err != nil || s.now().After(authCode.Expiry) || authCode.ClientID != client.ID {
		if err != storage.ErrNotFound {
			log.Printf("failed to get auth code: %v", err)
			tokenErr(w, errServerError, "", http.StatusInternalServerError)
		} else {
			tokenErr(w, errInvalidRequest, "Invalid or expired code parameter.", http.StatusBadRequest)
		}
		return
	}

	if authCode.RedirectURI != redirectURI {
		tokenErr(w, errInvalidRequest, "redirect_uri did not match URI from initial request.", http.StatusBadRequest)
		return
	}

	idToken, expiry, err := s.newIDToken(client.ID, authCode.Claims, authCode.Scopes, authCode.Nonce)
	if err != nil {
		log.Printf("failed to create ID token: %v", err)
		tokenErr(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	if err := s.storage.DeleteAuthCode(code); err != nil {
		log.Printf("failed to delete auth code: %v", err)
		tokenErr(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	reqRefresh := func() bool {
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
			RefreshToken: storage.NewID(),
			ClientID:     authCode.ClientID,
			ConnectorID:  authCode.ConnectorID,
			Scopes:       authCode.Scopes,
			Claims:       authCode.Claims,
			Nonce:        authCode.Nonce,
		}
		if err := s.storage.CreateRefresh(refresh); err != nil {
			log.Printf("failed to create refresh token: %v", err)
			tokenErr(w, errServerError, "", http.StatusInternalServerError)
			return
		}
		refreshToken = refresh.RefreshToken
	}
	s.writeAccessToken(w, idToken, refreshToken, expiry)
}

// handle a refresh token request https://tools.ietf.org/html/rfc6749#section-6
func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request, client storage.Client) {
	code := r.PostFormValue("refresh_token")
	scope := r.PostFormValue("scope")
	if code == "" {
		tokenErr(w, errInvalidRequest, "No refresh token in request.", http.StatusBadRequest)
		return
	}

	refresh, err := s.storage.GetRefresh(code)
	if err != nil || refresh.ClientID != client.ID {
		if err != storage.ErrNotFound {
			log.Printf("failed to get auth code: %v", err)
			tokenErr(w, errServerError, "", http.StatusInternalServerError)
		} else {
			tokenErr(w, errInvalidRequest, "Refresh token is invalid or has already been claimed by another client.", http.StatusBadRequest)
		}
		return
	}

	scopes := refresh.Scopes
	if scope != "" {
		requestedScopes := strings.Split(scope, " ")
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
			tokenErr(w, errInvalidRequest, msg, http.StatusBadRequest)
			return
		}
		scopes = requestedScopes
	}

	// TODO(ericchiang): re-auth with backends

	idToken, expiry, err := s.newIDToken(client.ID, refresh.Claims, scopes, refresh.Nonce)
	if err != nil {
		log.Printf("failed to create ID token: %v", err)
		tokenErr(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	if err := s.storage.DeleteRefresh(code); err != nil {
		log.Printf("failed to delete auth code: %v", err)
		tokenErr(w, errServerError, "", http.StatusInternalServerError)
		return
	}
	refresh.RefreshToken = storage.NewID()
	if err := s.storage.CreateRefresh(refresh); err != nil {
		log.Printf("failed to create refresh token: %v", err)
		tokenErr(w, errServerError, "", http.StatusInternalServerError)
		return
	}
	s.writeAccessToken(w, idToken, refresh.RefreshToken, expiry)
}

func (s *Server) writeAccessToken(w http.ResponseWriter, idToken, refreshToken string, expiry time.Time) {
	// TODO(ericchiang): figure out an access token story and support the user info
	// endpoint. For now use a random value so no one depends on the access_token
	// holding a specific structure.
	resp := struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token,omitempty"`
		IDToken      string `json:"id_token"`
	}{
		storage.NewID(),
		"bearer",
		int(expiry.Sub(s.now())),
		refreshToken,
		idToken,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("failed to marshal access token response: %v", err)
		tokenErr(w, errServerError, "", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}

func (s *Server) renderError(w http.ResponseWriter, status int, err, description string) {
	http.Error(w, fmt.Sprintf("%s: %s", err, description), status)
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

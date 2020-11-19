package server

import (
	"encoding/json"
	"net/http"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/storage"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

// NewSessionStore - Allocate a new CookieStore
// @params: server.SSO
// @returns: *sessions.CookieStore, error
func NewSessionStore(ssoConf SSO) (*sessions.CookieStore, error) {
	encKey := []byte(ssoConf.EncryptKey)
	if len(encKey) == 0 {
		encKey = securecookie.GenerateRandomKey(32)
	}
	sessionStore := sessions.NewCookieStore(encKey)

	maxage := ssoConf.MaxAge
	if maxage > 0 {
		sessionStore.MaxAge(maxage)
	} else {
		sessionStore.MaxAge(1024)
	}
	sessionStore.Options.Secure = false
	sessionStore.Options.HttpOnly = true
	return sessionStore, nil
}

// getSession - get the current Session
// @params: *http.Request, storage.AuthRequest
// @returns: *sessions.Session
func (s *Server) getSession(r *http.Request) *sessions.Session {
	session, _ := s.sessionStore.Get(r, s.sso.SessionName)
	return session
}

// getSessionIdentity - from the session get the value of Identity and decrypted it
// @params: *sessions.Session, storage.AuthRequest
// @returns: connector.Identity, bool
func (s *Server) getSessionIdentity(session *sessions.Session) (connector.Identity, bool) {
	var identity connector.Identity
	identityRaw, ok := session.Values["identity"].([]byte)
	if !ok {
		return identity, false
	}
	err := json.Unmarshal(identityRaw, &identity)
	if err != nil {
		return identity, false
	}
	return identity, true
}

func (s *Server) getSessionConnectorID(session *sessions.Session) (string, bool) {
	connID, ok := session.Values["connectorID"].(string)
	if !ok {
		return "", false
	}
	return connID, true
}

// sessionGetScopes - from the session get the value of Scopes
// @params: *sessions.Session
// @returns: map[string]bool
func (s *Server) sessionGetScopes(session *sessions.Session) map[string]bool {
	scopesRaw, ok := session.Values["scopes"].([]byte)
	if ok {
		var scopes map[string]bool
		err := json.Unmarshal(scopesRaw, &scopes)
		if err == nil {
			return scopes
		}
	}
	return make(map[string]bool)
}

// sessionScopesApproved - from the session check if we got all scopes wanted
// @params: storage.AuthRequest
// @returns: bool
func (s *Server) sessionScopesApproved(session *sessions.Session, authReq storage.AuthRequest) bool {
	// check all scopes are approved in the session
	scopes := s.sessionGetScopes(session)
	for _, wantedScope := range authReq.Scopes {
		_, ok := scopes[wantedScope]
		if !ok {
			return false
		}
	}
	return true
}

// authenticateSession - store all scopes wanted into session
// @params: http.ResponseWriter, *http.Request, storage.AuthRequest
// @returns: none
func (s *Server) authenticateSession(w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest) {
	// add scopes of the request to session scopes after approval
	session := s.getSession(r)
	scopes := s.sessionGetScopes(session)
	for _, wantedScope := range authReq.Scopes {
		scopes[wantedScope] = true
	}
	var err error
	session.Values["scopes"], err = json.Marshal(scopes)
	if err != nil {
		s.logger.Errorf("failed to marshal scopes: %v", err)
	} else {
		session.Save(r, w)
	}
}

// storeAuthenticateSession - store all identity into session
// @params: http.ResponseWriter, *http.Request, storage.AuthRequest, connector.Identity
// @returns: error
func (s *Server) storeAuthenticateSession(w http.ResponseWriter, r *http.Request, identity connector.Identity, authReq storage.AuthRequest) error {
	var err error
	session := s.getSession(r)
	session.Values["identity"], err = json.Marshal(identity)
	if err != nil {
		s.logger.Errorf("failed to marshal identity: %v", err)
		return err
	}
	session.Values["connectorID"] = authReq.ConnectorID
	session.Save(r, w)
	return nil
}

// sooCheckSess - try to find data into session, and send code respons to the client
// @params: http.ResponseWriter, *http.Request, storage.AuthRequest, authReqID string
// @returns: none
func (s *Server) sooCheckSess(w http.ResponseWriter, r *http.Request, authReqID string) {

	session := s.getSession(r)
	identity, idFound := s.getSessionIdentity(session)
	connectorID, connFound := s.getSessionConnectorID(session)

	if !idFound {
		// No session found for sso
		return
	} else {
		if !connFound {
			return
		}
		// Recreate the authReq with data store into session
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
		claims := storage.Claims{
			UserID:            identity.UserID,
			Username:          identity.Username,
			PreferredUsername: identity.PreferredUsername,
			Email:             identity.Email,
			EmailVerified:     identity.EmailVerified,
			Groups:            identity.Groups,
		}
		authReq.Claims = claims
		authReq.ConnectorID = connectorID

		// Check the finalize login method
		conn, err := s.getConnector(connectorID)
		if err != nil {
			s.logger.Errorf("Failed to create authorization request: %v", err)
			s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist")
			return
		}
		redirectURL, err := s.finalizeLogin(identity, authReq, conn)
		if err != nil {
			s.logger.Errorf("Failed to finalize login: %v", err)
			s.renderError(r, w, http.StatusInternalServerError, "Login error.")
			return
		}

		// if all scopes are approved end, else ask for approval for new scopes
		authenticated := s.sessionScopesApproved(session, authReq)
		if authenticated {
			s.sendCodeResponse(w, r, authReq)
		} else {
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		}
	}
}

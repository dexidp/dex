package manager

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/jonboulle/clockwork"

	"github.com/coreos/dex/session"
	"github.com/coreos/go-oidc/oidc"
)

type GenerateCodeFunc func() (string, error)

func DefaultGenerateCode() (string, error) {
	b := make([]byte, 8)
	n, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	if n != 8 {
		return "", errors.New("unable to read enough random bytes")
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func NewSessionManager(sRepo session.SessionRepo, skRepo session.SessionKeyRepo) *SessionManager {
	return &SessionManager{
		GenerateCode:   DefaultGenerateCode,
		Clock:          clockwork.NewRealClock(),
		ValidityWindow: session.DefaultSessionValidityWindow,
		sessions:       sRepo,
		keys:           skRepo,
	}
}

type SessionManager struct {
	GenerateCode   GenerateCodeFunc
	Clock          clockwork.Clock
	ValidityWindow time.Duration
	sessions       session.SessionRepo
	keys           session.SessionKeyRepo
}

func (m *SessionManager) NewSession(connectorID, clientID, clientState string, redirectURL url.URL, nonce string, register bool, scope []string) (string, error) {
	sID, err := m.GenerateCode()
	if err != nil {
		return "", err
	}

	now := m.Clock.Now()
	s := session.Session{
		ConnectorID: connectorID,
		ID:          sID,
		State:       session.SessionStateNew,
		CreatedAt:   now,
		ExpiresAt:   now.Add(m.ValidityWindow),
		ClientID:    clientID,
		ClientState: clientState,
		RedirectURL: redirectURL,
		Register:    register,
		Nonce:       nonce,
		Scope:       scope,
	}

	err = m.sessions.Create(s)
	if err != nil {
		return "", err
	}

	return sID, nil
}

func (m *SessionManager) NewSessionKey(sessionID string) (string, error) {
	key, err := m.GenerateCode()
	if err != nil {
		return "", err
	}

	k := session.SessionKey{
		Key:       key,
		SessionID: sessionID,
	}

	sessionKeyValidityWindow := 10 * time.Minute //RFC6749
	err = m.keys.Push(k, sessionKeyValidityWindow)
	if err != nil {
		return "", err
	}

	return k.Key, nil
}

func (m *SessionManager) ExchangeKey(key string) (string, error) {
	return m.keys.Pop(key)
}

func (m *SessionManager) GetSessionByKey(key string) (string, error) {
	return m.keys.Peek(key)
}

func (m *SessionManager) getSessionInState(sessionID string, state session.SessionState) (*session.Session, error) {
	s, err := m.sessions.Get(sessionID)
	if err != nil {
		return nil, err
	}

	if s.State != state {
		return nil, fmt.Errorf("session state %s, expect %s", s.State, state)
	}

	return s, nil
}

func (m *SessionManager) AttachRemoteIdentity(sessionID string, ident oidc.Identity) (*session.Session, error) {
	s, err := m.getSessionInState(sessionID, session.SessionStateNew)
	if err != nil {
		return nil, err
	}

	s.Identity = ident
	s.State = session.SessionStateRemoteAttached

	if err = m.sessions.Update(*s); err != nil {
		return nil, err
	}

	return s, nil
}

func (m *SessionManager) AttachUser(sessionID string, userID string) (*session.Session, error) {
	s, err := m.getSessionInState(sessionID, session.SessionStateRemoteAttached)
	if err != nil {
		return nil, err
	}

	s.UserID = userID
	s.State = session.SessionStateIdentified

	if err = m.sessions.Update(*s); err != nil {
		return nil, err
	}

	return s, nil
}

func (m *SessionManager) AttachGroups(sessionID string, groups []string) (*session.Session, error) {
	s, err := m.sessions.Get(sessionID)
	if err != nil {
		return nil, err
	}
	s.Groups = groups
	if err = m.sessions.Update(*s); err != nil {
		return nil, err
	}
	return s, nil
}

func (m *SessionManager) Kill(sessionID string) (*session.Session, error) {
	s, err := m.sessions.Get(sessionID)
	if err != nil {
		return nil, err
	}

	s.State = session.SessionStateDead

	if err = m.sessions.Update(*s); err != nil {
		return nil, err
	}

	return s, nil
}

func (m *SessionManager) Get(sessionID string) (*session.Session, error) {
	return m.sessions.Get(sessionID)
}

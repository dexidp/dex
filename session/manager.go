package session

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/jonboulle/clockwork"

	"github.com/coreos/go-oidc/oidc"
)

type GenerateCodeFunc func() (string, error)

func DefaultGenerateCode() (string, error) {
	b := make([]byte, 8)
	n, err := rand.Read(b)
	if err != nil {
		return "", err
	} else if n != 8 {
		return "", errors.New("unable to read enough random bytes")
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func NewSessionManager(sRepo SessionRepo, skRepo SessionKeyRepo) *SessionManager {
	return &SessionManager{
		GenerateCode:   DefaultGenerateCode,
		Clock:          clockwork.NewRealClock(),
		ValidityWindow: DefaultSessionValidityWindow,
		sessions:       sRepo,
		keys:           skRepo,
	}
}

type SessionManager struct {
	GenerateCode   GenerateCodeFunc
	Clock          clockwork.Clock
	ValidityWindow time.Duration
	sessions       SessionRepo
	keys           SessionKeyRepo
}

func (m *SessionManager) NewSession(connectorID, clientID, clientState string, redirectURL url.URL, nonce string, register bool) (string, error) {
	sID, err := m.GenerateCode()
	if err != nil {
		return "", err
	}

	now := m.Clock.Now()
	s := Session{
		ConnectorID: connectorID,
		ID:          sID,
		State:       SessionStateNew,
		CreatedAt:   now,
		ExpiresAt:   now.Add(m.ValidityWindow),
		ClientID:    clientID,
		ClientState: clientState,
		RedirectURL: redirectURL,
		Register:    register,
		Nonce:       nonce,
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

	k := SessionKey{
		Key:       key,
		SessionID: sessionID,
	}

	err = m.keys.Push(k, sessionKeyValidityWindow)
	if err != nil {
		return "", err
	}

	return k.Key, nil
}

func (m *SessionManager) ExchangeKey(key string) (string, error) {
	return m.keys.Pop(key)
}

func (m *SessionManager) getSessionInState(sessionID string, state SessionState) (*Session, error) {
	s, err := m.sessions.Get(sessionID)
	if err != nil {
		return nil, err
	}

	if s.State != state {
		return nil, fmt.Errorf("session state %s, expect %s", s.State, state)
	}

	return s, nil
}

func (m *SessionManager) AttachRemoteIdentity(sessionID string, ident oidc.Identity) (*Session, error) {
	s, err := m.getSessionInState(sessionID, SessionStateNew)
	if err != nil {
		return nil, err
	}

	s.Identity = ident
	s.State = SessionStateRemoteAttached

	if err = m.sessions.Update(*s); err != nil {
		return nil, err
	}

	return s, nil
}

func (m *SessionManager) AttachUser(sessionID string, userID string) (*Session, error) {
	s, err := m.getSessionInState(sessionID, SessionStateRemoteAttached)
	if err != nil {
		return nil, err
	}

	s.UserID = userID
	s.State = SessionStateIdentified

	if err = m.sessions.Update(*s); err != nil {
		return nil, err
	}

	return s, nil
}

func (m *SessionManager) Kill(sessionID string) (*Session, error) {
	s, err := m.sessions.Get(sessionID)
	if err != nil {
		return nil, err
	}

	s.State = SessionStateDead

	if err = m.sessions.Update(*s); err != nil {
		return nil, err
	}

	return s, nil
}

func (m *SessionManager) Get(sessionID string) (*Session, error) {
	return m.sessions.Get(sessionID)
}

package session

import (
	"errors"
	"time"

	"github.com/jonboulle/clockwork"
)

type SessionRepo interface {
	Get(string) (*Session, error)
	Create(Session) error
	Update(Session) error
}

type SessionKeyRepo interface {
	Push(SessionKey, time.Duration) error
	Pop(string) (string, error)
}

func NewSessionRepo() SessionRepo {
	return NewSessionRepoWithClock(clockwork.NewRealClock())
}

func NewSessionRepoWithClock(clock clockwork.Clock) SessionRepo {
	return &memSessionRepo{
		store: make(map[string]Session),
		clock: clock,
	}
}

type memSessionRepo struct {
	store map[string]Session
	clock clockwork.Clock
}

func (m *memSessionRepo) Get(sessionID string) (*Session, error) {
	s, ok := m.store[sessionID]
	if !ok || s.ExpiresAt.Before(m.clock.Now()) {
		return nil, errors.New("unrecognized ID")
	}
	return &s, nil
}

func (m *memSessionRepo) Create(s Session) error {
	if _, ok := m.store[s.ID]; ok {
		return errors.New("ID exists")
	}

	m.store[s.ID] = s
	return nil
}

func (m *memSessionRepo) Update(s Session) error {
	if _, ok := m.store[s.ID]; !ok {
		return errors.New("unrecognized ID")
	}
	m.store[s.ID] = s
	return nil
}

type expiringSessionKey struct {
	SessionKey
	expiresAt time.Time
}

func NewSessionKeyRepo() SessionKeyRepo {
	return NewSessionKeyRepoWithClock(clockwork.NewRealClock())
}

func NewSessionKeyRepoWithClock(clock clockwork.Clock) SessionKeyRepo {
	return &memSessionKeyRepo{
		store: make(map[string]expiringSessionKey),
		clock: clock,
	}
}

type memSessionKeyRepo struct {
	store map[string]expiringSessionKey
	clock clockwork.Clock
}

func (m *memSessionKeyRepo) Pop(key string) (string, error) {
	esk, ok := m.store[key]
	if !ok {
		return "", errors.New("unrecognized key")
	}
	defer delete(m.store, key)

	if esk.expiresAt.Before(m.clock.Now()) {
		return "", errors.New("expired key")
	}

	return esk.SessionKey.SessionID, nil
}

func (m *memSessionKeyRepo) Push(sk SessionKey, ttl time.Duration) error {
	m.store[sk.Key] = expiringSessionKey{
		SessionKey: sk,
		expiresAt:  m.clock.Now().Add(ttl),
	}
	return nil
}

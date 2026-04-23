package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// DeviceState holds the state of an active Device Code Flow.
type DeviceState struct {
	DeviceCode      string
	UserCode        string
	VerificationURI string
	PollInterval    int
	Token           *oauth2.Token
}

// DeviceStore manages Device Code Flow sessions.
type DeviceStore interface {
	// Save stores a new session and returns its ID.
	// Previous sessions are considered invalidated.
	Save(state DeviceState) string

	// Get returns a session by ID. Returns false if the session
	// is not found or has been invalidated by a newer one.
	Get(sessionID string) (DeviceState, bool)

	// GetLatest returns the most recent session and its ID.
	// Returns false if no session exists.
	GetLatest() (string, DeviceState, bool)

	// SetToken attaches a token to the session.
	// Returns false if the session is not found or not current.
	SetToken(sessionID string, token *oauth2.Token) bool
}

// memoryDeviceStore is an in-memory DeviceStore that supports
// a single active session at a time.
type memoryDeviceStore struct {
	mu        sync.Mutex
	sessionID string
	state     DeviceState
}

// NewMemoryDeviceStore creates an in-memory DeviceStore.
func NewMemoryDeviceStore() DeviceStore {
	return &memoryDeviceStore{}
}

func (s *memoryDeviceStore) Save(state DeviceState) string {
	id := generateSessionID()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessionID = id
	s.state = state

	return id
}

func (s *memoryDeviceStore) Get(sessionID string) (DeviceState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID != s.sessionID {
		return DeviceState{}, false
	}
	return s.state, true
}

func (s *memoryDeviceStore) GetLatest() (string, DeviceState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sessionID == "" {
		return "", DeviceState{}, false
	}
	return s.sessionID, s.state, true
}

func (s *memoryDeviceStore) SetToken(sessionID string, token *oauth2.Token) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID != s.sessionID {
		return false
	}
	s.state.Token = token
	return true
}

// generateSessionID creates a random session identifier.
func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

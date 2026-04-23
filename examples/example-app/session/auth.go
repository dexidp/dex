package session

import "sync"

// UserClaims holds basic user identity claims from an ID token.
type UserClaims struct {
	Subject           string `json:"sub"`
	Name              string `json:"name"`
	Email             string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
}

// AuthState holds the authenticated user's state.
type AuthState struct {
	Claims  *UserClaims
	IDToken string
	Checked bool
}

// AuthStore manages the application's authentication session state.
type AuthStore interface {
	// Set stores authenticated user claims and the raw ID token,
	// and marks the session as checked.
	Set(claims *UserClaims, rawIDToken string)

	// Get returns the current authentication session state.
	Get() AuthState

	// Clear resets the session and returns the last raw ID token (for logout).
	Clear() string
}

// memoryAuthStore is an in-memory AuthStore.
type memoryAuthStore struct {
	mu    sync.RWMutex
	state AuthState
}

// NewMemoryAuthStore creates an in-memory AuthStore.
func NewMemoryAuthStore() AuthStore {
	return &memoryAuthStore{}
}

func (s *memoryAuthStore) Set(claims *UserClaims, rawIDToken string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.Claims = claims
	if rawIDToken != "" {
		s.state.IDToken = rawIDToken
	}
	s.state.Checked = true
}

func (s *memoryAuthStore) Get() AuthState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *memoryAuthStore) Clear() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	idToken := s.state.IDToken
	s.state = AuthState{}
	return idToken
}

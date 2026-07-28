// Package session keeps the example app's own state: who is signed in to this
// application, and what it is waiting for from the provider.
package session

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// CookieName is the app's session cookie. It identifies the browser, not the
// user: a session exists from the first request, before anyone signs in, so
// that a login can be tied to the browser that started it.
const CookieName = "example_app_session"

// ttl is how long an idle session is kept. The app is a demo; the number only
// has to be long enough that nobody loses a session mid-experiment.
const ttl = 12 * time.Hour

// UserClaims holds basic user identity claims from an ID token.
type UserClaims struct {
	Subject           string `json:"sub"`
	Name              string `json:"name"`
	Email             string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
}

// PendingAuth is one authorization the app has started and not yet finished.
// Everything in it belongs to a single authorization request: reusing any of it
// across requests is what the state and PKCE parameters exist to prevent.
type PendingAuth struct {
	State        string
	Nonce        string
	CodeVerifier string
	// Silent marks a prompt=none request, whose failure is an answer ("no
	// session at the provider") rather than an error to show the user.
	Silent  bool
	Created time.Time
}

// Device is a device authorization this browser started. It lives on the
// session for the same reason everything else here does: two people trying the
// device flow at once should not share one user code.
type Device struct {
	DeviceCode      string
	UserCode        string
	VerificationURI string
	PollInterval    int
	Token           *oauth2.Token
}

// Session is one browser's state.
type Session struct {
	ID string

	// Claims and Token are the result of the last completed sign-in.
	Claims  *UserClaims
	Token   *oauth2.Token
	IDToken string

	// Device is the device authorization in progress, if any.
	Device *Device

	// LastTokens is whatever the last flow produced, sign-in or not. Client
	// credentials and token exchange do not sign anyone in, and a tool that
	// forgets the tokens you just fetched is a tool you have to run twice.
	LastTokens  *oauth2.Token
	LastIDToken string
	LastGrant   string

	// LastProviderCheck is when the app last confirmed with the provider that
	// the session there still exists. Without it the app would keep showing a
	// user who signed out of the provider in another tab.
	LastProviderCheck time.Time

	pending map[string]*PendingAuth
	expires time.Time
}

// SignedIn reports whether this browser has completed a sign-in.
func (s *Session) SignedIn() bool { return s.Claims != nil }

// Store keeps sessions for the process. A demo app has no reason to persist
// them, but it does have a reason to keep them apart: one global session would
// mean every browser hitting this app shares one identity.
type Store struct {
	mu       sync.Mutex
	sessions map[string]*Session
}

// NewStore returns an empty in-memory store.
func NewStore() *Store {
	return &Store{sessions: make(map[string]*Session)}
}

// FromRequest returns the session for this browser, creating one and setting
// the cookie if the request carries none. The returned session must be handed
// back to Save after any change.
func (st *Store) FromRequest(w http.ResponseWriter, r *http.Request, secure bool) *Session {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.sweepLocked()

	if c, err := r.Cookie(CookieName); err == nil {
		if s, ok := st.sessions[c.Value]; ok {
			s.expires = time.Now().Add(ttl)
			return s
		}
	}

	s := &Session{
		ID:      randomString(),
		pending: make(map[string]*PendingAuth),
		expires: time.Now().Add(ttl),
	}
	st.sessions[s.ID] = s

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    s.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	})

	return s
}

// Save persists changes made to a session.
func (st *Store) Save(s *Session) {
	st.mu.Lock()
	defer st.mu.Unlock()

	s.expires = time.Now().Add(ttl)
	st.sessions[s.ID] = s
}

// SignIn records a completed sign-in.
func (st *Store) SignIn(s *Session, claims *UserClaims, token *oauth2.Token, rawIDToken string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	s.Claims = claims
	s.Token = token
	if rawIDToken != "" {
		s.IDToken = rawIDToken
	}
	s.LastProviderCheck = time.Now()
	s.expires = time.Now().Add(ttl)
}

// SignOut drops what the app knows about the user and returns the last ID
// token, which RP-initiated logout sends back to the provider as a hint.
func (st *Store) SignOut(s *Session) string {
	st.mu.Lock()
	defer st.mu.Unlock()

	idToken := s.IDToken
	s.Claims = nil
	s.Token = nil
	s.IDToken = ""
	// The tools prefill from the last flow's result. Keeping it past a sign-out
	// would put a signed-out user's access token back on screen, which reads as
	// the app inventing tokens.
	s.LastTokens = nil
	s.LastIDToken = ""
	s.LastGrant = ""
	s.Device = nil
	// LastProviderCheck deliberately survives: it records when the provider was
	// last asked, and signing out here does not make that answer any older.
	// Clearing it would send the next page load straight back into a check,
	// which is a redirect loop when the answer is "no session".
	return idToken
}

// StartAuth records an authorization the app is about to send the browser off
// to complete, and returns it. Each one carries its own state and PKCE
// verifier.
func (st *Store) StartAuth(s *Session, silent bool) *PendingAuth {
	st.mu.Lock()
	defer st.mu.Unlock()

	// Anything still pending from a much earlier request was abandoned.
	for state, p := range s.pending {
		if time.Since(p.Created) > 10*time.Minute {
			delete(s.pending, state)
		}
	}

	p := &PendingAuth{
		State:        randomString(),
		Nonce:        randomString(),
		CodeVerifier: oauth2.GenerateVerifier(),
		Silent:       silent,
		Created:      time.Now(),
	}
	s.pending[p.State] = p
	return p
}

// TakeAuth returns the pending authorization matching a callback's state
// parameter and forgets it, so a state cannot be replayed. A miss means the
// callback did not come from a request this browser started.
func (st *Store) TakeAuth(s *Session, state string) (*PendingAuth, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()

	p, ok := s.pending[state]
	if ok {
		delete(s.pending, state)
	}
	return p, ok
}

// RememberTokens keeps the result of the last flow so the tools can offer it
// and the token page can be reached again.
func (st *Store) RememberTokens(s *Session, grant string, token *oauth2.Token, rawIDToken string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	s.LastTokens = token
	s.LastIDToken = rawIDToken
	s.LastGrant = grant
	s.expires = time.Now().Add(ttl)
}

// StartDevice records a device authorization for this browser.
func (st *Store) StartDevice(s *Session, d *Device) {
	st.mu.Lock()
	defer st.mu.Unlock()

	s.Device = d
	s.expires = time.Now().Add(ttl)
}

// SetDeviceToken attaches the token a device authorization ended with.
func (st *Store) SetDeviceToken(s *Session, token *oauth2.Token) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if s.Device != nil {
		s.Device.Token = token
	}
}

// MarkChecked records that the provider was just asked about the session,
// whatever the answer, so a failing check cannot spin.
func (st *Store) MarkChecked(s *Session) {
	st.mu.Lock()
	defer st.mu.Unlock()

	s.LastProviderCheck = time.Now()
}

// sweepLocked drops expired sessions. Called on the way in, which is often
// enough for a process that only serves a handful of browsers.
func (st *Store) sweepLocked() {
	now := time.Now()
	for id, s := range st.sessions {
		if now.After(s.expires) {
			delete(st.sessions, id)
		}
	}
}

func randomString() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic("example-app: out of randomness: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

package session

import (
	"net/url"
	"time"

	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/oidc"

	"github.com/coreos/dex/scope"
)

const (
	sessionKeyValidityWindow = 10 * time.Minute //RFC6749

	// The default token expiration time.
	// This is exported, so it can be used to set the expiration
	// time in refresh token flow.
	DefaultSessionValidityWindow = 12 * time.Hour
)

type SessionState string

const (
	SessionStateNew            = SessionState("NEW")
	SessionStateRemoteAttached = SessionState("REMOTE_ATTACHED")
	SessionStateIdentified     = SessionState("IDENTIFIED")
	SessionStateDead           = SessionState("EXCHANGED")
)

type SessionKey struct {
	Key       string
	SessionID string
}

type Session struct {
	ConnectorID string
	ID          string
	State       SessionState
	CreatedAt   time.Time
	ExpiresAt   time.Time
	ClientID    string
	ClientState string
	RedirectURL url.URL
	Identity    oidc.Identity
	UserID      string

	// Regsiter indicates that this session is a registration flow.
	Register bool

	// Nonce is optionally provided in the initial authorization request, and
	// propogated in such cases to the generated claims.
	Nonce string

	// Scope is the 'scope' field in the authentication request. Example scopes
	// are 'openid', 'email', 'offline', etc.
	Scope scope.Scopes
}

// Claims returns a new set of Claims for the current session.
// The "sub" of the returned Claims is that of the dex User, not whatever
// remote Identity was used to authenticate.
func (s *Session) Claims(issuerURL string) jose.Claims {
	claims := oidc.NewClaims(issuerURL, s.UserID, s.ClientID, s.CreatedAt, s.ExpiresAt)
	if s.Nonce != "" {
		claims["nonce"] = s.Nonce
	}
	return claims
}

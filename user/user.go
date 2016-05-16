package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"net/mail"
	"net/url"

	"github.com/jonboulle/clockwork"
	"github.com/pborman/uuid"

	"github.com/coreos/dex/repo"
	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oidc"
)

const (
	MaxEmailLength = 200

	// ClaimPasswordResetPassword represents the hash of the password to be
	// reset; in other words, the old password
	ClaimPasswordResetPassword = "http://coreos.com/password/old-hash"

	// ClaimEmailVerificationEmail represents the email to be verified. Note
	// that we are intentionally not using the "email" claim for this purpose.
	ClaimEmailVerificationEmail = "http://coreos.com/email/verificationEmail"

	// ClaimPasswordResetCallback represents where a user should be sent after
	// resetting their password.
	ClaimPasswordResetCallback = "http://coreos.com/password/reset-callback"

	// Claim representing where a user should be sent after verifying their email address.
	ClaimEmailVerificationCallback = "http://coreos.com/email/verification-callback"

	// Claim representing where a user should be sent after responding to an invitation
	ClaimInvitationCallback = "http://coreos.com/invitation/callback"
)

var (
	clock = clockwork.NewRealClock()
)

func assertStringClaim(claims jose.Claims, k string) string {
	s, ok, err := claims.StringClaim(k)
	if !ok || err != nil {
		panic(fmt.Sprintf("claims were not validated correctly, missing or wrong claim: %v", k))
	}
	return s
}

func assertURLClaim(claims jose.Claims, k string) *url.URL {
	ustring := assertStringClaim(claims, k)
	ret, err := url.Parse(ustring)
	if err != nil {
		panic(fmt.Sprintf("url claim was not validated correctly: %v", k))
	}
	return ret
}

type UserIDGenerator func() (string, error)

func DefaultUserIDGenerator() (string, error) {
	return uuid.New(), nil
}

type User struct {
	// ID is the machine-generated, stable, unique identifier for this User.
	ID string

	// DisplayName is human readable name meant for display purposes.
	// DisplayName is not neccesarily unique with a UserRepo.
	DisplayName string

	Email string

	EmailVerified bool

	Admin bool

	Disabled bool

	CreatedAt time.Time
}

type UserFilter struct {
	// TODO(bobbyrullo): actually put stuff in here.
}

// AddToClaims adds basic information about the user to the given Claims.
// http://openid.net/specs/openid-connect-core-1_0.html#StandardClaims
func (u *User) AddToClaims(claims jose.Claims) {
	claims.Add("name", u.DisplayName)
	if u.Email != "" {
		claims.Add("email", u.Email)
		if u.EmailVerified {
			claims.Add("email_verified", true)
		}
	}
}

// UserRepo implementations maintain a persistent set of users.
// The following invariants must be maintained:
//  * Users must have a unique Email and ID
//  * Emails are case insensitive.
//  * No other Users may have the same RemoteIdentity as one of the
//    users. (This constraint may be relaxed in the future)
type UserRepo interface {
	Get(tx repo.Transaction, id string) (User, error)

	// List returns a list of users meeting the given conditions.
	// A nextPageToken is returned when there are further results to be had,
	// with the expectation that it will be passed into a subsequent List
	// call. When nextPageToken is non-empty filter and maxResults are ignored.
	List(tx repo.Transaction, filter UserFilter, maxResults int, nextPageToken string) ([]User, string, error)

	Create(repo.Transaction, User) error

	GetByEmail(tx repo.Transaction, email string) (User, error)

	Disable(tx repo.Transaction, id string, disabled bool) error

	Update(repo.Transaction, User) error

	GetByRemoteIdentity(repo.Transaction, RemoteIdentity) (User, error)

	AddRemoteIdentity(tx repo.Transaction, userID string, remoteID RemoteIdentity) error

	RemoveRemoteIdentity(tx repo.Transaction, userID string, remoteID RemoteIdentity) error

	GetRemoteIdentities(tx repo.Transaction, userID string) ([]RemoteIdentity, error)

	GetAdminCount(repo.Transaction) (int, error)
}

var (
	ErrorDuplicateID    = errors.New("ID not available")
	ErrorDuplicateEmail = errors.New("email not available")

	ErrorDuplicateRemoteIdentity = errors.New("remote identity already in use for another user")
	ErrorInvalidEmail            = errors.New("invalid Email")
	ErrorInvalidID               = errors.New("invalid ID")
	ErrorNotFound                = errors.New("user not found in repository")
)

// RemoteIdentity represents a User's identity at an IDP.
type RemoteIdentity struct {
	// IDPCID is the identifier of the IDP which hosts this identity.
	ConnectorID string

	// ID is the identifier of this User at the IDP.
	ID string
}

func ValidEmail(email string) bool {
	address, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}

	// Has mail.ParseAddress parsed the entire string and only found a single
	// address without a name?
	return address.Address == email
}

func ValidPassword(plaintext string) bool {
	return len(plaintext) > 5
}

type UserWithRemoteIdentities struct {
	User             User             `json:"user"`
	RemoteIdentities []RemoteIdentity `json:"remoteIdentities"`
}

func (u *User) UnmarshalJSON(data []byte) error {
	var dec struct {
		ID            string `json:"id"`
		DisplayName   string `json:"displayName"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"emailVerified"`
	}

	err := json.Unmarshal(data, &dec)
	if err != nil {
		return fmt.Errorf("invalid User entry: %v", err)
	}

	u.ID = dec.ID
	u.DisplayName = dec.DisplayName
	u.Email = dec.Email
	u.EmailVerified = dec.EmailVerified
	return nil
}

func (u *RemoteIdentity) UnmarshalJSON(data []byte) error {
	var dec struct {
		ConnectorID string `json:"connectorID"`
		ID          string `json:"id"`
	}

	err := json.Unmarshal(data, &dec)
	if err != nil {
		return fmt.Errorf("invalid RemoteIdentity entry: %v", err)
	}

	u.ID = dec.ID
	u.ConnectorID = dec.ConnectorID

	return nil
}

type TokenClaims struct {
	Claims jose.Claims
}

// Returns TokenClaims if and only if
// - the given token string is an appropriately formatted JWT
// - the JWT contains nonempty "aud" and "sub" claims
// - the JWT can be verified for the client associated with the "aud" claim
//   using the given keys
func parseAndVerifyTokenClaims(token string, issuer url.URL, keys []key.PublicKey) (TokenClaims, error) {
	jwt, err := jose.ParseJWT(token)
	if err != nil {
		return TokenClaims{}, err
	}

	claims, err := jwt.Claims()
	if err != nil {
		return TokenClaims{}, err
	}

	clientID, ok, err := claims.StringClaim("aud")
	if err != nil {
		return TokenClaims{}, err
	}
	if !ok || clientID == "" {
		return TokenClaims{}, errors.New("no aud(client ID) claim")
	}

	sub, ok, err := claims.StringClaim("sub")
	if err != nil {
		return TokenClaims{}, err
	}
	if !ok || sub == "" {
		return TokenClaims{}, errors.New("no sub claim")
	}

	noop := func() error { return nil }

	keysFunc := func() []key.PublicKey {
		return keys
	}

	verifier := oidc.NewJWTVerifier(issuer.String(), clientID, noop, keysFunc)
	if err := verifier.Verify(jwt); err != nil {
		return TokenClaims{}, err
	}

	timeClaimsToInt(claims)

	return TokenClaims{claims}, nil
}

// timeClaimsToInt converts float64 time claims to ints.
// This is unfortunately neccessary for interop as some clients incorrectly fail
// to marshal floats as times.
func timeClaimsToInt(claims jose.Claims) {
	for _, k := range []string{"exp", "iat"} {
		v, ok := claims[k]
		if !ok {
			continue
		}

		fVal, ok := v.(float64)
		if !ok {
			continue
		}

		// round
		claims[k] = int64(fVal + math.Copysign(0.5, fVal))
	}
}

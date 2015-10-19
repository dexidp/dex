package user

import (
	"fmt"
	"net/url"
	"time"

	"github.com/jonboulle/clockwork"

	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oidc"
)

var (
	clock = clockwork.NewRealClock()
)

// NewEmailVerification creates an object which can be sent to a user in serialized form to verify that they control an email address.
// The clientID is the ID of the registering user. The callback is where a user should land after verifying their email.
func NewEmailVerification(user User, clientID string, issuer url.URL, callback url.URL, expires time.Duration) EmailVerification {
	claims := oidc.NewClaims(issuer.String(), user.ID, clientID, clock.Now(), clock.Now().Add(expires))
	claims.Add(ClaimEmailVerificationCallback, callback.String())
	claims.Add(ClaimEmailVerificationEmail, user.Email)
	return EmailVerification{claims}
}

type EmailVerification struct {
	Claims jose.Claims
}

// Assumes that parseAndVerifyTokenClaims has already been called on claims
func verifyEmailVerificationClaims(claims jose.Claims) (EmailVerification, error) {
	email, ok, err := claims.StringClaim(ClaimEmailVerificationEmail)
	if err != nil {
		return EmailVerification{}, err
	}
	if !ok || email == "" {
		return EmailVerification{}, fmt.Errorf("no %q claim", ClaimEmailVerificationEmail)
	}

	cb, ok, err := claims.StringClaim(ClaimEmailVerificationCallback)
	if err != nil {
		return EmailVerification{}, err
	}
	if !ok || cb == "" {
		return EmailVerification{}, fmt.Errorf("no %q claim", ClaimEmailVerificationCallback)
	}
	if _, err := url.Parse(cb); err != nil {
		return EmailVerification{}, fmt.Errorf("callback URL not parseable: %v", cb)
	}

	return EmailVerification{claims}, nil
}

// ParseAndVerifyEmailVerificationToken parses a string into a an EmailVerification, verifies the signature, and ensures that required claims are present.
// In addition to the usual claims required by the OIDC spec, "aud" and "sub" must be present as well as ClaimEmailVerificationCallback and ClaimEmailVerificationEmail.
func ParseAndVerifyEmailVerificationToken(token string, issuer url.URL, keys []key.PublicKey) (EmailVerification, error) {
	tokenClaims, err := parseAndVerifyTokenClaims(token, issuer, keys)
	if err != nil {
		return EmailVerification{}, err
	}

	return verifyEmailVerificationClaims(tokenClaims.Claims)
}

func (e EmailVerification) UserID() string {
	uid, ok, err := e.Claims.StringClaim("sub")
	if !ok || err != nil {
		panic("EmailVerification: no sub claim. This should be impossible.")
	}
	return uid
}

func (e EmailVerification) Email() string {
	email, ok, err := e.Claims.StringClaim(ClaimEmailVerificationEmail)
	if !ok || err != nil {
		panic("EmailVerification: no email claim. This should be impossible.")
	}
	return email
}

func (e EmailVerification) Callback() *url.URL {
	cb, ok, err := e.Claims.StringClaim(ClaimEmailVerificationCallback)
	if !ok || err != nil {
		panic("EmailVerification: no callback claim. This should be impossible.")
	}

	cbURL, err := url.Parse(cb)
	if err != nil {
		panic("EmailVerificaiton: can't parse callback. This should be impossible.")
	}
	return cbURL
}

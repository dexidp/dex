package user

import (
	"fmt"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oidc"
)

// NewEmailVerification creates an object which can be sent to a user
// in serialized form to verify that they control an email address.
// The clientID is the ID of the registering user. The callback is
// where a user should land after verifying their email.
func NewEmailVerification(user User, clientID string, issuer url.URL, callback url.URL, expires time.Duration) EmailVerification {
	claims := oidc.NewClaims(issuer.String(), user.ID, clientID, clock.Now(), clock.Now().Add(expires))
	claims.Add(ClaimEmailVerificationCallback, callback.String())
	claims.Add(ClaimEmailVerificationEmail, user.Email)
	return EmailVerification{claims}
}

type EmailVerification struct {
	Claims jose.Claims
}

// ParseAndVerifyEmailVerificationToken parses a string into a an
// EmailVerification, verifies the signature, and ensures that
// required claims are present.  In addition to the usual claims
// required by the OIDC spec, "aud" and "sub" must be present as well
// as ClaimEmailVerificationCallback and ClaimEmailVerificationEmail.
func ParseAndVerifyEmailVerificationToken(token string, issuer url.URL, keys []key.PublicKey) (EmailVerification, error) {
	tokenClaims, err := parseAndVerifyTokenClaims(token, issuer, keys)
	if err != nil {
		return EmailVerification{}, err
	}

	email, ok, err := tokenClaims.Claims.StringClaim(ClaimEmailVerificationEmail)
	if err != nil {
		return EmailVerification{}, err
	}
	if !ok || email == "" {
		return EmailVerification{}, fmt.Errorf("no %q claim", ClaimEmailVerificationEmail)
	}

	cb, ok, err := tokenClaims.Claims.StringClaim(ClaimEmailVerificationCallback)
	if err != nil {
		return EmailVerification{}, err
	}
	if !ok || cb == "" {
		return EmailVerification{}, fmt.Errorf("no %q claim", ClaimEmailVerificationCallback)
	}
	if _, err := url.Parse(cb); err != nil {
		return EmailVerification{}, fmt.Errorf("callback URL not parseable: %v", cb)
	}

	return EmailVerification{tokenClaims.Claims}, nil
}

func (e EmailVerification) UserID() string {
	return assertStringClaim(e.Claims, "sub")
}

func (e EmailVerification) Email() string {
	return assertStringClaim(e.Claims, ClaimEmailVerificationEmail)
}

func (e EmailVerification) Callback() *url.URL {
	return assertURLClaim(e.Claims, ClaimEmailVerificationCallback)
}

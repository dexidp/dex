package user

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/jonboulle/clockwork"

	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oidc"
)

const (
	// Claim representing where a user should be sent after verifying their email address.
	ClaimEmailVerificationCallback = "http://coreos.com/email/verification-callback"

	// ClaimEmailVerificationEmail represents the email to be verified. Note
	// that we are intentionally not using the "email" claim for this purpose.
	ClaimEmailVerificationEmail = "http://coreos.com/email/verificationEmail"
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
	claims jose.Claims
}

// Token serializes the EmailVerification into a signed JWT.
func (e EmailVerification) Token(signer jose.Signer) (string, error) {
	if signer == nil {
		return "", errors.New("no signer")
	}

	jwt, err := jose.NewSignedJWT(e.claims, signer)
	if err != nil {
		return "", err
	}

	return jwt.Encode(), nil
}

// ParseAndVerifyEmailVerificationToken parses a string into a an EmailVerification, verifies the signature, and ensures that required claims are present.
// In addition to the usual claims required by the OIDC spec, "aud" and "sub" must be present as well as ClaimEmailVerificationCallback and ClaimEmailVerificationEmail.
func ParseAndVerifyEmailVerificationToken(token string, issuer url.URL, keys []key.PublicKey) (EmailVerification, error) {
	jwt, err := jose.ParseJWT(token)
	if err != nil {
		return EmailVerification{}, err
	}

	claims, err := jwt.Claims()
	if err != nil {
		return EmailVerification{}, err
	}

	clientID, ok, err := claims.StringClaim("aud")
	if err != nil {
		return EmailVerification{}, err
	}
	if !ok {
		return EmailVerification{}, errors.New("no aud(client ID) claim")
	}

	cb, ok, err := claims.StringClaim(ClaimEmailVerificationCallback)
	if err != nil {
		return EmailVerification{}, err
	}
	if cb == "" {
		return EmailVerification{}, fmt.Errorf("no %q claim", ClaimEmailVerificationCallback)
	}
	if _, err := url.Parse(cb); err != nil {
		return EmailVerification{}, fmt.Errorf("callback URL not parseable: %v", cb)
	}

	email, ok, err := claims.StringClaim(ClaimEmailVerificationEmail)
	if err != nil {
		return EmailVerification{}, err
	}
	if email == "" {
		return EmailVerification{}, fmt.Errorf("no %q claim", ClaimEmailVerificationEmail)
	}

	sub, ok, err := claims.StringClaim("sub")
	if err != nil {
		return EmailVerification{}, err
	}
	if sub == "" {
		return EmailVerification{}, errors.New("no sub claim")
	}

	noop := func() error { return nil }

	keysFunc := func() []key.PublicKey {
		return keys
	}

	verifier := oidc.NewJWTVerifier(issuer.String(), clientID, noop, keysFunc)
	if err := verifier.Verify(jwt); err != nil {
		return EmailVerification{}, err
	}

	return EmailVerification{
		claims: claims,
	}, nil

}

func (e EmailVerification) UserID() string {
	uid, ok, err := e.claims.StringClaim("sub")
	if !ok || err != nil {
		panic("EmailVerification: no sub claim. This should be impossible.")
	}
	return uid
}

func (e EmailVerification) Email() string {
	email, ok, err := e.claims.StringClaim(ClaimEmailVerificationEmail)
	if !ok || err != nil {
		panic("EmailVerification: no email claim. This should be impossible.")
	}
	return email
}

func (e EmailVerification) Callback() *url.URL {
	cb, ok, err := e.claims.StringClaim(ClaimEmailVerificationCallback)
	if !ok || err != nil {
		panic("EmailVerification: no callback claim. This should be impossible.")
	}

	cbURL, err := url.Parse(cb)
	if err != nil {
		panic("EmailVerificaiton: can't parse callback. This should be impossible.")
	}
	return cbURL
}

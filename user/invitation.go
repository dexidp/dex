package user

import (
	"fmt"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oidc"
)

func NewInvitation(user User, password Password, issuer url.URL, clientID string, callback url.URL, expires time.Duration) Invitation {
	claims := oidc.NewClaims(issuer.String(), user.ID, clientID, clock.Now(), clock.Now().Add(expires))
	claims.Add(ClaimPasswordResetPassword, string(password))
	claims.Add(ClaimEmailVerificationEmail, user.Email)
	claims.Add(ClaimInvitationCallback, callback.String())
	return Invitation{claims}
}

// An Invitation is a token that can be used for verifying an email
// address and resetting a password in a single stroke. It will be
// sent as part of a link in an email automatically to newly created
// users if email is configured.
type Invitation struct {
	Claims jose.Claims
}

func ParseAndVerifyInvitationToken(token string, issuer url.URL, keys []key.PublicKey) (Invitation, error) {
	tokenClaims, err := parseAndVerifyTokenClaims(token, issuer, keys)
	if err != nil {
		return Invitation{}, err
	}

	cb, ok, err := tokenClaims.Claims.StringClaim(ClaimInvitationCallback)
	if err != nil {
		return Invitation{}, err
	}
	if !ok || cb == "" {
		return Invitation{}, fmt.Errorf("no %q claim", ClaimInvitationCallback)
	}
	if _, err := url.Parse(cb); err != nil {
		return Invitation{}, fmt.Errorf("callback URL not parseable: %v", cb)
	}

	pw, ok, err := tokenClaims.Claims.StringClaim(ClaimPasswordResetPassword)
	if err != nil {
		return Invitation{}, err
	}
	if !ok || pw == "" {
		return Invitation{}, fmt.Errorf("no %q claim", ClaimPasswordResetPassword)
	}

	email, ok, err := tokenClaims.Claims.StringClaim(ClaimEmailVerificationEmail)
	if err != nil {
		return Invitation{}, err
	}
	if !ok || email == "" {
		return Invitation{}, fmt.Errorf("no %q claim", ClaimEmailVerificationEmail)
	}

	return Invitation{tokenClaims.Claims}, nil
}

func (iv Invitation) UserID() string {
	return assertStringClaim(iv.Claims, "sub")
}

func (iv Invitation) Password() Password {
	pw := assertStringClaim(iv.Claims, ClaimPasswordResetPassword)
	return Password(pw)
}

func (iv Invitation) Email() string {
	return assertStringClaim(iv.Claims, ClaimEmailVerificationEmail)
}

func (iv Invitation) ClientID() string {
	return assertStringClaim(iv.Claims, "aud")
}

func (iv Invitation) Callback() *url.URL {
	return assertURLClaim(iv.Claims, ClaimInvitationCallback)
}

func (iv Invitation) PasswordReset(issuer url.URL, expires time.Duration) PasswordReset {
	return NewPasswordReset(
		iv.UserID(),
		iv.Password(),
		issuer,
		iv.ClientID(),
		*iv.Callback(),
		expires,
	)
}

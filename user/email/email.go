package email

import (
	"net/url"
	"time"

	"github.com/coreos/go-oidc/jose"

	"github.com/coreos/dex/email"
	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/user"
)

// UserEmailer provides functions for sending emails to Users.
type UserEmailer struct {
	ur                  user.UserRepo
	pwi                 user.PasswordInfoRepo
	signerFn            signerFunc
	tokenValidityWindow time.Duration
	issuerURL           url.URL
	emailer             *email.TemplatizedEmailer
	fromAddress         string

	passwordResetURL url.URL
	verifyEmailURL   url.URL
	invitationURL    url.URL
}

// NewUserEmailer creates a new UserEmailer.
func NewUserEmailer(ur user.UserRepo,
	pwi user.PasswordInfoRepo,
	signerFn signerFunc,
	tokenValidityWindow time.Duration,
	issuerURL url.URL,
	emailer *email.TemplatizedEmailer,
	fromAddress string,
	passwordResetURL url.URL,
	verifyEmailURL url.URL,
	invitationURL url.URL,
) *UserEmailer {
	return &UserEmailer{
		ur:                  ur,
		pwi:                 pwi,
		signerFn:            signerFn,
		tokenValidityWindow: tokenValidityWindow,
		issuerURL:           issuerURL,
		emailer:             emailer,
		fromAddress:         fromAddress,
		passwordResetURL:    passwordResetURL,
		verifyEmailURL:      verifyEmailURL,
		invitationURL:       invitationURL,
	}
}

func (u *UserEmailer) userPasswordInfo(email string) (user.User, user.PasswordInfo, error) {
	usr, err := u.ur.GetByEmail(nil, email)
	if err != nil {
		log.Errorf("Error getting user: %q", err)
		return user.User{}, user.PasswordInfo{}, err
	}

	pwi, err := u.pwi.Get(nil, usr.ID)
	if err != nil {
		log.Errorf("Error getting password: %q", err)
		return user.User{}, user.PasswordInfo{}, err
	}

	return usr, pwi, nil
}

func (u *UserEmailer) signedClaimsToken(claims jose.Claims) (string, error) {
	signer, err := u.signerFn()
	if err != nil || signer == nil {
		log.Errorf("error getting signer: %v (%v)", err, signer)
		return "", err
	}

	jwt, err := jose.NewSignedJWT(claims, signer)
	if err != nil {
		log.Errorf("error constructing or signing a JWT: %v", err)
		return "", err
	}
	return jwt.Encode(), nil
}

// SendResetPasswordEmail sends a password reset email to the user specified by the email addresss, containing a link with a signed token which can be visitied to initiate the password change/reset process.
// This method DOES NOT check for client ID, redirect URL validity - it is expected that upstream users have already done so.
// A link that can be used to reset the given user's password is returned.
func (u *UserEmailer) SendResetPasswordEmail(email string, redirectURL url.URL, clientID string) (*url.URL, error) {
	usr, pwi, err := u.userPasswordInfo(email)
	if err != nil {
		return nil, err
	}

	passwordReset := user.NewPasswordReset(usr.ID, pwi.Password, u.issuerURL,
		clientID, redirectURL, u.tokenValidityWindow)

	token, err := u.signedClaimsToken(passwordReset.Claims)
	if err != nil {
		return nil, err
	}

	resetURL := u.passwordResetURL
	q := resetURL.Query()
	q.Set("token", token)
	resetURL.RawQuery = q.Encode()

	if u.emailer != nil {
		err = u.emailer.SendMail(u.fromAddress, "Reset Your Password", "password-reset",
			map[string]interface{}{
				"email": usr.Email,
				"link":  resetURL.String(),
			}, usr.Email)
		if err != nil {
			log.Errorf("error sending password reset email %v: ", err)
		}
		return nil, err
	}
	return &resetURL, nil
}

// SendInviteEmail is sends an email that allows the user to both
// reset their password *and* verify their email address. Similar to
// SendResetPasswordEmail, the given url and client id are assumed
// valid. A link that can be used to validate the given email address
// and reset the password is returned.
func (u *UserEmailer) SendInviteEmail(email string, redirectURL url.URL, clientID string) (*url.URL, error) {
	usr, pwi, err := u.userPasswordInfo(email)
	if err != nil {
		return nil, err
	}

	invitation := user.NewInvitation(usr, pwi.Password, u.issuerURL,
		clientID, redirectURL, u.tokenValidityWindow)

	token, err := u.signedClaimsToken(invitation.Claims)
	if err != nil {
		return nil, err
	}

	resetURL := u.invitationURL
	q := resetURL.Query()
	q.Set("token", token)
	resetURL.RawQuery = q.Encode()

	if u.emailer != nil {
		err = u.emailer.SendMail(u.fromAddress, "Activate Your Account", "invite",
			map[string]interface{}{
				"email": usr.Email,
				"link":  resetURL.String(),
			}, usr.Email)
		if err != nil {
			log.Errorf("error sending password reset email %v: ", err)
		}
		return nil, err
	}
	return &resetURL, nil
}

// SendEmailVerification sends an email to the user with the given userID containing a link which when visited marks the user as having had their email verified.
// If there is no emailer is configured, the URL of the aforementioned link is returned, otherwise nil is returned.
func (u *UserEmailer) SendEmailVerification(userID, clientID string, redirectURL url.URL) (*url.URL, error) {
	usr, err := u.ur.Get(nil, userID)
	if err == user.ErrorNotFound {
		log.Errorf("No Such user for ID: %q", userID)
		return nil, err
	}
	if err != nil {
		log.Errorf("Error getting user: %q", err)
		return nil, err
	}

	ev := user.NewEmailVerification(usr, clientID, u.issuerURL, redirectURL, u.tokenValidityWindow)

	signer, err := u.signerFn()
	if err != nil || signer == nil {
		log.Errorf("error getting signer: %v (signer: %v)", err, signer)
		return nil, err
	}

	jwt, err := jose.NewSignedJWT(ev.Claims, signer)
	if err != nil {
		log.Errorf("error constructing or signing EmailVerification JWT: %v", err)
		return nil, err
	}
	token := jwt.Encode()

	verifyURL := u.verifyEmailURL
	q := verifyURL.Query()
	q.Set("token", token)
	verifyURL.RawQuery = q.Encode()

	if u.emailer != nil {
		err = u.emailer.SendMail(u.fromAddress, "Please verify your email address.", "verify-email",
			map[string]interface{}{
				"email": usr.Email,
				"link":  verifyURL.String(),
			}, usr.Email)
		if err != nil {
			log.Errorf("error sending email verification email %v: ", err)
		}
		return nil, err

	}
	return &verifyURL, nil
}

func (u *UserEmailer) SetEmailer(emailer *email.TemplatizedEmailer) {
	u.emailer = emailer
}

type signerFunc func() (jose.Signer, error)

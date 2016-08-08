package oidc

import (
	"errors"

	"golang.org/x/oauth2"
)

// Nonce returns an auth code option which requires the ID Token created by the
// OpenID Connect provider to contain the specified nonce.
func Nonce(nonce string) oauth2.AuthCodeOption {
	return oauth2.SetAuthURLParam("nonce", nonce)
}

// NonceSource represents a source which can verify a nonce is valid and has not
// been claimed before.
type NonceSource interface {
	ClaimNonce(nonce string) error
}

// VerifyNonce ensures that the ID Token contains a nonce which can be claimed by the nonce source.
func VerifyNonce(source NonceSource) VerificationOption {
	return nonceVerifier{source}
}

type nonceVerifier struct {
	nonceSource NonceSource
}

func (n nonceVerifier) verifyIDToken(token *IDToken) error {
	if token.Nonce == "" {
		return errors.New("oidc: no nonce present in ID Token")
	}
	return n.nonceSource.ClaimNonce(token.Nonce)
}

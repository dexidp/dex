package signer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	jose "github.com/go-jose/go-jose/v4"
)

// TypeAccessToken is the JWT "typ" header (RFC 9068) of the access tokens Dex
// issues to clients that have an ID-JAG policy.
const TypeAccessToken = "at+jwt"

// KeySet implements the oidc.KeySet interface backed by a Dex Signer,
// verifying a JWT's signature against the signer's current validation keys.
type KeySet struct {
	Signer Signer
	// AcceptAccessTokens also accepts tokens typed as access tokens. Only
	// verifiers that take an access token, such as userinfo and introspection,
	// set it, so a typed access token is refused where an ID token is required.
	AcceptAccessTokens bool
}

func (k *KeySet) VerifySignature(ctx context.Context, jwt string) (payload []byte, err error) {
	jws, err := jose.ParseSigned(jwt, []jose.SignatureAlgorithm{jose.RS256, jose.RS384, jose.RS512, jose.ES256, jose.ES384, jose.ES512})
	if err != nil {
		return nil, err
	}

	// ID tokens and untyped access tokens carry no typ or "JWT". Typed access
	// tokens carry "at+jwt" and pass only when AcceptAccessTokens is set. Other
	// typed tokens, such as ID-JAGs, never pass, because Dex does not verify
	// them itself.
	if typ, ok := jws.Signatures[0].Protected.ExtraHeaders[jose.HeaderType]; ok {
		s, _ := typ.(string)
		accessToken := strings.EqualFold(s, TypeAccessToken) || strings.EqualFold(s, "application/"+TypeAccessToken)
		if !strings.EqualFold(s, "JWT") && !(k.AcceptAccessTokens && accessToken) {
			return nil, fmt.Errorf("JWT typ %v is not accepted", typ)
		}
	}

	keyID := ""
	for _, sig := range jws.Signatures {
		keyID = sig.Header.KeyID
		break
	}

	keys, err := k.Signer.ValidationKeys(ctx)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		if keyID == "" || key.KeyID == keyID {
			if payload, err := jws.Verify(key); err == nil {
				return payload, nil
			}
		}
	}

	return nil, errors.New("failed to verify id token signature")
}

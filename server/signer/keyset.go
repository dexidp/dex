package signer

import (
	"context"
	"errors"

	jose "github.com/go-jose/go-jose/v4"

	"github.com/dexidp/dex/server/oauth2"
)

// KeySet implements the oidc.KeySet interface backed by a Dex Signer,
// verifying a JWT's signature against the signer's current validation keys.
type KeySet struct {
	Signer Signer
}

func (k *KeySet) VerifySignature(ctx context.Context, jwt string) (payload []byte, err error) {
	jws, err := jose.ParseSigned(jwt, []jose.SignatureAlgorithm{jose.RS256, jose.RS384, jose.RS512, jose.ES256, jose.ES384, jose.ES512})
	if err != nil {
		return nil, err
	}

	// An ID-JAG is an assertion for a third-party authorization server and is
	// never accepted where dex verifies its own tokens.
	if typ, _ := jws.Signatures[0].Protected.ExtraHeaders[jose.HeaderType].(string); typ == oauth2.JWTTypeIDJAG {
		return nil, errors.New("ID-JAG is not accepted")
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

package signer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	jose "github.com/go-jose/go-jose/v4"
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

	// Dex sets typ only on tokens it never verifies itself, such as ID-JAGs.
	if typ, ok := jws.Signatures[0].Protected.ExtraHeaders[jose.HeaderType]; ok {
		if s, _ := typ.(string); !strings.EqualFold(s, "JWT") {
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

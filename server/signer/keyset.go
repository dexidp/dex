package signer

import (
	"context"
	"errors"

	jose "github.com/go-jose/go-jose/v4"
)

// SupportedSigningAlgs lists every JWS signing algorithm a Dex signer (local or
// Vault) can produce. Verifiers for Dex-issued tokens must allow all of them:
// otherwise go-oidc's IDTokenVerifier.Verify defaults to RS256 only and rejects
// ES256/EdDSA tokens before KeySet.VerifySignature ever runs.
var SupportedSigningAlgs = []jose.SignatureAlgorithm{
	jose.RS256, jose.RS384, jose.RS512,
	jose.ES256, jose.ES384, jose.ES512,
	jose.EdDSA,
}

// SupportedSigningAlgStrings returns SupportedSigningAlgs as strings, for use in
// oidc.Config.SupportedSigningAlgs.
func SupportedSigningAlgStrings() []string {
	algs := make([]string, len(SupportedSigningAlgs))
	for i, alg := range SupportedSigningAlgs {
		algs[i] = string(alg)
	}
	return algs
}

// KeySet implements the oidc.KeySet interface backed by a Dex Signer,
// verifying a JWT's signature against the signer's current validation keys.
type KeySet struct {
	Signer Signer
}

func (k *KeySet) VerifySignature(ctx context.Context, jwt string) (payload []byte, err error) {
	jws, err := jose.ParseSigned(jwt, SupportedSigningAlgs)
	if err != nil {
		return nil, err
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

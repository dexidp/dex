package signer

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"errors"
	"fmt"

	"github.com/go-jose/go-jose/v4"
)

// signatureAlgorithm returns the JOSE signing algorithm declared by the JWK or
// inferred from its key material when the algorithm field is empty.
func signatureAlgorithm(jwk *jose.JSONWebKey) (alg jose.SignatureAlgorithm, err error) {
	if jwk.Key == nil {
		return alg, errors.New("no signing key")
	}
	if jwk.Algorithm != "" {
		return jose.SignatureAlgorithm(jwk.Algorithm), nil
	}
	return signatureAlgorithmFromKey(jwk.Key)
}

func signatureAlgorithmFromKey(key any) (alg jose.SignatureAlgorithm, err error) {
	switch key := key.(type) {
	case *rsa.PublicKey, *rsa.PrivateKey:
		// Because OIDC mandates that we support RS256, we always return that
		// value. In the future, we might want to make this configurable on a
		// per client basis. For example allowing PS256 or ECDSA variants.
		//
		// See https://github.com/dexidp/dex/issues/692
		return jose.RS256, nil
	case *ecdsa.PublicKey:
		return signatureAlgorithmFromECDSACurve(key.Curve)
	case *ecdsa.PrivateKey:
		return signatureAlgorithmFromECDSACurve(key.Curve)
	case ed25519.PublicKey, ed25519.PrivateKey:
		return jose.EdDSA, nil
	default:
		return alg, fmt.Errorf("unsupported signing key type %T", key)
	}
}

func signatureAlgorithmFromECDSACurve(curve elliptic.Curve) (jose.SignatureAlgorithm, error) {
	if curve == nil {
		return "", errors.New("unsupported ecdsa curve")
	}
	switch curve.Params() {
	case elliptic.P256().Params():
		return jose.ES256, nil
	case elliptic.P384().Params():
		return jose.ES384, nil
	case elliptic.P521().Params():
		return jose.ES512, nil
	default:
		return "", errors.New("unsupported ecdsa curve")
	}
}

func signPayload(key *jose.JSONWebKey, alg jose.SignatureAlgorithm, payload []byte) (jws string, err error) {
	signingKey := jose.SigningKey{Key: key, Algorithm: alg}

	signer, err := jose.NewSigner(signingKey, &jose.SignerOptions{})
	if err != nil {
		return "", fmt.Errorf("new signer: %v", err)
	}
	signature, err := signer.Sign(payload)
	if err != nil {
		return "", fmt.Errorf("signing payload: %v", err)
	}
	return signature.CompactSerialize()
}

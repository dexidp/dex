package signer

import (
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"

	"gopkg.in/square/go-jose.v2"
)

// The hash algorithm for the at_hash is determined by the signing
// algorithm used for the id_token. From the spec:
//
//    ...the hash algorithm used is the hash algorithm used in the alg Header
//    Parameter of the ID Token's JOSE Header. For instance, if the alg is RS256,
//    hash the access_token value with SHA-256
//
// https://openid.net/specs/openid-connect-core-1_0.html#ImplicitIDToken
var hashForSigAlgo = map[jose.SignatureAlgorithm]func() hash.Hash{
	jose.RS256: sha256.New,
	jose.RS384: sha512.New384,
	jose.RS512: sha512.New,
	jose.ES256: sha256.New,
	jose.ES384: sha512.New384,
	jose.ES512: sha512.New,
	// Ed25519 use SHA-512 internally
	// https://bitbucket.org/openid/connect/issues/1125/_hash-algorithm-for-eddsa-id-tokens
	// XXX: This does not applies to Ed448
	jose.EdDSA: sha512.New,
}

func HashForSigAlgorithm(alg jose.SignatureAlgorithm) (hash.Hash, error) {
	newHash, ok := hashForSigAlgo[alg]
	if !ok {
		return nil, fmt.Errorf("unsupported signature algorithm: %s", alg)
	}

	return newHash(), nil
}

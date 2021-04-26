package storage

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"errors"
	"fmt"
	"hash"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"gopkg.in/square/go-jose.v2"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/signer"
	"github.com/dexidp/dex/storage"
)

type Signer struct {
	storage          storage.KeyStorage
	logger           log.Logger
	now              func() time.Time
	rotationStrategy RotationStrategy
}

func New(storage storage.KeyStorage, logger log.Logger, strategy RotationStrategy) *Signer {
	now := time.Now
	return &Signer{
		storage:          newKeyCacher(storage, now),
		now:              now,
		logger:           logger,
		rotationStrategy: strategy,
	}
}

func (s *Signer) GetSigningKeys() (signer.SigningKeyResponse, error) {
	keys, err := s.storage.GetKeys()
	if err != nil {
		return signer.SigningKeyResponse{}, err
	}

	if keys.SigningKeyPub == nil {
		return signer.SigningKeyResponse{}, errors.New("no public keys found")
	}

	jwks := jose.JSONWebKeySet{
		Keys: make([]jose.JSONWebKey, len(keys.VerificationKeys)+1),
	}
	jwks.Keys[0] = *keys.SigningKeyPub
	for i, verificationKey := range keys.VerificationKeys {
		jwks.Keys[i+1] = *verificationKey.PublicKey
	}

	return signer.SigningKeyResponse{
		Jwks:         jwks,
		NextRotation: &keys.NextRotation,
	}, nil
}

func (s *Signer) Hasher() (hash.Hash, error) {
	keys, err := s.storage.GetKeys()
	if err != nil {
		return nil, err
	}
	sigAlgo, err := signatureAlgorithm(keys.SigningKey)
	if err != nil {
		return nil, err
	}
	return signer.HashForSigAlgorithm(sigAlgo)
}

func (s *Signer) Sign(payload []byte) (jws string, err error) {
	keys, err := s.storage.GetKeys()
	if err != nil {
		return "", err
	}
	sigAlgo, err := signatureAlgorithm(keys.SigningKey)
	if err != nil {
		return "", err
	}

	signingKey := jose.SigningKey{
		Key:       keys.SigningKey,
		Algorithm: sigAlgo,
	}

	signer, err := jose.NewSigner(signingKey, &jose.SignerOptions{})
	if err != nil {
		return "", fmt.Errorf("new signier: %v", err)
	}
	signature, err := signer.Sign(payload)
	if err != nil {
		return "", fmt.Errorf("signing payload: %v", err)
	}
	return signature.CompactSerialize()
}

func (s *Signer) GetKeySet() (oidc.KeySet, error) {
	return &storageKeySet{
		KeyStorage: s.storage,
	}, nil
}

// Determine the signature algorithm for a JWT.
func signatureAlgorithm(jwk *jose.JSONWebKey) (alg jose.SignatureAlgorithm, err error) {
	if jwk.Key == nil {
		return alg, errors.New("no signing key")
	}
	switch key := jwk.Key.(type) {
	case *rsa.PrivateKey, *rsa.PublicKey:
		return jose.RS256, nil
	case *ecdsa.PrivateKey, *ecdsa.PublicKey:
		keyCurve := key.(elliptic.Curve)
		switch keyCurve.Params() {
		case elliptic.P256().Params():
			return jose.ES256, nil
		case elliptic.P384().Params():
			return jose.ES384, nil
		case elliptic.P521().Params():
			return jose.ES512, nil
		default:
			return alg, errors.New("unsupported ecdsa curve")
		}
	default:
		return alg, fmt.Errorf("unsupported signing key type %T", key)
	}
}

package signer

import (
	"errors"
	"hash"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"gopkg.in/square/go-jose.v2"
)

type Signer interface {
	GetSigningKeys() (SigningKeyResponse, error)
	GetKeySet() (oidc.KeySet, error)

	Sign(payload []byte) (jws string, err error)

	// Hasher return new instance of hash.Hash used to sign access token
	Hasher() (hash.Hash, error)

	RotateKey() error
}

type SigningKeyResponse struct {
	Jwks         jose.JSONWebKeySet
	NextRotation *time.Time
}

var (
	ErrAlreadyRotated       = errors.New("keys already rotated by another server instance")
	ErrRotationNotSupported = errors.New("key rotation not supported")
)

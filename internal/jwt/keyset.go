package jwt

import (
	"context"
	"errors"

	"github.com/go-jose/go-jose/v4"

	"github.com/dexidp/dex/storage"
)

var ErrFailedVerify = errors.New("failed to verify id token signature")

// StorageKeySet implements the oidc.KeySet interface backed by Dex storage
type StorageKeySet struct {
	storage.Storage
}

func NewStorageKeySet(store storage.Storage) *StorageKeySet {
	return &StorageKeySet{
		store,
	}
}

func (s *StorageKeySet) VerifySignature(ctx context.Context, jwt string) (payload []byte, err error) {
	jws, err := jose.ParseSigned(jwt, []jose.SignatureAlgorithm{jose.RS256, jose.RS384, jose.RS512, jose.ES256, jose.ES384, jose.ES512})
	if err != nil {
		return nil, err
	}

	keyID := ""
	for _, sig := range jws.Signatures {
		keyID = sig.Header.KeyID
		break
	}

	skeys, err := s.Storage.GetKeys(ctx)
	if err != nil {
		return nil, err
	}

	keys := []*jose.JSONWebKey{skeys.SigningKeyPub}
	for _, vk := range skeys.VerificationKeys {
		keys = append(keys, vk.PublicKey)
	}

	for _, key := range keys {
		if keyID == "" || key.KeyID == keyID {
			if payload, err := jws.Verify(key); err == nil {
				return payload, nil
			}
		}
	}

	return nil, ErrFailedVerify
}

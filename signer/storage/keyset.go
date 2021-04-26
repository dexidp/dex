package storage

import (
	"context"
	"errors"

	"gopkg.in/square/go-jose.v2"

	"github.com/dexidp/dex/storage"
)

// storageKeySet implements the oidc.KeySet interface backed by Dex storage
type storageKeySet struct {
	storage.KeyStorage
}

func (s *storageKeySet) VerifySignature(_ context.Context, jwt string) (payload []byte, err error) {
	jws, err := jose.ParseSigned(jwt)
	if err != nil {
		return nil, err
	}

	keyID := ""
	for _, sig := range jws.Signatures {
		keyID = sig.Header.KeyID
		break
	}

	skeys, err := s.KeyStorage.GetKeys()
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

	return nil, errors.New("failed to verify id token signature")
}

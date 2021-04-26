package vault

import (
	"context"
	"errors"

	"golang.org/x/sync/singleflight"
	"gopkg.in/square/go-jose.v2"
)

// vaultKeySet implements oidc.KeySet
type vaultKeySet struct {
	signer *Signer

	cachedJwk   map[string]jose.JSONWebKey
	cacheFiller singleflight.Group
}

func (v *vaultKeySet) VerifySignature(ctx context.Context, jwt string) (payload []byte, err error) {
	jws, err := jose.ParseSigned(jwt)
	if err != nil {
		return nil, err
	}

	keyID := ""
	for _, sig := range jws.Signatures {
		keyID = sig.Header.KeyID
		break
	}

	jwk, err := v.getKeyVersion(keyID)
	if err != nil {
		return nil, err
	}

	return jws.Verify(jwk)
}

func (v *vaultKeySet) getKeyVersion(keyID string) (jose.JSONWebKey, error) {
	cached, ok := v.cachedJwk[keyID]
	if ok {
		return cached, nil
	}

	out, err, _ := v.cacheFiller.Do(keyID, func() (interface{}, error) {
		keyInfo, err := v.signer.getKeyInfo()
		if err != nil {
			return jose.JSONWebKey{}, err
		}

		keyVersion, ok := keyInfo.Versions[keyID]
		if !ok {
			return jose.JSONWebKey{}, errors.New("unknown key version")
		}

		out, err := keyVersionToJwk(keyInfo.Type, keyID, keyVersion)
		if err == nil {
			// cache fill
			v.cachedJwk[keyID] = out
		}

		return out, err
	})

	if err != nil {
		return jose.JSONWebKey{}, err
	}

	return out.(jose.JSONWebKey), nil
}

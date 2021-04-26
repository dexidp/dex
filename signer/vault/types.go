package vault

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strconv"

	"golang.org/x/crypto/ed25519"
	"gopkg.in/square/go-jose.v2"
)

type keyInfo struct {
	Type     string                    `mapstructure:"type"`
	Versions map[string]keyVersionInfo `mapstructure:"keys"`
}

func (k *keyInfo) LatestVersion() int {
	var latestVersion int64 = -1
	for v := range k.Versions {
		version, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			continue
		}

		if version > latestVersion {
			latestVersion = version
		}
	}

	return int(latestVersion)
}

type keyVersionInfo struct {
	CreationTime string `mapstructure:"creation_time"`
	Name         string `mapstructure:"name"`
	PublicKey    string `mapstructure:"public_key"`
}

func (k keyVersionInfo) getPublicKeyEd25519() (ed25519.PublicKey, error) {
	return base64.StdEncoding.DecodeString(k.PublicKey)
}

func (k keyVersionInfo) getPublicKeyPkix() (interface{}, error) {
	block, _ := pem.Decode([]byte(k.PublicKey))
	return x509.ParsePKIXPublicKey(block.Bytes)
}

func (k keyVersionInfo) GetPublicKey(keyType string) (interface{}, error) {
	switch keyType {
	case "ed25519":
		return k.getPublicKeyEd25519()
	default:
		return k.getPublicKeyPkix()
	}
}

// sigAlgoMapping contains mapping from Vault key types (https://www.vaultproject.io/docs/secrets/transit#key-types)
// to jose.SignatureAlgorithm
var sigAlgoMapping = map[string]jose.SignatureAlgorithm{
	"ed25519":    jose.EdDSA,
	"ecdsa-p256": jose.ES256,
	"ecdsa-p384": jose.ES384,
	"ecdsa-p521": jose.ES512,
	"rsa-2048":   jose.RS256,
	"rsa-3072":   jose.RS384,
	"rsa-4096":   jose.RS512,
}

// sigHashMapping must match storage.hashForSigAlgo
var sigHashMapping = map[string]string{
	// "ed25519":    "",  // omit for vault to pick
	"ecdsa-p256": "sha2-256",
	"ecdsa-p384": "sha2-384",
	"ecdsa-p521": "sha2-512",
	"rsa-2048":   "sha2-256",
	"rsa-3072":   "sha2-384",
	"rsa-4096":   "sha2-512",
}

func keyInfoToJwks(info keyInfo) (jose.JSONWebKeySet, error) {
	out := jose.JSONWebKeySet{}
	for keyID, value := range info.Versions {
		key, err := keyVersionToJwk(info.Type, keyID, value)
		if err != nil {
			return jose.JSONWebKeySet{}, fmt.Errorf("cannot parse key version %s: %w", keyID, err)
		}
		out.Keys = append(out.Keys, key)
	}

	if len(out.Keys) == 0 {
		return out, errors.New("no keys found")
	}

	return out, nil
}

func keyVersionToJwk(keyType string, keyID string, version keyVersionInfo) (jose.JSONWebKey, error) {
	algo, ok := sigAlgoMapping[keyType]
	if !ok {
		return jose.JSONWebKey{}, fmt.Errorf("unsupported algorithm %s", keyType)
	}

	pubkey, err := version.GetPublicKey(keyType)
	if err != nil {
		return jose.JSONWebKey{}, fmt.Errorf("unable to parse key %s: %w", keyID, err)
	}

	return jose.JSONWebKey{
		Key:       pubkey,
		KeyID:     keyID,
		Algorithm: string(algo),
		Use:       "sig",
	}, nil
}

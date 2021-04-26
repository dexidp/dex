package vault

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"strconv"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	vault "github.com/hashicorp/vault/api"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/sync/singleflight"
	"gopkg.in/square/go-jose.v2"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/signer"
)

type Signer struct {
	vault  *vault.Client
	config Config
	logger log.Logger
	keySet vaultKeySet

	keyAlgo      string
	singleflight singleflight.Group
}

func (s *Signer) init() error {
	s.logger.Info("vault: getting key info")
	info, err := s.getKeyInfo()
	if err != nil {
		return err
	}
	s.keyAlgo = info.Type
	s.keySet = vaultKeySet{
		signer:    s,
		cachedJwk: make(map[string]jose.JSONWebKey),
	}
	return nil
}

func (s *Signer) getKeyInfo() (keyInfo, error) {
	info, err, _ := s.singleflight.Do("keyInfo", func() (interface{}, error) {
		data, err := s.vault.Logical().Read(fmt.Sprintf("%s/keys/%s", s.config.TransitMount, s.config.KeyName))
		if err != nil {
			return nil, err
		}
		if data == nil {
			return nil, errors.New("cannot get key information")
		}

		var info keyInfo
		if err = mapstructure.Decode(data.Data, &info); err != nil {
			return keyInfo{}, err
		}
		return info, nil
	})

	if err != nil {
		return keyInfo{}, err
	}

	return info.(keyInfo), nil
}

func (s *Signer) Hasher() (hash.Hash, error) {
	return signer.HashForSigAlgorithm(sigAlgoMapping[s.keyAlgo])
}

func (s *Signer) GetSigningKeys() (signer.SigningKeyResponse, error) {
	keys, err := s.getKeyInfo()
	if err != nil {
		return signer.SigningKeyResponse{}, err
	}
	jwks, err := keyInfoToJwks(keys)
	if err != nil {
		return signer.SigningKeyResponse{}, err
	}

	return signer.SigningKeyResponse{
		Jwks: jwks,
	}, nil
}

func (s *Signer) GetKeySet() (oidc.KeySet, error) {
	return &s.keySet, nil
}

func (s *Signer) Sign(payload []byte) (string, error) {
	info, err := s.getKeyInfo()
	if err != nil {
		return "", err
	}
	algo := info.Type
	latestKeyVersion := info.LatestVersion()

	header := map[jose.HeaderKey]string{
		"alg": string(sigAlgoMapping[algo]),
		"kid": strconv.Itoa(latestKeyVersion),
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	signedPayloadBuf, err := s.buildSignedPayload(headerJSON, payload)
	if err != nil {
		return "", err
	}

	var input string
	preHashed := true
	if algo == "ed25519" {
		// ed25519 does not support prehashing
		input = base64.StdEncoding.EncodeToString(signedPayloadBuf)
		preHashed = false
	} else {
		hasher, err := s.Hasher()
		if err != nil {
			return "", err
		}
		hasher.Write(signedPayloadBuf)

		var hashedPayload []byte
		hashedPayload = hasher.Sum(hashedPayload)

		input = base64.StdEncoding.EncodeToString(hashedPayload)
	}

	res, err := s.vault.Logical().Write(fmt.Sprintf("%s/sign/%s", s.config.TransitMount, s.config.KeyName), map[string]interface{}{
		"hash_algorithm":       sigHashMapping[algo],
		"input":                input,
		"prehashed":            preHashed,
		"signature_algorithm":  "pkcs1v15",
		"marshaling_algorithm": "jws",
		"key_version":          latestKeyVersion,
	})
	if err != nil {
		return "", err
	}

	sig, ok := res.Data["signature"].(string)
	if !ok {
		return "", errors.New("no signature returned from vault")
	}
	sigParts := strings.SplitN(sig, ":", 3)

	return string(signedPayloadBuf) + "." + sigParts[2], nil
}

//nolint:unparam
func (s *Signer) buildSignedPayload(header []byte, payload []byte) ([]byte, error) {
	signedPayloadBuf := bytes.Buffer{}

	b64encoder := base64.NewEncoder(base64.RawURLEncoding, &signedPayloadBuf)
	b64encoder.Write(header)
	b64encoder.Close()

	signedPayloadBuf.WriteRune('.')

	b64encoder = base64.NewEncoder(base64.RawURLEncoding, &signedPayloadBuf)
	b64encoder.Write(payload)
	b64encoder.Close()

	return signedPayloadBuf.Bytes(), nil
}

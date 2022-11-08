package secret

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"gopkg.in/square/go-jose.v2"
)

type Config struct {
	// TeamId defines the 10-character Team ID associated with a developer account
	// See https://developer.apple.com/documentation/sign_in_with_apple/generate_and_validate_tokens
	TeamID string
	// KeyID defines the key id for private key file
	KeyID string

	// PrivateKeyFile defines a PEM encoded private key file
	PrivateKeyFile string

	// SecretDuration defines the number of seconds a jwt secret will be valid for
	SecretDuration int64

	// SecretExpiryMin defines the minimum number of seconds from current time
	// before declaring a secret has expired
	SecretExpiryMin int64

	Issuer   string
	ClientID string
}

type Secret struct {
	cfg           Config
	currentSecret string
	claims        *claims
}

type claims struct {
	Iss string `json:"iss"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
	Aud string `json:"aud"`
	Sub string `json:"sub"`
}

func NewSecret(cfg *Config) (*Secret, error) {
	if len(cfg.TeamID) == 0 {
		return nil, fmt.Errorf("missing team_id configuration")
	}
	if len(cfg.KeyID) == 0 {
		return nil, fmt.Errorf("missing key_id configuration")
	}
	if len(cfg.PrivateKeyFile) == 0 {
		return nil, fmt.Errorf("missing private_key_file configuration")
	}
	if cfg.SecretDuration == 0 {
		// Max duration is currently 6 months
		cfg.SecretDuration = 15777000
	}
	if cfg.SecretExpiryMin == 0 {
		cfg.SecretExpiryMin = 30
	}
	return &Secret{cfg: *cfg}, nil
}

func parseKeyFile(keyfile string) (interface{}, error) {
	raw, err := os.ReadFile(keyfile)
	if err != nil {
		return "", err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM block")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	switch key.(type) {
	case *rsa.PrivateKey, *ecdsa.PrivateKey:
	default:
		return "", fmt.Errorf("found unknown private key type in PKCS#8 wrapping")
	}
	return key, nil
}

func (s *Secret) IsExpired() bool {
	if s.claims != nil {
		if time.Now().UTC().Add(time.Second*time.Duration(s.cfg.SecretExpiryMin)).Unix() < s.claims.Exp {
			return false
		}
	}
	return true
}

func (s *Secret) GetSecret() (string, error) {
	claims := claims{
		Iss: s.cfg.TeamID,
		Iat: time.Now().Unix(),
		Exp: time.Now().UTC().Add(time.Second * time.Duration(s.cfg.SecretDuration)).Unix(),
		Aud: s.cfg.Issuer,
		Sub: s.cfg.ClientID,
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	key, err := parseKeyFile(s.cfg.PrivateKeyFile)
	if err != nil {
		return "", err
	}

	signingKey := jose.SigningKey{Key: key, Algorithm: jose.ES256}

	signerOpts := jose.SignerOptions{}
	signerOpts.WithHeader("kid", s.cfg.KeyID)

	signer, err := jose.NewSigner(signingKey, &signerOpts)
	if err != nil {
		return "", fmt.Errorf("new signer: %v", err)
	}

	signature, err := signer.Sign(payload)
	if err != nil {
		return "", fmt.Errorf("signing payload: %v", err)
	}
	s.currentSecret, err = signature.CompactSerialize()
	if err != nil {
		return "", err
	}
	s.claims = &claims
	return s.currentSecret, nil
}

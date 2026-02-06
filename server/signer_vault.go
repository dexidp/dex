package server

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"hash"

	"github.com/go-jose/go-jose/v4"
	vault "github.com/hashicorp/vault/api"
)

// VaultSignerConfig holds configuration for the Vault signer.
type VaultSignerConfig struct {
	Addr    string `json:"addr"`
	Token   string `json:"token"`
	KeyName string `json:"keyName"`
}

// vaultSigner signs payloads using HashiCorp Vault's Transit backend.
type vaultSigner struct {
	client  *vault.Client
	keyName string
}

// newVaultSigner creates a new Vault signer that uses Transit backend for signing.
func newVaultSigner(c VaultSignerConfig) (*vaultSigner, error) {
	config := vault.DefaultConfig()
	config.Address = c.Addr

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %v", err)
	}

	if c.Token != "" {
		client.SetToken(c.Token)
	}

	return &vaultSigner{
		client:  client,
		keyName: c.KeyName,
	}, nil
}

func (v *vaultSigner) Start(ctx context.Context) {
	// Vault signer does not need background rotation tasks
}

func (v *vaultSigner) Sign(ctx context.Context, payload []byte) (string, error) {
	// 1. Fetch keys to determine the key to use (latest version) and its ID.
	keysMap, latestVersion, err := v.getTransitKeysMap(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get keys for signing context: %v", err)
	}

	// Determine the key version and ID to use
	// We use the latest version by default
	signingJWK, ok := keysMap[latestVersion]
	if !ok {
		return "", fmt.Errorf("latest key version %d not found in public keys", latestVersion)
	}

	// 2. Construct JWS Header and Payload first (Signing Input)
	header := map[string]interface{}{
		"alg": signingJWK.Algorithm,
		"kid": signingJWK.KeyID,
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %v", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerBytes)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)

	// The input to the signature is "header.payload"
	signingInput := fmt.Sprintf("%s.%s", headerB64, payloadB64)

	// 3. Sign the signingInput using Vault
	var vaultInput string
	data := map[string]interface{}{}

	// Determine Vault params based on JWS algorithm
	params, err := getVaultParams(signingJWK.Algorithm)
	if err != nil {
		return "", err
	}

	// Apply params to data map
	for k, v := range params.extraParams {
		data[k] = v
	}

	// Hash input if needed
	if params.hasher != nil {
		params.hasher.Write([]byte(signingInput))
		hash := params.hasher.Sum(nil)
		vaultInput = base64.StdEncoding.EncodeToString(hash)
	} else {
		// No pre-hashing (EdDSA)
		vaultInput = base64.StdEncoding.EncodeToString([]byte(signingInput))
	}
	data["input"] = vaultInput

	signPath := fmt.Sprintf("transit/sign/%s", v.keyName)
	signSecret, err := v.client.Logical().WriteWithContext(ctx, signPath, data)
	if err != nil {
		return "", fmt.Errorf("vault sign: %v", err)
	}

	signatureString, ok := signSecret.Data["signature"].(string)
	if !ok {
		return "", fmt.Errorf("vault response missing signature")
	}

	// Parse vault signature: "vault:v1:base64sig"
	var signatureB64 []byte
	if len(signatureString) > 8 && signatureString[:6] == "vault:" {
		parts := splitVaultSignature(signatureString)
		if len(parts) == 3 {
			// part 1 is "vault", part 2 is "v1", part 3 is signature
			// The signature is already base64 encoded, decoding it is not needed and
			// will make the code failing.
			signatureB64 = []byte(parts[2])
		}
	} else {
		return "", fmt.Errorf("unexpected signature format: %s", signatureString)
	}

	return fmt.Sprintf("%s.%s.%s", headerB64, payloadB64, signatureB64), nil
}

func (v *vaultSigner) ValidationKeys(ctx context.Context) ([]*jose.JSONWebKey, error) {
	keysMap, _, err := v.getTransitKeysMap(ctx)
	if err != nil {
		return nil, err
	}

	keys := make([]*jose.JSONWebKey, 0, len(keysMap))
	for _, k := range keysMap {
		keys = append(keys, k)
	}
	return keys, nil
}

// getTransitKeysMap returns a map of key_version -> JWK and the latest version number
func (v *vaultSigner) getTransitKeysMap(ctx context.Context) (map[int64]*jose.JSONWebKey, int64, error) {
	path := fmt.Sprintf("transit/keys/%s", v.keyName)
	secret, err := v.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read key from vault: %v", err)
	}
	if secret == nil {
		return nil, 0, fmt.Errorf("key %q not found in vault", v.keyName)
	}

	latestVersion, ok := secret.Data["latest_version"].(json.Number)
	if !ok {
		// Try float64 which is default for unmarshal interface{}
		if lv, ok := secret.Data["latest_version"].(float64); ok {
			latestVersion = json.Number(fmt.Sprintf("%d", int(lv)))
		} else if lv, ok := secret.Data["latest_version"].(int); ok {
			latestVersion = json.Number(fmt.Sprintf("%d", lv))
		}
	}
	latestVerInt, err := latestVersion.Int64()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get latest version: %v", err)
	}

	keysObj, ok := secret.Data["keys"].(map[string]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("invalid response from vault")
	}

	jwksMap := make(map[int64]*jose.JSONWebKey)

	for verStr, data := range keysObj {
		d, ok := data.(map[string]interface{})
		if !ok {
			continue
		}

		var ver int64
		fmt.Sscanf(verStr, "%d", &ver)

		pemStr, ok := d["public_key"].(string)
		if !ok {
			continue
		}

		jwk, err := parsePEMToJWK(pemStr)
		if err != nil {
			continue
		}

		jwksMap[ver] = jwk
	}

	return jwksMap, latestVerInt, nil
}

func parsePEMToJWK(pemStr string) (*jose.JSONWebKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}

	alg := ""
	switch k := pub.(type) {
	case *rsa.PublicKey:
		alg = "RS256"
	case *ecdsa.PublicKey:
		switch k.Curve {
		case elliptic.P256():
			alg = "ES256"
		case elliptic.P384():
			alg = "ES384"
		case elliptic.P521():
			alg = "ES512"
		default:
			return nil, fmt.Errorf("unsupported ECDSA curve")
		}
	case ed25519.PublicKey:
		alg = "EdDSA"
	default:
		return nil, fmt.Errorf("unsupported key type %T", pub)
	}

	jwk := &jose.JSONWebKey{
		Key:       pub,
		Algorithm: alg,
		Use:       "sig",
	}

	thumbprint, err := jwk.Thumbprint(crypto.SHA256)
	if err != nil {
		return nil, err
	}
	jwk.KeyID = base64.RawURLEncoding.EncodeToString(thumbprint)

	return jwk, nil
}

func splitVaultSignature(sig string) []string {
	// Basic split implementation
	// "vault:v1:signature"
	var parts []string
	start := 0
	for i := 0; i < len(sig); i++ {
		if sig[i] == ':' {
			parts = append(parts, sig[start:i])
			start = i + 1
		}
	}
	parts = append(parts, sig[start:])
	return parts
}

func (v *vaultSigner) Algorithm(ctx context.Context) (jose.SignatureAlgorithm, error) {
	keysMap, latestVersion, err := v.getTransitKeysMap(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get keys: %v", err)
	}

	signingJWK, ok := keysMap[latestVersion]
	if !ok {
		return "", fmt.Errorf("latest key version %d not found", latestVersion)
	}
	return jose.SignatureAlgorithm(signingJWK.Algorithm), nil
}

type vaultAlgoParams struct {
	hasher      hash.Hash
	extraParams map[string]interface{}
}

func getVaultParams(alg string) (vaultAlgoParams, error) {
	params := vaultAlgoParams{
		extraParams: map[string]interface{}{
			"marshaling_algorithm": "jws",
			"signature_algorithm":  "pkcs1v15",
		},
	}

	switch alg {
	case "RS256":
		params.hasher = sha256.New()
		params.extraParams["prehashed"] = true
		params.extraParams["hash_algorithm"] = "sha2-256"
	case "ES256":
		params.hasher = sha256.New()
		params.extraParams["prehashed"] = true
		params.extraParams["hash_algorithm"] = "sha2-256"
	case "ES384":
		params.hasher = sha512.New384()
		params.extraParams["prehashed"] = true
		params.extraParams["hash_algorithm"] = "sha2-384"
	case "ES512":
		params.hasher = sha512.New()
		params.extraParams["prehashed"] = true
		params.extraParams["hash_algorithm"] = "sha2-512"
	case "EdDSA":
		// No hashing
		params.hasher = nil
	default:
		return params, fmt.Errorf("unsupported signing algorithm: %s", alg)
	}
	return params, nil
}

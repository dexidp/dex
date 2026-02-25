package signer

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	vault "github.com/openbao/openbao/api/v2"
)

// Conformance tests verify that Vault and OpenBao behave identically with the signer.
// These tests use a single SDK (OpenBao API) that works with both systems.
//
// To run tests for a specific system, set the environment variables:
//
// For Vault:
//   DEX_VAULT_ADDR=http://localhost:8200
//   DEX_VAULT_TOKEN=root-token
//   go test -v -run TestVaultSignerConformance
//
// For OpenBao:
//   DEX_OPENBAO_ADDR=http://localhost:8210
//   DEX_OPENBAO_TOKEN=root-token
//   go test -v -run TestVaultSignerConformance
//
// To test both systems in parallel, set both sets of environment variables.

type conformanceTestConfig struct {
	name  string
	addr  string
	token string
}

// getTestConfigs returns list of test configs based on environment variables
func getTestConfigs(t *testing.T) []conformanceTestConfig {
	var configs []conformanceTestConfig

	// Check for Vault
	vaultAddr := os.Getenv("DEX_VAULT_ADDR")
	vaultToken := os.Getenv("DEX_VAULT_TOKEN")
	if vaultAddr != "" && vaultToken != "" {
		configs = append(configs, conformanceTestConfig{
			name:  "Vault",
			addr:  vaultAddr,
			token: vaultToken,
		})
	}

	// Check for OpenBao
	openbaoAddr := os.Getenv("DEX_OPENBAO_ADDR")
	openbaoToken := os.Getenv("DEX_OPENBAO_TOKEN")
	if openbaoAddr != "" && openbaoToken != "" {
		configs = append(configs, conformanceTestConfig{
			name:  "OpenBao",
			addr:  openbaoAddr,
			token: openbaoToken,
		})
	}

	if len(configs) == 0 {
		t.Skip("Skipping conformance tests. Set DEX_VAULT_TOKEN+DEX_VAULT_ADDR or DEX_OPENBAO_TOKEN+DEX_OPENBAO_ADDR to run.")
	}

	return configs
}

// TestVaultSignerConformance_SigningAndVerification tests that signing and verification work the same way
// across Vault and OpenBao implementations.
func TestVaultSignerConformance_SigningAndVerification(t *testing.T) {
	configs := getTestConfigs(t)

	testCases := []struct {
		name    string
		keyType string
		alg     string
	}{
		{
			name:    "RSA-2048",
			keyType: "rsa-2048",
			alg:     "RS256",
		},
		{
			name:    "ECDSA-P256",
			keyType: "ecdsa-p256",
			alg:     "ES256",
		},
		{
			name:    "ECDSA-P384",
			keyType: "ecdsa-p384",
			alg:     "ES384",
		},
		{
			name:    "ED25519",
			keyType: "ed25519",
			alg:     "EdDSA",
		},
	}

	for _, config := range configs {
		t.Run(config.name, func(t *testing.T) {
			ctx := context.Background()

			// Create client
			vaultConfig := vault.DefaultConfig()
			vaultConfig.Address = config.addr
			client, err := vault.NewClient(vaultConfig)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}
			client.SetToken(config.token)

			// Enable transit engine
			if err := enableTransitEngine(client); err != nil {
				t.Fatalf("failed to enable transit engine: %v", err)
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					keyName := fmt.Sprintf("test-key-%s-%s-%d", config.name, tc.keyType, time.Now().Unix())

					// Create key
					keyData := map[string]interface{}{
						"type": tc.keyType,
					}
					_, err := client.Logical().WriteWithContext(ctx, fmt.Sprintf("transit/keys/%s", keyName), keyData)
					if err != nil {
						t.Fatalf("failed to create key: %v", err)
					}

					defer cleanupTests(t, ctx, client, keyName)

					// Create signer
					signerConfig := VaultConfig{
						Addr:    config.addr,
						Token:   config.token,
						KeyName: keyName,
					}
					signer, err := newVaultSigner(signerConfig)
					if err != nil {
						t.Fatalf("failed to create signer: %v", err)
					}

					// Test 1: Verify algorithm
					alg, err := signer.Algorithm(ctx)
					if err != nil {
						t.Fatalf("failed to get algorithm: %v", err)
					}
					if string(alg) != tc.alg {
						t.Errorf("expected algorithm %s, got %s", tc.alg, alg)
					}

					// Test 2: Get validation keys
					keys, err := signer.ValidationKeys(ctx)
					if err != nil {
						t.Fatalf("failed to get validation keys: %v", err)
					}
					if len(keys) == 0 {
						t.Fatal("expected at least one validation key")
					}
					if keys[0].Algorithm != tc.alg {
						t.Errorf("expected key algorithm %s, got %s", tc.alg, keys[0].Algorithm)
					}
					if keys[0].Use != "sig" {
						t.Errorf("expected key use 'sig', got %s", keys[0].Use)
					}

					// Test 3: Sign and verify JWT
					payload := map[string]interface{}{
						"iss": "https://dex.example.com",
						"sub": "user123",
						"aud": "client-app",
						"exp": time.Now().Add(time.Hour).Unix(),
						"iat": time.Now().Unix(),
					}
					payloadBytes, err := json.Marshal(payload)
					if err != nil {
						t.Fatalf("failed to marshal payload: %v", err)
					}

					jwtString, err := signer.Sign(ctx, payloadBytes)
					if err != nil {
						t.Fatalf("failed to sign payload: %v", err)
					}

					// Verify JWT signature
					jws, err := jose.ParseSigned(jwtString, []jose.SignatureAlgorithm{jose.SignatureAlgorithm(tc.alg)})
					if err != nil {
						t.Fatalf("failed to parse signed JWT: %v", err)
					}

					verifiedPayload, err := jws.Verify(keys[0])
					if err != nil {
						t.Fatalf("failed to verify JWT signature: %v", err)
					}

					var decodedPayload map[string]interface{}
					if err := json.Unmarshal(verifiedPayload, &decodedPayload); err != nil {
						t.Fatalf("failed to unmarshal verified payload: %v", err)
					}

					if decodedPayload["sub"] != payload["sub"] {
						t.Errorf("payload mismatch: expected sub=%s, got %s", payload["sub"], decodedPayload["sub"])
					}

					// Test 4: Multiple signatures with same key
					for i := 0; i < 3; i++ {
						randomPayload := make([]byte, 32)
						_, err := rand.Read(randomPayload)
						if err != nil {
							t.Fatalf("failed to generate random payload: %v", err)
						}

						payloadData := map[string]interface{}{
							"data": base64.StdEncoding.EncodeToString(randomPayload),
							"iat":  time.Now().Unix(),
						}
						payloadBytes, err := json.Marshal(payloadData)
						if err != nil {
							t.Fatalf("failed to marshal payload: %v", err)
						}

						jwtString, err := signer.Sign(ctx, payloadBytes)
						if err != nil {
							t.Fatalf("sign attempt %d failed: %v", i+1, err)
						}

						jws, err := jose.ParseSigned(jwtString, []jose.SignatureAlgorithm{jose.SignatureAlgorithm(tc.alg)})
						if err != nil {
							t.Fatalf("parse attempt %d failed: %v", i+1, err)
						}

						_, err = jws.Verify(keys[0])
						if err != nil {
							t.Fatalf("verify attempt %d failed: %v", i+1, err)
						}
					}
				})
			}
		})
	}
}

// TestVaultSignerConformance_KeyRotation tests that key rotation works identically
// across Vault and OpenBao implementations.
func TestVaultSignerConformance_KeyRotation(t *testing.T) {
	configs := getTestConfigs(t)

	for _, config := range configs {
		t.Run(config.name, func(t *testing.T) {
			ctx := context.Background()

			// Create client
			vaultConfig := vault.DefaultConfig()
			vaultConfig.Address = config.addr
			client, err := vault.NewClient(vaultConfig)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}
			client.SetToken(config.token)

			// Enable transit engine
			if err := enableTransitEngine(client); err != nil {
				t.Fatalf("failed to enable transit engine: %v", err)
			}

			keyName := fmt.Sprintf("test-rotation-key-%s-%d", config.name, time.Now().Unix())

			// Create initial key
			keyData := map[string]interface{}{
				"type": "ecdsa-p256",
			}
			_, err = client.Logical().WriteWithContext(ctx, fmt.Sprintf("transit/keys/%s", keyName), keyData)
			if err != nil {
				t.Fatalf("failed to create key: %v", err)
			}

			defer cleanupTests(t, ctx, client, keyName)

			// Create signer
			signerConfig := VaultConfig{
				Addr:    config.addr,
				Token:   config.token,
				KeyName: keyName,
			}
			signer, err := newVaultSigner(signerConfig)
			if err != nil {
				t.Fatalf("failed to create signer: %v", err)
			}

			// Sign with initial key version
			payload1 := map[string]interface{}{"version": "v1", "iat": time.Now().Unix()}
			payload1Bytes, err := json.Marshal(payload1)
			if err != nil {
				t.Fatalf("failed to marshal payload: %v", err)
			}

			jwt1, err := signer.Sign(ctx, payload1Bytes)
			if err != nil {
				t.Fatalf("failed to sign with v1: %v", err)
			}

			// Get keys before rotation
			keysBefore, err := signer.ValidationKeys(ctx)
			if err != nil {
				t.Fatalf("failed to get keys before rotation: %v", err)
			}
			if len(keysBefore) != 1 {
				t.Errorf("expected 1 key before rotation, got %d", len(keysBefore))
			}

			// Rotate key
			_, err = client.Logical().WriteWithContext(ctx, fmt.Sprintf("transit/keys/%s/rotate", keyName), nil)
			if err != nil {
				t.Fatalf("failed to rotate key: %v", err)
			}

			// Sign with new key version
			payload2 := map[string]interface{}{"version": "v2", "iat": time.Now().Unix()}
			payload2Bytes, err := json.Marshal(payload2)
			if err != nil {
				t.Fatalf("failed to marshal payload: %v", err)
			}

			jwt2, err := signer.Sign(ctx, payload2Bytes)
			if err != nil {
				t.Fatalf("failed to sign with v2: %v", err)
			}

			// Get keys after rotation
			keysAfter, err := signer.ValidationKeys(ctx)
			if err != nil {
				t.Fatalf("failed to get keys after rotation: %v", err)
			}
			if len(keysAfter) != 2 {
				t.Errorf("expected 2 keys after rotation, got %d", len(keysAfter))
			}

			// Verify both JWTs can be validated with the current keyset
			jws1, err := jose.ParseSigned(jwt1, []jose.SignatureAlgorithm{jose.ES256})
			if err != nil {
				t.Fatalf("failed to parse jwt1: %v", err)
			}

			jws2, err := jose.ParseSigned(jwt2, []jose.SignatureAlgorithm{jose.ES256})
			if err != nil {
				t.Fatalf("failed to parse jwt2: %v", err)
			}

			// Find matching keys and verify
			verified1 := false
			verified2 := false

			for _, key := range keysAfter {
				if _, err := jws1.Verify(key); err == nil {
					verified1 = true
				}
				if _, err := jws2.Verify(key); err == nil {
					verified2 = true
				}
			}

			if !verified1 {
				t.Error("failed to verify JWT signed with version 1")
			}
			if !verified2 {
				t.Error("failed to verify JWT signed with version 2")
			}
		})
	}
}

// TestVaultSignerConformance_PublicKeyDiscovery tests that public key discovery works identically
// across Vault and OpenBao implementations.
func TestVaultSignerConformance_PublicKeyDiscovery(t *testing.T) {
	configs := getTestConfigs(t)

	for _, config := range configs {
		t.Run(config.name, func(t *testing.T) {
			ctx := context.Background()

			// Create client
			vaultConfig := vault.DefaultConfig()
			vaultConfig.Address = config.addr
			client, err := vault.NewClient(vaultConfig)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}
			client.SetToken(config.token)

			// Enable transit engine
			if err := enableTransitEngine(client); err != nil {
				t.Fatalf("failed to enable transit engine: %v", err)
			}

			keyName := fmt.Sprintf("test-discovery-key-%s-%d", config.name, time.Now().Unix())

			// Create key
			keyData := map[string]interface{}{
				"type": "rsa-2048",
			}
			_, err = client.Logical().WriteWithContext(ctx, fmt.Sprintf("transit/keys/%s", keyName), keyData)
			if err != nil {
				t.Fatalf("failed to create key: %v", err)
			}

			defer cleanupTests(t, ctx, client, keyName)

			// Create signer
			signerConfig := VaultConfig{
				Addr:    config.addr,
				Token:   config.token,
				KeyName: keyName,
			}
			signer, err := newVaultSigner(signerConfig)
			if err != nil {
				t.Fatalf("failed to create signer: %v", err)
			}

			// Get public keys (simulating JWKS endpoint)
			keys, err := signer.ValidationKeys(ctx)
			if err != nil {
				t.Fatalf("failed to get validation keys: %v", err)
			}

			// Verify keys have required JWKS fields
			for i, key := range keys {
				if key.KeyID == "" {
					t.Errorf("key %d missing KeyID", i)
				}
				if key.Algorithm == "" {
					t.Errorf("key %d missing Algorithm", i)
				}
				if key.Use != "sig" {
					t.Errorf("key %d has wrong Use field: expected 'sig', got '%s'", i, key.Use)
				}
				if key.Key == nil {
					t.Errorf("key %d missing public key", i)
				}

				// Verify key can be marshaled to JWKS format
				jwksData, err := json.Marshal(key)
				if err != nil {
					t.Errorf("key %d cannot be marshaled to JSON: %v", i, err)
				}

				var jwksCheck map[string]interface{}
				if err := json.Unmarshal(jwksData, &jwksCheck); err != nil {
					t.Errorf("key %d JWKS data is invalid: %v", i, err)
				}

				// Check for standard JWKS fields
				requiredFields := []string{"kty", "use", "kid", "alg"}
				for _, field := range requiredFields {
					if _, ok := jwksCheck[field]; !ok {
						t.Errorf("key %d missing required JWKS field: %s", i, field)
					}
				}
			}

			// Sign a JWT
			payload := map[string]interface{}{
				"iss": "https://dex.example.com",
				"sub": "test-user",
				"aud": "test-client",
				"exp": time.Now().Add(time.Hour).Unix(),
				"iat": time.Now().Unix(),
			}
			payloadBytes, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("failed to marshal payload: %v", err)
			}

			jwtString, err := signer.Sign(ctx, payloadBytes)
			if err != nil {
				t.Fatalf("failed to sign JWT: %v", err)
			}

			// Parse JWT and verify it has correct kid in header
			jws, err := jose.ParseSigned(jwtString, []jose.SignatureAlgorithm{jose.RS256})
			if err != nil {
				t.Fatalf("failed to parse JWT: %v", err)
			}

			if len(jws.Signatures) == 0 {
				t.Fatal("JWT has no signatures")
			}

			kid := jws.Signatures[0].Header.KeyID
			if kid == "" {
				t.Error("JWT header missing kid")
			}

			// Verify kid matches one of the public keys
			kidFound := false
			for _, key := range keys {
				if key.KeyID == kid {
					kidFound = true
					break
				}
			}
			if !kidFound {
				t.Errorf("JWT kid '%s' not found in public keys", kid)
			}
		})
	}
}

// enableTransitEngine enables the transit secrets engine if not already enabled.
func enableTransitEngine(client *vault.Client) error {
	// Check if already enabled
	mounts, err := client.Sys().ListMounts()
	if err != nil {
		return fmt.Errorf("failed to list mounts: %v", err)
	}

	if _, exists := mounts["transit/"]; exists {
		return nil
	}

	// Enable transit engine
	mountInput := &vault.MountInput{
		Type: "transit",
	}
	if err := client.Sys().Mount("transit", mountInput); err != nil {
		return fmt.Errorf("failed to mount transit: %v", err)
	}

	return nil
}

func cleanupTests(t *testing.T, ctx context.Context, client *vault.Client, keyName string) {
	updateData := map[string]interface{}{
		"deletion_allowed": true,
	}
	_, err := client.Logical().WriteWithContext(ctx, fmt.Sprintf("transit/keys/%s/config", keyName), updateData)
	if err != nil {
		t.Logf("failed to update key config: %v", err)
	}
	_, err = client.Logical().DeleteWithContext(ctx, fmt.Sprintf("transit/keys/%s", keyName))
	if err != nil {
		t.Logf("failed to delete key: %v", err)
	}
}

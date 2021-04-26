package conformance

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/coreos/go-oidc"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/dexidp/dex/signer"
)

type subTest struct {
	name string
	run  func(t *testing.T, s signer.Signer)
}

func runTests(t *testing.T, newSigner func() signer.Signer, tests []subTest) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := newSigner()
			test.run(t, s)
		})
	}
}

// RunTests runs a set of conformance tests against a storage. newStorage should
// return an initialized but empty storage. The storage will be closed at the
// end of each test run.
func RunTests(t *testing.T, newSigner func() signer.Signer) {
	runTests(t, newSigner, []subTest{
		{"Keys", testKeys},
		{"Hasher", testHasher},
		{"Sign", testSign},
	})
}

func testKeys(t *testing.T, s signer.Signer) {
	keys, err := s.GetSigningKeys()
	if err != nil {
		t.Fatalf("Fail to get signing keys: %s", err.Error())
	}

	if len(keys.Jwks.Keys) == 0 {
		t.Errorf("no keys available")
	}
	if keys.NextRotation != nil && keys.NextRotation.Before(time.Now().Add(-10*time.Minute)) {
		t.Errorf("Invalid next key rotation value: %s", keys.NextRotation.String())
	}
}

func testHasher(t *testing.T, s signer.Signer) {
	testValue := []byte("test")
	keys, err := s.GetSigningKeys()
	if err != nil {
		t.Fatalf("Fail to get signing keys: %s", err.Error())
	}
	hasher, err := s.Hasher()
	if err != nil {
		t.Fatalf("Fail to get signing hasher: %s", err.Error())
	}
	hasher.Write(testValue)
	expected := hasher.Sum(nil)

	for _, key := range keys.Jwks.Keys {
		// there should be a signer for any key algorithm that matches the value
		keyHasher, err := signer.HashForSigAlgorithm(jose.SignatureAlgorithm(key.Algorithm))
		if err != nil {
			t.Fatalf("Key %s: fail to get signature hasher: %s", key.KeyID, err.Error())
		}
		keyHasher.Write(testValue)
		out := keyHasher.Sum(nil)
		if bytes.Equal(out, expected) {
			return
		}
	}

	t.Error("Mismatched hash algorithm")
}

func testSign(t *testing.T, s signer.Signer) {
	issuer := "http://www.example.com"
	clientID := "client-id"
	now := time.Now()

	payload := jwt.Claims{
		Issuer:   issuer,
		Audience: jwt.Audience{clientID},
		Expiry:   jwt.NewNumericDate(now.Add(1 * time.Minute)),
		IssuedAt: jwt.NewNumericDate(now),
	}
	payloadJSON, _ := json.Marshal(payload)

	jws, err := s.Sign(payloadJSON)
	if err != nil {
		t.Fatal(err.Error())
	}

	keySet, err := s.GetKeySet()
	if err != nil {
		t.Fatal(err.Error())
	}

	verifier := oidc.NewVerifier(issuer, keySet, &oidc.Config{
		ClientID:             clientID,
		Now:                  func() time.Time { return now },
		SupportedSigningAlgs: []string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512", "EdDSA"},
	})
	_, err = verifier.Verify(context.Background(), jws)
	if err != nil {
		t.Fatal(err.Error())
	}
}

package secret

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testHarness struct {
	privateKeyFile *os.File
}

func createKeyFile(_ *testing.T) *os.File {
	privkey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	bytes, _ := x509.MarshalPKCS8PrivateKey(privkey)
	block := pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: bytes,
	}
	raw := pem.EncodeToMemory(&block)
	tmpfile, _ := os.CreateTemp("", "privatekey")
	tmpfile.Write(raw)
	return tmpfile
}

func newHarness(t *testing.T) *testHarness {
	return &testHarness{
		privateKeyFile: createKeyFile(t),
	}
}

func (th *testHarness) CleanUp() {
	os.Remove(th.privateKeyFile.Name())
}

func TestBasicSecret(t *testing.T) {
	th := newHarness(t)
	defer th.CleanUp()

	config := &Config{
		TeamID:         "teamfoo",
		KeyID:          "somekey",
		PrivateKeyFile: th.privateKeyFile.Name(),
	}
	secret, err := NewSecret(config)
	assert.Nil(t, err)
	assert.NotNil(t, secret)
	assert.True(t, secret.IsExpired())
	secretStr, err := secret.GetSecret()
	assert.Nil(t, err)
	assert.NotNil(t, secretStr)
}

func TestExpiryRegeneratesSecret(t *testing.T) {
	th := newHarness(t)
	defer th.CleanUp()

	config := &Config{
		TeamID:          "teamfoo",
		KeyID:           "somekey",
		PrivateKeyFile:  th.privateKeyFile.Name(),
		SecretDuration:  1,
		SecretExpiryMin: 10,
	}

	// Duration is 1 second, while expiry min is 10, so it will
	// create a new secret on every invocation
	secret, _ := NewSecret(config)
	secretStr, err := secret.GetSecret()
	assert.Nil(t, err)
	assert.NotNil(t, secretStr)
	assert.True(t, secret.IsExpired())
	secretStr2, err := secret.GetSecret()
	assert.Nil(t, err)
	assert.NotNil(t, secretStr2)
	assert.NotEqual(t, secretStr, secretStr2)
}

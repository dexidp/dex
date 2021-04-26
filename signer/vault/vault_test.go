package vault

import (
	"fmt"
	"os"
	"testing"

	vault "github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"

	"github.com/dexidp/dex/signer"
	"github.com/dexidp/dex/signer/conformance"
)

var logger = &logrus.Logger{
	Out:       os.Stderr,
	Formatter: &logrus.TextFormatter{DisableColors: true},
	Level:     logrus.DebugLevel,
}

func TestSigner(t *testing.T) {
	if os.Getenv("TEST_VAULT_ADDR") == "" {
		t.Skip("vault is not configured")
	}

	for _, keyType := range []string{"rsa-2048", "rsa-3072", "rsa-4096", "ecdsa-p256", "ecdsa-p384", "ecdsa-p521", "ed25519"} {
		t.Run(keyType, func(t *testing.T) {
			testConformance(t, keyType)
		})
	}
}

func testConformance(t *testing.T, keyType string) {
	client, err := vault.NewClient(&vault.Config{
		AgentAddress: os.Getenv("TEST_VAULT_ADDR"),
	})
	if err != nil {
		t.Fatal(err.Error())
	}

	client.Sys().Mount("/transit/", &vault.MountInput{
		Type: "transit",
	})
	client.Logical().Write(fmt.Sprintf("/transit/keys/%s", keyType), map[string]interface{}{
		"type": keyType,
	})

	s := Signer{
		vault: client,
		config: Config{
			Address:      client.Address(),
			TransitMount: "/transit/",
			KeyName:      keyType,
		},
		logger: logger,
	}
	if err = s.init(); err != nil {
		t.Fatal(err.Error())
	}

	conformance.RunTests(t, func() signer.Signer {
		return &s
	})
}

package vault

import (
	"time"

	vault "github.com/hashicorp/vault/api"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/signer"
)

type Config struct {
	Address string `json:"address" yaml:"address"`
	// TransitMount is the path to transit storage engine
	TransitMount string `json:"mount" yaml:"mount"`
	KeyName      string `json:"key" yaml:"key"`
}

func (c Config) Open(logger log.Logger, tokenValid time.Duration, rotationPeriod time.Duration) (signer.Signer, error) {
	client, err := vault.NewClient(&vault.Config{
		AgentAddress: c.Address,
	})
	if err != nil {
		return nil, err
	}
	out := &Signer{
		vault:  client,
		config: c,
		logger: logger,
	}
	if err = out.init(); err != nil {
		return nil, err
	}

	return out, nil
}

package storage

import (
	"time"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/signer"
	"github.com/dexidp/dex/storage"
)

type Config struct {
	Storage          storage.KeyStorage
	RotationStrategy RotationStrategy
	Now              func() time.Time
}

func (c *Config) Open(logger log.Logger, tokenValid time.Duration, rotationPeriod time.Duration) (signer.Signer, error) {
	now := c.Now
	if now == nil {
		now = time.Now
	}

	strategy := c.RotationStrategy
	if strategy.key == nil {
		strategy = DefaultRotationStrategy(rotationPeriod, tokenValid)
	}

	return &Signer{
		storage:          newKeyCacher(c.Storage, now),
		logger:           logger,
		now:              now,
		rotationStrategy: strategy,
	}, nil
}

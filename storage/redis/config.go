package redis

import (
	redisv8 "github.com/go-redis/redis/v8"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/storage"
)

type Config struct {
	Addrs            []string `json:"addrs" yaml:"addrs"`
	Password         string   `json:"password" yaml:"password"`
	SentinelPassword string   `json:"sentinel_password" yaml:"sentinel_password"`
	MasterName       string   `json:"master_name" yaml:"master_name"`
}

func (c *Config) Open(logger log.Logger) (storage.Storage, error) {
	return c.open(logger), nil
}

func (c *Config) open(logger log.Logger) *client {
	opts := &redisv8.UniversalOptions{
		Addrs:            c.Addrs,
		Password:         c.Password,
		SentinelPassword: c.SentinelPassword,
		MasterName:       c.MasterName,
	}
	return &client{
		db:     redisv8.NewUniversalClient(opts),
		logger: logger,
	}
}

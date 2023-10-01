package redis

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/conformance"
)

func cleanDB(c *client) error {
	ctx := context.TODO()
	for _, prefix := range []string{
		clientPrefix,
		authCodePrefix,
		refreshTokenPrefix,
		authRequestPrefix,
		passwordPrefix,
		offlineSessionPrefix,
		connectorPrefix,
		deviceRequestPrefix,
		deviceTokenPrefix,
	} {
		keys, err := c.db.Keys(ctx, prefix+"*").Result()
		if err != nil {
			return err
		}
		if len(keys) == 0 {
			continue
		}
		err = c.db.Del(ctx, keys...).Err()
		if err != nil {
			return err
		}
	}
	return nil
}

func TestRedis(t *testing.T) {
	testRedisEnv := "DEX_REDIS_ADDR"
	addr := os.Getenv(testRedisEnv)
	if addr == "" {
		t.Skipf("test environment variable %q not set, skipping", testRedisEnv)
		return
	}

	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: &logrus.TextFormatter{DisableColors: true},
		Level:     logrus.DebugLevel,
	}

	newStorage := func() storage.Storage {
		config := &Config{
			Addrs: []string{addr},
		}
		client := config.open(logger)
		if err := cleanDB(client); err != nil {
			fmt.Fprintln(os.Stdout, err)
			t.Fatal(err)
		}
		return client
	}

	conformance.RunTests(t, newStorage)
}

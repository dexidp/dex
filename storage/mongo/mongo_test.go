package mongo

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/conformance"
	"github.com/sirupsen/logrus"
)

func withTimeout(t time.Duration, f func()) {
	c := make(chan struct{})
	defer close(c)

	go func() {
		select {
		case <-c:
		case <-time.After(t):
			// Dump a stack trace of the program. Useful for debugging deadlocks.
			buf := make([]byte, 2<<20)
			fmt.Fprintf(os.Stderr, "%s\n", buf[:runtime.Stack(buf, true)])
			panic("test took too long")
		}
	}()

	f()
}

func cleanDB(c *mongoStorage) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	c.authCodeCollection().DeleteMany(ctx, emptyFilter())
	c.authReqCollection().DeleteMany(ctx, emptyFilter())
	c.clientCollection().DeleteMany(ctx, emptyFilter())
	c.refreshTokenCollection().DeleteMany(ctx, emptyFilter())
	c.passwordCollection().DeleteMany(ctx, emptyFilter())
	c.offlineSessionCollection().DeleteMany(ctx, emptyFilter())
	c.connectorCollection().DeleteMany(ctx, emptyFilter())
	c.keysCollection().DeleteMany(ctx, emptyFilter())
	c.deviceRequestCollection().DeleteMany(ctx, emptyFilter())
	c.deviceTokenCollection().DeleteMany(ctx, emptyFilter())

	return nil
}

var logger = &logrus.Logger{
	Out:       os.Stderr,
	Formatter: &logrus.TextFormatter{DisableColors: true},
	Level:     logrus.DebugLevel,
}

const testMongoEnv = "DEX_MONGO_URI"

func TestMongo(t *testing.T) {
	uri := os.Getenv(testMongoEnv)
	if uri == "" {
		t.Skipf("test environment variable %q not set, skipping", testMongoEnv)
	}

	newStorage := func() storage.Storage {
		s := &Mongo{
			URI:                   uri,
			Database:              "oidc",
			ConnectionTimeout:     time.Second * 200,
			DatabaseTimeout:       time.Second * 200,
			UseGCInsteadOfIndexes: true,
		}
		conn, err := s.open(logger)
		if err != nil {
			fmt.Fprintln(os.Stdout, err)
			t.Fatal(err)
		}

		if err := cleanDB(conn); err != nil {
			fmt.Fprintln(os.Stdout, err)
			t.Fatal(err)
		}
		return conn
	}

	withTimeout(time.Second*10, func() {
		conformance.RunTests(t, newStorage)
	})

	withTimeout(time.Minute*1, func() {
		conformance.RunTransactionTests(t, newStorage)
	})
}

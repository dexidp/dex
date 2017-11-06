package etcd

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/coreos/dex/storage"
	"github.com/coreos/dex/storage/conformance"
	"github.com/coreos/etcd/clientv3"
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

func cleanDB(c *conn) error {
	ctx := context.TODO()
	for _, prefix := range []string{
		clientPrefix,
		authCodePrefix,
		refreshTokenPrefix,
		authRequestPrefix,
		passwordPrefix,
		offlineSessionPrefix,
		connectorPrefix,
	} {
		_, err := c.db.Delete(ctx, prefix, clientv3.WithPrefix())
		if err != nil {
			return err
		}
	}
	return nil
}

var logger = &logrus.Logger{
	Out:       os.Stderr,
	Formatter: &logrus.TextFormatter{DisableColors: true},
	Level:     logrus.DebugLevel,
}

func TestEtcd(t *testing.T) {
	testEtcdEnv := "DEX_ETCD_ENDPOINTS"
	endpointsStr := os.Getenv(testEtcdEnv)
	if endpointsStr == "" {
		t.Skipf("test environment variable %q not set, skipping", testEtcdEnv)
		return
	}
	endpoints := strings.Split(endpointsStr, ",")

	newStorage := func() storage.Storage {
		s := &Etcd{
			Endpoints: endpoints,
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

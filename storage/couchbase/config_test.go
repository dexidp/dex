package couchbase

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/conformance"
	"github.com/sirupsen/logrus"
)

var logger = &logrus.Logger{
	Out:       os.Stderr,
	Formatter: &logrus.TextFormatter{DisableColors: true},
	Level:     logrus.DebugLevel,
}

func testDB(t *testing.T, o opener, withTransactions bool) {
	// t.Fatal has a bad habbit of not actually printing the error
	fatal := func(i interface{}) {
		fmt.Fprintln(os.Stdout, i)
		t.Fatal(i)
	}

	newStorage := func() storage.Storage {
		conn, err := o.open(logger)
		if err != nil {
			fatal(err)
		}
		return conn
	}
	withTimeout(time.Minute*1, func() {
		conformance.RunTests(t, newStorage)
	})
	if withTransactions {
		withTimeout(time.Minute*1, func() {
			conformance.RunTransactionTests(t, newStorage)
		})
	}
}

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

type opener interface {
	open(logger log.Logger) (*conn, error)
}

const testCbEnv = "DEX_COUCHBASE_HOST"

func TestCouchbase(t *testing.T) {
	host := os.Getenv(testCbEnv)
	if host == "" {
		t.Skipf("test environment variable %q not set, skipping", testCbEnv)
	}
	p := &Couchbase{
		NetworkDB: NetworkDB{
			Bucket:   getenv("DEX_COUCHBASE_DATABASE", "couchbase"),
			User:     getenv("DEX_COUCHBASE_USER", "couchbase"),
			Password: getenv("DEX_COUCHBASE_PASSWORD", "couchbase"),
			Host:     host,
		},
		SSL: SSL{
			CertFile: "/tmp/cert.pem", // Couchbase container doesn't support SSL.
		},
	}
	testDB(t, p, true)
}

func getenv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

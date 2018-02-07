package mongodb

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/coreos/dex/storage"
	"github.com/coreos/dex/storage/conformance"
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

func cleanDB(c *mgoStorage) error {
	for _, col := range []string{
		ColAuthRequest,
		ColClient,
		ColAuthCode,
		ColRefreshToken,
		ColPassword,
		ColOfflineSessions,
		ColConnector,
		ColSetting,
	} {
		err := c.db.C(col).DropCollection()
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

func TestMongoDB(t *testing.T) {
	testMgoURLEnv := "DEX_MGO_URL"
	mgoURL := os.Getenv(testMgoURLEnv)
	if mgoURL == "" {
		t.Skipf("test environment variable %q not set, skipping", testMgoURLEnv)
		return
	}

	newStorage := func() storage.Storage {
		ms, err := newMgoStorage(logger, mgoURL)
		if err != nil {
			fmt.Fprintln(os.Stdout, err)
			t.Fatal(err)
		}
		return ms
	}

	withTimeout(time.Second*10, func() {
		conformance.RunTests(t, newStorage)
	})

	withTimeout(time.Minute*1, func() {
		conformance.RunTransactionTests(t, newStorage)
	})
}

func TestGetMgoName(t *testing.T) {
	table := map[string]string{
		"mongodb://example.com/testdex":           "testdex",
		"mongodb://example.com:27018/testdex":     "testdex",
		"example.com/testdex":                     "testdex",
		"mongodb://example.com/testdex?a=x":       "testdex",
		"mongodb://example.com:27018/testdex?a=x": "testdex",
		"example.com/testdex?a=x":                 "testdex",
		"mongodb://example.com":                   defaultDBName,
		"mongodb://example.com:27018":             defaultDBName,
		"example.com":                             defaultDBName,
		"mongodb://example.com/":                  defaultDBName,
		"mongodb://example.com:27018/":            defaultDBName,
		"example.com/":                            defaultDBName,
	}
	for k, v := range table {
		if cur := getDbNameFromURL(k); cur != v {
			t.Errorf("test getDbNameFromURL failed, (%s)%s - %s", k, cur, v)
		}
	}
}

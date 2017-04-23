package sql

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/dex/storage"
	"github.com/coreos/dex/storage/conformance"
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
	tables := []string{"client", "auth_request", "auth_code",
		"refresh_token", "keys", "password"}

	for _, tbl := range tables {
		_, err := c.Exec("delete from " + tbl)
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

type opener interface {
	open(logrus.FieldLogger) (*conn, error)
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
		if err := cleanDB(conn); err != nil {
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

func TestSQLite3(t *testing.T) {
	testDB(t, &SQLite3{":memory:"}, false)
}

func getenv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

const testPostgresEnv = "DEX_POSTGRES_HOST"

func TestPostgres(t *testing.T) {
	host := os.Getenv(testPostgresEnv)
	if host == "" {
		t.Skipf("test environment variable %q not set, skipping", testPostgresEnv)
	}
	p := &Postgres{
		NetworkDB: NetworkDB{
			Database:          getenv("DEX_POSTGRES_DATABASE", "postgres"),
			User:              getenv("DEX_POSTGRES_USER", "postgres"),
			Password:          getenv("DEX_POSTGRES_PASSWORD", "postgres"),
			Host:              host,
			ConnectionTimeout: 5,
		},
		SSL: SSL{
			Mode: pgSSLDisable, // Postgres container doesn't support SSL.
		},
	}
	testDB(t, p, true)
}

const testMySQLEnv = "DEX_MYSQL_HOST"

func TestMySQL(t *testing.T) {
	host := os.Getenv(testMySQLEnv)
	if host == "" {
		t.Skipf("test environment variable %q not set, skipping", testMySQLEnv)
	}
	s := &MySQL{
		NetworkDB: NetworkDB{
			Database:          getenv("DEX_MYSQL_DATABASE", "mysql"),
			User:              getenv("DEX_MYSQL_USER", "mysql"),
			Password:          getenv("DEX_MYSQL_PASSWORD", ""),
			Host:              host,
			ConnectionTimeout: 5,
		},
		SSL: SSL{
			Mode: mysqlSSLFalse,
		},
		params: map[string]string{
			"innodb_lock_wait_timeout": "3",
		},
	}
	testDB(t, s, true)
}

package sql

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/conformance"
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
	_, err := c.Exec(`
		delete from client;
		delete from auth_request;
		delete from auth_code;
		delete from refresh_token;
		delete from keys;
		delete from password;
	`)
	return err
}

var logger = &logrus.Logger{
	Out:       os.Stderr,
	Formatter: &logrus.TextFormatter{DisableColors: true},
	Level:     logrus.DebugLevel,
}

func TestSQLite3(t *testing.T) {
	newStorage := func() storage.Storage {
		// NOTE(ericchiang): In memory means we only get one connection at a time. If we
		// ever write tests that require using multiple connections, for instance to test
		// transactions, we need to move to a file based system.
		s := &SQLite3{":memory:"}
		conn, err := s.open(logger)
		if err != nil {
			fmt.Fprintln(os.Stdout, err)
			t.Fatal(err)
		}
		return conn
	}

	withTimeout(time.Second*10, func() {
		conformance.RunTests(t, newStorage)
	})
}

func getenv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

const testPostgresEnv = "DEX_POSTGRES_HOST"

func TestCreateDataSourceName(t *testing.T) {
	var testCases = []struct {
		description string
		input       *Postgres
		expected    string
	}{
		{
			description: "with no configuration",
			input:       &Postgres{},
			expected:    "connect_timeout=0 sslmode='verify-full'",
		},
		{
			description: "with typical configuration",
			input: &Postgres{
				Host:     "1.2.3.4",
				Port:     6543,
				User:     "some-user",
				Password: "some-password",
				Database: "some-db",
			},
			expected: "connect_timeout=0 host='1.2.3.4' port=6543 user='some-user' password='some-password' dbname='some-db' sslmode='verify-full'",
		},
		{
			description: "with unix socket host",
			input: &Postgres{
				Host: "/var/run/postgres",
				SSL: PostgresSSL{
					Mode: "disable",
				},
			},
			expected: "connect_timeout=0 host='/var/run/postgres' sslmode='disable'",
		},
		{
			description: "with tcp host",
			input: &Postgres{
				Host: "coreos.com",
				SSL: PostgresSSL{
					Mode: "disable",
				},
			},
			expected: "connect_timeout=0 host='coreos.com' sslmode='disable'",
		},
		{
			description: "with tcp host:port",
			input: &Postgres{
				Host: "coreos.com:6543",
			},
			expected: "connect_timeout=0 host='coreos.com' port=6543 sslmode='verify-full'",
		},
		{
			description: "with tcp host and port",
			input: &Postgres{
				Host: "coreos.com",
				Port: 6543,
			},
			expected: "connect_timeout=0 host='coreos.com' port=6543 sslmode='verify-full'",
		},
		{
			description: "with ssl ca cert",
			input: &Postgres{
				Host: "coreos.com",
				SSL: PostgresSSL{
					Mode:   "verify-ca",
					CAFile: "/some/file/path",
				},
			},
			expected: "connect_timeout=0 host='coreos.com' sslmode='verify-ca' sslrootcert='/some/file/path'",
		},
		{
			description: "with ssl client cert",
			input: &Postgres{
				Host: "coreos.com",
				SSL: PostgresSSL{
					Mode:     "verify-ca",
					CAFile:   "/some/ca/path",
					CertFile: "/some/cert/path",
					KeyFile:  "/some/key/path",
				},
			},
			expected: "connect_timeout=0 host='coreos.com' sslmode='verify-ca' sslrootcert='/some/ca/path' sslcert='/some/cert/path' sslkey='/some/key/path'",
		},
		{
			description: "with funny characters in credentials",
			input: &Postgres{
				Host:     "coreos.com",
				User:     `some'user\slashed`,
				Password: "some'password!",
			},
			expected: `connect_timeout=0 host='coreos.com' user='some\'user\\slashed' password='some\'password!' sslmode='verify-full'`,
		},
	}

	var actual string
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			actual = testCase.input.createDataSourceName()

			if actual != testCase.expected {
				t.Fatalf("%s != %s", actual, testCase.expected)
			}
		})
	}
}

func TestPostgres(t *testing.T) {
	host := os.Getenv(testPostgresEnv)
	if host == "" {
		t.Skipf("test environment variable %q not set, skipping", testPostgresEnv)
	}
	p := Postgres{
		Database: getenv("DEX_POSTGRES_DATABASE", "postgres"),
		User:     getenv("DEX_POSTGRES_USER", "postgres"),
		Password: getenv("DEX_POSTGRES_PASSWORD", "postgres"),
		Host:     host,
		SSL: PostgresSSL{
			Mode: sslDisable, // Postgres container doesn't support SSL.
		},
		ConnectionTimeout: 5,
	}

	// t.Fatal has a bad habbit of not actually printing the error
	fatal := func(i interface{}) {
		fmt.Fprintln(os.Stdout, i)
		t.Fatal(i)
	}

	newStorage := func() storage.Storage {
		conn, err := p.open(logger, p.createDataSourceName())
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
	withTimeout(time.Minute*1, func() {
		conformance.RunTransactionTests(t, newStorage)
	})
}

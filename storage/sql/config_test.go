package sql

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

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
	_, err := c.Exec(`
		delete from client;
		delete from auth_request;
		delete from auth_code;
		delete from refresh_token;
		delete from keys;
	`)
	return err
}

func TestSQLite3(t *testing.T) {
	newStorage := func() storage.Storage {
		// NOTE(ericchiang): In memory means we only get one connection at a time. If we
		// ever write tests that require using multiple connections, for instance to test
		// transactions, we need to move to a file based system.
		s := &SQLite3{":memory:"}
		conn, err := s.open()
		if err != nil {
			t.Fatal(err)
		}
		return conn
	}

	withTimeout(time.Second*10, func() {
		conformance.RunTests(t, newStorage)
	})
}

func TestPostgres(t *testing.T) {
	if os.Getenv("DEX_POSTGRES_HOST") == "" {
		t.Skip("postgres envs not set, skipping tests")
	}
	p := Postgres{
		Database: os.Getenv("DEX_POSTGRES_DATABASE"),
		User:     os.Getenv("DEX_POSTGRES_USER"),
		Password: os.Getenv("DEX_POSTGRES_PASSWORD"),
		Host:     os.Getenv("DEX_POSTGRES_HOST"),
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
		conn, err := p.open()
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
}

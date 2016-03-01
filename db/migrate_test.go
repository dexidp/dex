package db

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"testing"

	"github.com/go-gorp/gorp"
	"github.com/kylelemons/godebug/pretty"
)

func initDB(dsn string) *gorp.DbMap {
	c, err := NewConnection(Config{DSN: dsn})
	if err != nil {
		panic(fmt.Sprintf("error making db connection: %q", err))
	}
	if _, err := c.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", migrationTable)); err != nil {
		panic(fmt.Sprintf("failed to drop migration table: %v", err))
	}
	if err = c.DropTablesIfExists(); err != nil {
		panic(fmt.Sprintf("Unable to drop database tables: %v", err))
	}

	return c
}

// TestGetPlannedMigrations is a sanity check, ensuring that at least one
// migration can be found.
func TestGetPlannedMigrations(t *testing.T) {
	dsn := os.Getenv("DEX_TEST_DSN")
	if dsn == "" {
		t.Skip("Test will not run without DEX_TEST_DSN environment variable.")
		return
	}
	dbMap := initDB(dsn)
	ms, err := GetPlannedMigrations(dbMap)
	if err != nil {
		pwd, err := os.Getwd()
		t.Logf("pwd: %v", pwd)
		t.Fatalf("unexpected err: %q", err)
	}

	if len(ms) == 0 {
		t.Fatalf("expected non-empty migrations")
	}
}

func TestMigrateClientMetadata(t *testing.T) {
	dsn := os.Getenv("DEX_TEST_DSN")
	if dsn == "" {
		t.Skip("Test will not run without DEX_TEST_DSN environment variable.")
		return
	}
	dbMap := initDB(dsn)

	nMigrations := 9
	n, err := MigrateMaxMigrations(dbMap, nMigrations)
	if err != nil {
		t.Fatalf("failed to perform initial migration: %v", err)
	}
	if n != nMigrations {
		t.Fatalf("expected to perform %d migrations, got %d", nMigrations, n)
	}

	tests := []struct {
		before string
		after  string
	}{
		// only update rows without a "redirect_uris" key
		{
			`{"redirectURLs":["foo"]}`,
			`{"redirectURLs" : ["foo"], "redirect_uris" : ["foo"]}`,
		},
		{
			`{"redirectURLs":["foo","bar"]}`,
			`{"redirectURLs" : ["foo","bar"], "redirect_uris" : ["foo","bar"]}`,
		},
		{
			`{"redirect_uris":["foo"],"another_field":8}`,
			`{"redirect_uris":["foo"],"another_field":8}`,
		},
		{
			`{"redirectURLs" : ["foo"], "redirect_uris" : ["foo"]}`,
			`{"redirectURLs" : ["foo"], "redirect_uris" : ["foo"]}`,
		},
	}

	for i, tt := range tests {
		model := &clientIdentityModel{
			ID:       strconv.Itoa(i),
			Secret:   []byte("verysecret"),
			Metadata: tt.before,
		}
		if err := dbMap.Insert(model); err != nil {
			t.Fatalf("could not insert model: %v", err)
		}
	}

	n, err = MigrateMaxMigrations(dbMap, 1)
	if err != nil {
		t.Fatalf("failed to perform initial migration: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected to perform 1 migration, got %d", n)
	}

	for i, tt := range tests {
		id := strconv.Itoa(i)
		m, err := dbMap.Get(clientIdentityModel{}, id)
		if err != nil {
			t.Errorf("case %d: failed to get model: %v", i, err)
			continue
		}
		cim, ok := m.(*clientIdentityModel)
		if !ok {
			t.Errorf("case %d: unrecognized model type: %T", i, m)
			continue
		}

		if cim.Metadata != tt.after {
			t.Errorf("case %d: want=%q, got=%q", i, tt.after, cim.Metadata)
		}
	}
}

func TestMigrationNumber11(t *testing.T) {
	dsn := os.Getenv("DEX_TEST_DSN")
	if dsn == "" {
		t.Skip("Test will not run without DEX_TEST_DSN environment variable.")
		return
	}

	tests := []struct {
		sqlStmt    string
		wantEmails []string
		wantError  bool
	}{
		{
			sqlStmt: `INSERT INTO authd_user
				(id, email, email_verified, display_name, admin, created_at)
			VALUES
				(1, 'Foo@example.com', TRUE, 'foo', FALSE, extract(epoch from now())),
				(2, 'Bar@example.com', TRUE, 'foo', FALSE, extract(epoch from now()))
			;`,
			wantEmails: []string{"foo@example.com", "bar@example.com"},
			wantError:  false,
		},
		{
			sqlStmt: `INSERT INTO authd_user
				(id, email, email_verified, display_name, admin, created_at)
			VALUES
				(1, 'Foo@example.com', TRUE, 'foo', FALSE, extract(epoch from now())),
				(2, 'foo@example.com', TRUE, 'foo', FALSE, extract(epoch from now())),
				(3, 'bar@example.com', TRUE, 'foo', FALSE, extract(epoch from now()))
			;`,
			wantError: true,
		},
	}
	migrateN := func(dbMap *gorp.DbMap, n int) error {
		nPerformed, err := MigrateMaxMigrations(dbMap, n)
		if err == nil && n != nPerformed {
			err = fmt.Errorf("expected to perform %d migrations, performed %d", n, nPerformed)
		}
		return err
	}

	for i, tt := range tests {
		err := func() error {
			dbMap := initDB(dsn)

			nMigrations := 10
			if err := migrateN(dbMap, nMigrations); err != nil {
				return fmt.Errorf("failed to perform initial migration: %v", err)
			}
			if _, err := dbMap.Exec(tt.sqlStmt); err != nil {
				return fmt.Errorf("failed to insert users: %v", err)
			}
			if err := migrateN(dbMap, 1); err != nil {
				if tt.wantError {
					return nil
				}
				return fmt.Errorf("failed to perform migration: %v", err)
			}

			if tt.wantError {
				return fmt.Errorf("expected an error when migrating")
			}

			var gotEmails []string
			if _, err := dbMap.Select(&gotEmails, `SELECT email FROM authd_user;`); err != nil {
				return fmt.Errorf("could not get user emails: %v", err)
			}

			sort.Strings(tt.wantEmails)
			sort.Strings(gotEmails)
			if diff := pretty.Compare(tt.wantEmails, gotEmails); diff != "" {
				return fmt.Errorf("wantEmails != gotEmails: %s", diff)
			}
			return nil
		}()
		if err != nil {
			t.Errorf("case %d: %v", i, err)
		}
	}
}

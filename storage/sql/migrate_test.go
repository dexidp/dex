//go:build cgo
// +build cgo

package sql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/dexidp/dex/storage"
	sqlite3 "github.com/mattn/go-sqlite3"
)

func TestMigrate(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	logger := slog.New(slog.DiscardHandler)

	errCheck := func(err error) bool {
		sqlErr, ok := err.(sqlite3.Error)
		if !ok {
			return false
		}
		return sqlErr.ExtendedCode == sqlite3.ErrConstraintUnique
	}

	var sqliteMigrations []migration
	for _, m := range migrations {
		if m.flavor == nil || m.flavor == &flavorSQLite3 {
			sqliteMigrations = append(sqliteMigrations, m)
		}
	}

	c := &conn{
		db:                 db,
		flavor:             &flavorSQLite3,
		logger:             logger,
		alreadyExistsCheck: errCheck,
	}
	for _, want := range []int{len(sqliteMigrations), 0} {
		got, err := c.migrate()
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("expected %d migrations, got %d", want, got)
		}
	}
}

func TestMigrateUnencryptedConnectors(t *testing.T) {
	// Create database without encryption
	dbFile, err := os.CreateTemp("", "dex-migration-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbFile.Close()
	defer os.Remove(dbFile.Name())

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create storage without encryption
	s1 := &SQLite3{
		File: dbFile.Name(),
		Encryption: EncryptionConfig{
			Enabled: false,
		},
	}
	conn1, err := s1.open(logger)
	if err != nil {
		t.Fatal(err)
	}

	// Create test connectors (unencrypted)
	testConnectors := []storage.Connector{
		{
			ID:              "ldap-test",
			Type:            "ldap",
			Name:            "LDAP Test",
			ResourceVersion: "1",
			Config:          []byte(`{"host":"ldap.example.com","bindDN":"cn=admin","bindPW":"secret123"}`),
		},
		{
			ID:              "oidc-test",
			Type:            "oidc",
			Name:            "OIDC Test",
			ResourceVersion: "1",
			Config:          []byte(`{"issuer":"https://example.com","clientID":"client123","clientSecret":"supersecret456"}`),
		},
	}

	for _, conn := range testConnectors {
		if err := conn1.CreateConnector(context.Background(), conn); err != nil {
			t.Fatal(err)
		}
	}

	// Verify configs are NOT encrypted
	for _, conn := range testConnectors {
		retrieved, err := conn1.GetConnector(context.Background(), conn.ID)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(retrieved.Config), "encrypted:") {
			t.Fatal("connector should not be encrypted yet")
		}
	}

	conn1.Close()
	t.Log("Created unencrypted connectors")

	// Reopen with encryption enabled
	os.Setenv("DEX_FERNET_KEY", "cHxZB8z3TcK9mR6vL2nY5qW8sD1fG4hJ7kM0oP3rT6u=")
	defer os.Unsetenv("DEX_FERNET_KEY")

	s2 := &SQLite3{
		File: dbFile.Name(),
		Encryption: EncryptionConfig{
			Enabled:   true,
			KeyEnvVar: "DEX_FERNET_KEY",
		},
	}
	conn2, err := s2.open(logger)
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close()

	t.Log("Reopened with encryption enabled (migration should have run)")

	// Verify configs are encrypted in database
	var encryptedCount int
	rows, err := conn2.db.Query("SELECT id, config FROM connector")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var config []byte
		if err := rows.Scan(&id, &config); err != nil {
			t.Fatal(err)
		}

		fmt.Println("config ", string(config))

		if !strings.Contains(string(config), "encrypted:") {
			t.Fatalf("connector %s should be encrypted in database", id)
		}

		// Verify sensitive fields are encrypted
		if strings.Contains(string(config), "secret123") {
			t.Fatal("plain text password found in database")
		}
		if strings.Contains(string(config), "supersecret456") {
			t.Fatal("plain text client secret found in database")
		}

		encryptedCount++
	}

	if encryptedCount != len(testConnectors) {
		t.Fatalf("expected %d encrypted connectors, got %d", len(testConnectors), encryptedCount)
	}

	t.Log("Verified all connectors encrypted in database")

	// Verify we can still read and decrypt them
	for _, originalConn := range testConnectors {
		retrieved, err := conn2.GetConnector(context.Background(), originalConn.ID)
		if err != nil {
			t.Fatal(err)
		}

		// Retrieved config should be decrypted
		if strings.Contains(string(retrieved.Config), "encrypted:") {
			t.Fatal("retrieved connector should be decrypted")
		}

		var retrievedMap, originalMap map[string]any
		if err := json.Unmarshal(retrieved.Config, &retrievedMap); err != nil {
			t.Fatalf("failed to parse retrieved config: %v", err)
		}
		if err := json.Unmarshal(originalConn.Config, &originalMap); err != nil {
			t.Fatalf("failed to parse original config: %v", err)
		}

		// Check all keys match
		if len(retrievedMap) != len(originalMap) {
			t.Fatalf("config key count mismatch for %s: got %d, want %d",
				originalConn.ID, len(retrievedMap), len(originalMap))
		}

		// Verify each key-value pair
		for key, originalVal := range originalMap {
			retrievedVal, ok := retrievedMap[key]
			if !ok {
				t.Fatalf("missing key %s in retrieved config for %s", key, originalConn.ID)
			}
			if fmt.Sprint(retrievedVal) != fmt.Sprint(originalVal) {
				t.Fatalf("value mismatch for key %s in %s: got %v, want %v",
					key, originalConn.ID, retrievedVal, originalVal)
			}
		}
	}

	t.Log("Successfully decrypted and verified all connectors")
}

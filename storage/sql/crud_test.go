//go:build cgo
// +build cgo

package sql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestDecoder(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`create table foo ( id integer primary key, bar blob );`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`insert into foo ( id, bar ) values (1, ?);`, []byte(`["a", "b"]`)); err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := db.QueryRow(`select bar from foo where id = 1;`).Scan(decoder(&got)); err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wanted %q got %q", want, got)
	}
}

func TestEncoder(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`create table foo ( id integer primary key, bar blob );`); err != nil {
		t.Fatal(err)
	}
	put := []string{"a", "b"}
	if _, err := db.Exec(`insert into foo ( id, bar ) values (1, ?)`, encoder(put)); err != nil {
		t.Fatal(err)
	}

	var got []byte
	if err := db.QueryRow(`select bar from foo where id = 1;`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	want := []byte(`["a","b"]`)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wanted %q got %q", want, got)
	}
}

// TestGetConnectorMixedEncryption tests that GetConnector can handle a mix of
// encrypted and unencrypted connector configs in the database.
func TestGetConnectorMixedEncryption(t *testing.T) {
	dbFile, err := os.CreateTemp("", "dex-mixed-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbFile.Close()
	defer os.Remove(dbFile.Name())

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Create storage with encryption enabled
	os.Setenv("DEX_FERNET_KEY", "cHxZB8z3TcK9mR6vL2nY5qW8sD1fG4hJ7kM0oP3rT6u=")
	defer os.Unsetenv("DEX_FERNET_KEY")

	s := &SQLite3{
		File: dbFile.Name(),
		Encryption: EncryptionConfig{
			Enabled:   true,
			KeyEnvVar: "DEX_FERNET_KEY",
		},
	}
	conn, err := s.open(logger)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	t.Log("✓ Created storage with encryption enabled")

	clientSecret := "google-secret-123"

	// Insert directly to connector WITHOUT encrypted config
	unencryptedConfig := fmt.Sprintf(`{"issuer":"https://accounts.google.com","clientID":"google-id","clientSecret":"%s"}`, clientSecret)
	_, err = conn.db.Exec(`INSERT INTO connector (id, type, name, resource_version, config) VALUES (?, ?, ?, ?, ?)`,
		"oidc-plain", "oidc", "Google Plain", "1", []byte(unencryptedConfig))
	if err != nil {
		t.Fatal(err)
	}

	t.Log("✓ Inserted unencrypted OIDC connector directly")

	// Insert directly to connector WITH encrypted config
	// First, encrypt a config manually
	encryptedSecret, err := conn.encryption.encryptor.encrypt(clientSecret)
	if err != nil {
		t.Fatal(err)
	}
	encryptedConfig := fmt.Sprintf(`{"issuer":"https://github.com","clientID":"github-id","clientSecret":"%s"}`, encryptedSecret)

	_, err = conn.db.Exec(`INSERT INTO connector (id, type, name, resource_version, config) VALUES (?, ?, ?, ?, ?)`,
		"oidc-encrypted", "oidc", "GitHub Encrypted", "1", []byte(encryptedConfig))
	if err != nil {
		t.Fatal(err)
	}

	t.Log("✓ Inserted encrypted OIDC connector directly")

	// Use GetConnector to retrieve both connectors
	plain, err := conn.GetConnector(context.Background(), "oidc-plain")
	if err != nil {
		t.Fatalf("failed to get plain connector: %v", err)
	}

	encrypted, err := conn.GetConnector(context.Background(), "oidc-encrypted")
	if err != nil {
		t.Fatalf("failed to get encrypted connector: %v", err)
	}

	t.Log("✓ Retrieved both connectors successfully")

	// Verify both are properly decrypted
	if strings.Contains(string(plain.Config), "encrypted:") {
		t.Fatal("plain connector should not have encrypted markers")
	}
	if strings.Contains(string(encrypted.Config), "encrypted:") {
		t.Fatal("encrypted connector should be fully decrypted")
	}

	// Verify secrets are present
	if !strings.Contains(string(plain.Config), clientSecret) {
		t.Fatal("plain connector missing secret")
	}
	if !strings.Contains(string(encrypted.Config), clientSecret) {
		t.Fatal("encrypted connector missing decrypted secret")
	}

	t.Log("✓ Both configs properly handled (unencrypted and encrypted)")
}

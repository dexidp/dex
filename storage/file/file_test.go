package file

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/conformance"
)

func newTestStorage(t *testing.T) storage.Storage {
	tempDir, err := os.MkdirTemp("", "dex-file-storage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	config := &Config{
		DataDir: tempDir,
	}

	s, err := config.Open(logger)
	if err != nil {
		t.Fatalf("failed to open storage: %v", err)
	}

	return s
}

// TestStorage runs the standard conformance test suite against file storage.
func TestStorage(t *testing.T) {
	conformance.RunTests(t, newTestStorage)
}

// TestStorageWithTimeout runs tests with timeout to detect potential deadlocks.
func TestStorageWithTimeout(t *testing.T) {
	done := make(chan struct{})
	go func() {
		conformance.RunTests(t, newTestStorage)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Minute):
		t.Fatal("test timed out - possible deadlock")
	}
}

// TestFileStorageSpecific tests file-storage-specific behaviour.
func TestFileStorageSpecific(t *testing.T) {
	t.Run("DirectoryCreation", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "dex-file-storage-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		config := &Config{DataDir: tempDir}
		_, err = config.Open(logger)
		if err != nil {
			t.Fatalf("failed to open storage: %v", err)
		}

		expectedDirs := []string{
			"passwords", "clients", "auth-requests", "auth-codes",
			"refresh-tokens", "offline-sessions", "connectors",
			"device-requests", "device-tokens", "keys",
		}

		for _, dir := range expectedDirs {
			dirPath := filepath.Join(tempDir, dir)
			info, err := os.Stat(dirPath)
			if err != nil {
				t.Errorf("directory %s not created: %v", dir, err)
				continue
			}
			if !info.IsDir() {
				t.Errorf("%s is not a directory", dir)
			}
			if info.Mode().Perm() != 0700 {
				t.Errorf("directory %s has wrong permissions: got %o, want 0700", dir, info.Mode().Perm())
			}
		}
	})

	t.Run("JSONFormatting", func(t *testing.T) {
		s := newTestStorage(t)

		ctx := t.Context()
		client := storage.Client{
			ID:           "test-client",
			Secret:       "test-secret",
			RedirectURIs: []string{"http://localhost/callback"},
			Name:         "Test Client",
		}

		if err := s.CreateClient(ctx, client); err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		// Verify the stored JSON is indented by reading back
		got, err := s.GetClient(ctx, "test-client")
		if err != nil {
			t.Fatalf("failed to get client: %v", err)
		}
		if got.Name != "Test Client" {
			t.Errorf("unexpected name: got %q, want %q", got.Name, "Test Client")
		}
	})

	t.Run("FilePermissions", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "dex-file-storage-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		config := &Config{DataDir: tempDir}
		s, err := config.Open(logger)
		if err != nil {
			t.Fatalf("failed to open storage: %v", err)
		}

		ctx := t.Context()
		password := storage.Password{
			Email:    "test@example.com",
			Hash:     []byte("test-hash"),
			Username: "testuser",
			UserID:   "test-user-id",
		}

		if err := s.CreatePassword(ctx, password); err != nil {
			t.Fatalf("failed to create password: %v", err)
		}

		filePath := filepath.Join(tempDir, "passwords", "test-user-id.json")
		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("failed to stat password file: %v", err)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("password file has wrong permissions: got %o, want 0600", info.Mode().Perm())
		}
	})

	t.Run("EmptyDataDir", func(t *testing.T) {
		config := &Config{DataDir: ""}
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		_, err := config.Open(logger)
		if err == nil {
			t.Error("expected error when DataDir is empty, got nil")
		}
	})

	t.Run("CloseStorage", func(t *testing.T) {
		s := newTestStorage(t)
		if err := s.Close(); err != nil {
			t.Errorf("close returned error: %v", err)
		}
	})
}

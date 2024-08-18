package main

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func TestNewLogger(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		logger, err := newLogger(slog.LevelInfo, "json")
		require.NoError(t, err)
		require.NotEqual(t, (*slog.Logger)(nil), logger)
	})

	t.Run("Text", func(t *testing.T) {
		logger, err := newLogger(slog.LevelError, "text")
		require.NoError(t, err)
		require.NotEqual(t, (*slog.Logger)(nil), logger)
	})

	t.Run("Unknown", func(t *testing.T) {
		logger, err := newLogger(slog.LevelError, "gofmt")
		require.Error(t, err)
		require.Equal(t, "log format is not one of the supported values (json, text): gofmt", err.Error())
		require.Equal(t, (*slog.Logger)(nil), logger)
	})
}

func TestStorageInitializationRetry(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a mock storage that fails a certain number of times before succeeding
	mockStorage := &mockRetryStorage{
		failuresLeft: 3,
	}

	config := Config{
		Issuer: "http://127.0.0.1:5556/dex",
		Storage: Storage{
			Type:          "mock",
			Config:        mockStorage,
			RetryAttempts: 5,
			RetryDelay:    "1s",
		},
		Web: Web{
			HTTP: "127.0.0.1:5556",
		},
		Logger: Logger{
			Level:  slog.LevelInfo,
			Format: "json",
		},
	}

	logger, _ := newLogger(config.Logger.Level, config.Logger.Format)

	s, err := initializeStorageWithRetry(config.Storage, logger)
	require.NoError(t, err)
	require.NotNil(t, s)

	require.Equal(t, 0, mockStorage.failuresLeft)
}

type mockRetryStorage struct {
	failuresLeft int
}

func (m *mockRetryStorage) Open(logger *slog.Logger) (storage.Storage, error) {
	if m.failuresLeft > 0 {
		m.failuresLeft--
		return nil, errors.New("mock storage failure")
	}
	return memory.New(logger), nil
}

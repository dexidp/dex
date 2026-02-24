package main

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
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

func TestGetSocketType(t *testing.T) {
	urls := [][]string{
		{"tcp://127.0.0.1:8080", "tcp"},
		{"unix:///tmp/my.sock", "unix"},
		{"127.0.0.1:9000", "tcp"},
		{"/var/run/app.sock", "unix"},
		{"./socket.sock", "unix"},
		{"relative.sock", "unix"},
		{"example.com:80", "tcp"},
		{"unix://./run/rel.sock", "unix"},
		{"[::1]:443", "tcp"},
		{"[::FFFF:129.144.52.38]:80", "tcp"},
		{"a/b/c", "unix"},
		{"/d/e/f", "unix"},
		{"localhost:80", "tcp"},
	}
	for _, url := range urls {
		t.Run(url[0], func(t *testing.T) {
			require.Equal(t, url[1], getSocketType(url[0]))
		})
	}
}

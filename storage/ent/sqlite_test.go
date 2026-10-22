package ent

import (
	"context"
	"log/slog"
	"reflect"
	"testing"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/conformance"
)

func newSQLiteStorage(t *testing.T) storage.Storage {
	logger := slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := SQLite3{File: ":memory:"}
	s, err := cfg.Open(logger)
	if err != nil {
		panic(err)
	}
	return s
}

func TestSQLite3(t *testing.T) {
	conformance.RunTests(t, newSQLiteStorage)
	conformance.RunConcurrencyTests(t, newSQLiteStorage)
}

func TestSQLite3AllowsPublicDynamicClientWithoutDisplayMetadata(t *testing.T) {
	s := newSQLiteStorage(t)
	t.Cleanup(func() { _ = s.Close() })

	want := storage.Client{
		ID:                      "public-dynamic-client",
		Public:                  true,
		DynamicallyRegistered:   true,
		RedirectURIs:            []string{"http://127.0.0.1/callback"},
		GrantTypes:              []string{"authorization_code"},
		ResponseTypes:           []string{"code"},
		AllowedScopes:           []string{"openid"},
		TokenEndpointAuthMethod: "none",
	}
	if err := s.CreateClient(context.Background(), want); err != nil {
		t.Fatalf("create public dynamic client: %v", err)
	}

	got, err := s.GetClient(context.Background(), want.ID)
	if err != nil {
		t.Fatalf("get public dynamic client: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("public dynamic client round trip mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

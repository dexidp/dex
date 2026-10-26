//go:build cgo
// +build cgo

package ent

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
	sqlstore "github.com/dexidp/dex/storage/sql"
)

func openBoth(t *testing.T) (sqlS, entS storage.Storage) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(t.Output(), nil))
	s1, err := (&sqlstore.SQLite3{File: ":memory:"}).Open(logger)
	require.NoError(t, err)
	t.Cleanup(func() { s1.Close() })
	s2, err := (&SQLite3{File: ":memory:"}).Open(logger)
	require.NoError(t, err)
	t.Cleanup(func() { s2.Close() })
	return s1, s2
}

func baseAuthCode(id string) storage.AuthCode {
	return storage.AuthCode{
		ID:          id,
		ClientID:    "client",
		RedirectURI: "https://example.com/cb",
		Nonce:       "nonce",
		Scopes:      []string{"openid"},
		ConnectorID: "mock",
		Expiry:      time.Now().Add(time.Hour).UTC(),
		Claims:      storage.Claims{UserID: "u1", Username: "name", Email: "a@b.c", EmailVerified: true},
		PKCE:        storage.PKCE{},
	}
}

func baseClient(id string) storage.Client {
	return storage.Client{
		ID:           id,
		Secret:       "secret",
		RedirectURIs: []string{"https://example.com/cb"},
		Name:         "app",
		LogoURL:      "https://example.com/logo.png",
	}
}

func TestOptionalAuthCodeFields(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*storage.AuthCode)
	}{
		{"nonce", func(a *storage.AuthCode) { a.Nonce = "" }},
		{"claims_username", func(a *storage.AuthCode) { a.Claims.Username = "" }},
		{"claims_email", func(a *storage.AuthCode) { a.Claims.Email = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sqlS, entS := openBoth(t)
			ctx := context.Background()
			a := baseAuthCode("code-" + tc.name)
			tc.mutate(&a)
			sqlErr := sqlS.CreateAuthCode(ctx, a)
			entErr := entS.CreateAuthCode(ctx, a)
			require.NoError(t, sqlErr, "sql backend error")
			require.NoError(t, entErr, "ent backend error")

			got, err := entS.GetAuthCode(ctx, a.ID)
			require.NoError(t, err)
			require.Equal(t, a.Nonce, got.Nonce)
			require.Equal(t, a.Claims.Username, got.Claims.Username)
			require.Equal(t, a.Claims.Email, got.Claims.Email)
		})
	}
}

func TestOptionalClientFields(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*storage.Client)
	}{
		{"client name", func(c *storage.Client) { c.Name = "" }},
		{"client logo_url", func(c *storage.Client) { c.LogoURL = "" }},
		{"public client empty secret", func(c *storage.Client) { c.Public = true; c.Secret = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sqlS, entS := openBoth(t)
			ctx := context.Background()
			c := baseClient("cli-" + tc.name)
			tc.mutate(&c)
			sqlErr := sqlS.CreateClient(ctx, c)
			entErr := entS.CreateClient(ctx, c)
			require.NoError(t, sqlErr, "sql backend error")
			require.NoError(t, entErr, "ent backend error")

			got, err := entS.GetClient(ctx, c.ID)
			require.NoError(t, err)
			require.Equal(t, c.Name, got.Name)
			require.Equal(t, c.LogoURL, got.LogoURL)
			require.Equal(t, c.Secret, got.Secret)
		})
	}
}

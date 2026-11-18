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

// A refresh token copies the nonce and claims from the auth code, so it must
// accept the same empty values, or offline_access fails at /token.
func TestOptionalRefreshTokenFields(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*storage.RefreshToken)
	}{
		{"nonce", func(r *storage.RefreshToken) { r.Nonce = "" }},
		{"claims_username", func(r *storage.RefreshToken) { r.Claims.Username = "" }},
		{"claims_email", func(r *storage.RefreshToken) { r.Claims.Email = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sqlS, entS := openBoth(t)
			ctx := context.Background()
			now := time.Now().UTC()
			r := storage.RefreshToken{
				ID:          "refresh-" + tc.name,
				Token:       "token",
				CreatedAt:   now,
				LastUsed:    now,
				ClientID:    "client",
				ConnectorID: "mock",
				Claims:      storage.Claims{UserID: "u1", Username: "name", Email: "a@b.c", EmailVerified: true},
				Scopes:      []string{"openid", "offline_access"},
				Nonce:       "nonce",
			}
			tc.mutate(&r)
			require.NoError(t, sqlS.CreateRefresh(ctx, r), "sql backend error")
			require.NoError(t, entS.CreateRefresh(ctx, r), "ent backend error")

			got, err := entS.GetRefresh(ctx, r.ID)
			require.NoError(t, err)
			require.Equal(t, r.Nonce, got.Nonce)
			require.Equal(t, r.Claims.Username, got.Claims.Username)
			require.Equal(t, r.Claims.Email, got.Claims.Email)
		})
	}
}

// A public client starts the device flow without a client secret.
func TestOptionalDeviceRequestSecret(t *testing.T) {
	sqlS, entS := openBoth(t)
	ctx := context.Background()
	d := storage.DeviceRequest{
		UserCode:   "ABCD-EFGH",
		DeviceCode: "device-code",
		ClientID:   "client",
		Scopes:     []string{"openid"},
		Expiry:     time.Now().Add(time.Hour).UTC(),
	}
	require.NoError(t, sqlS.CreateDeviceRequest(ctx, d), "sql backend error")
	require.NoError(t, entS.CreateDeviceRequest(ctx, d), "ent backend error")

	got, err := entS.GetDeviceRequest(ctx, d.UserCode)
	require.NoError(t, err)
	require.Empty(t, got.ClientSecret)
}

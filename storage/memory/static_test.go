package memory

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/dexidp/dex/storage"
)

func TestStaticClients(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	backing := New(logger)

	c1 := storage.Client{ID: "foo", Secret: "foo_secret"}
	c2 := storage.Client{ID: "bar", Secret: "bar_secret"}
	c3 := storage.Client{ID: "spam", Secret: "spam_secret"}

	backing.CreateClient(ctx, c1)
	s := storage.WithStaticClients(backing, []storage.Client{c2})

	tests := []struct {
		name    string
		action  func() error
		wantErr bool
	}{
		{
			name: "get client from static storage",
			action: func() error {
				_, err := s.GetClient(ctx, c2.ID)
				return err
			},
		},
		{
			name: "get client from backing storage",
			action: func() error {
				_, err := s.GetClient(ctx, c1.ID)
				return err
			},
		},
		{
			name: "update static client",
			action: func() error {
				updater := func(c storage.Client) (storage.Client, error) {
					c.Secret = "new_" + c.Secret
					return c, nil
				}
				return s.UpdateClient(ctx, c2.ID, updater)
			},
			wantErr: true,
		},
		{
			name: "update non-static client",
			action: func() error {
				updater := func(c storage.Client) (storage.Client, error) {
					c.Secret = "new_" + c.Secret
					return c, nil
				}
				return s.UpdateClient(ctx, c1.ID, updater)
			},
		},
		{
			name: "list clients",
			action: func() error {
				clients, err := s.ListClients(ctx)
				if err != nil {
					return err
				}
				if n := len(clients); n != 2 {
					return fmt.Errorf("expected 2 clients got %d", n)
				}
				return nil
			},
		},
		{
			name: "create client",
			action: func() error {
				return s.CreateClient(ctx, c3)
			},
		},
	}

	for _, tc := range tests {
		err := tc.action()
		if err != nil && !tc.wantErr {
			t.Errorf("%s: %v", tc.name, err)
		}
		if err == nil && tc.wantErr {
			t.Errorf("%s: expected error, didn't get one", tc.name)
		}
	}
}

func TestStaticPasswords(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	backing := New(logger)

	p1 := storage.Password{Email: "foo@example.com", Username: "foo_secret"}
	p2 := storage.Password{Email: "bar@example.com", Username: "bar_secret"}
	p3 := storage.Password{Email: "spam@example.com", Username: "spam_secret"}
	p4 := storage.Password{Email: "Spam@example.com", Username: "Spam_secret"}

	backing.CreatePassword(ctx, p1)
	s := storage.WithStaticPasswords(backing, []storage.Password{p2}, logger)

	tests := []struct {
		name    string
		action  func() error
		wantErr bool
	}{
		{
			name: "get password from static storage",
			action: func() error {
				_, err := s.GetPassword(ctx, p2.Email)
				return err
			},
		},
		{
			name: "get password from backing storage",
			action: func() error {
				_, err := s.GetPassword(ctx, p1.Email)
				return err
			},
		},
		{
			name: "get password from static storage with casing",
			action: func() error {
				_, err := s.GetPassword(ctx, strings.ToUpper(p2.Email))
				return err
			},
		},
		{
			name: "update static password",
			action: func() error {
				updater := func(p storage.Password) (storage.Password, error) {
					p.Username = "new_" + p.Username
					return p, nil
				}
				return s.UpdatePassword(ctx, p2.Email, updater)
			},
			wantErr: true,
		},
		{
			name: "update non-static password",
			action: func() error {
				updater := func(p storage.Password) (storage.Password, error) {
					p.Username = "new_" + p.Username
					return p, nil
				}
				return s.UpdatePassword(ctx, p1.Email, updater)
			},
		},
		{
			name: "create passwords",
			action: func() error {
				if err := s.CreatePassword(ctx, p4); err != nil {
					return err
				}
				return s.CreatePassword(ctx, p3)
			},
			wantErr: true,
		},
		{
			name: "get password",
			action: func() error {
				p, err := s.GetPassword(ctx, p4.Email)
				if err != nil {
					return err
				}
				if strings.Compare(p.Email, p4.Email) != 0 {
					return fmt.Errorf("expected %s passwords got %s", p4.Email, p.Email)
				}
				return nil
			},
		},
		{
			name: "list passwords",
			action: func() error {
				passwords, err := s.ListPasswords(ctx)
				if err != nil {
					return err
				}
				if n := len(passwords); n != 3 {
					return fmt.Errorf("expected 3 passwords got %d", n)
				}
				return nil
			},
		},
	}

	for _, tc := range tests {
		err := tc.action()
		if err != nil && !tc.wantErr {
			t.Errorf("%s: %v", tc.name, err)
		}
		if err == nil && tc.wantErr {
			t.Errorf("%s: expected error, didn't get one", tc.name)
		}
	}
}

func TestStaticConnectors(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	backing := New(logger)

	config1 := []byte(`{"issuer": "https://accounts.google.com"}`)
	config2 := []byte(`{"host": "ldap.example.com:636"}`)
	config3 := []byte(`{"issuer": "https://example.com"}`)

	c1 := storage.Connector{ID: storage.NewID(), Type: "oidc", Name: "oidc", ResourceVersion: "1", Config: config1}
	c2 := storage.Connector{ID: storage.NewID(), Type: "ldap", Name: "ldap", ResourceVersion: "1", Config: config2}
	c3 := storage.Connector{ID: storage.NewID(), Type: "saml", Name: "saml", ResourceVersion: "1", Config: config3}

	backing.CreateConnector(ctx, c1)
	s := storage.WithStaticConnectors(backing, []storage.Connector{c2})

	tests := []struct {
		name    string
		action  func() error
		wantErr bool
	}{
		{
			name: "get connector from static storage",
			action: func() error {
				_, err := s.GetConnector(ctx, c2.ID)
				return err
			},
		},
		{
			name: "get connector from backing storage",
			action: func() error {
				_, err := s.GetConnector(ctx, c1.ID)
				return err
			},
		},
		{
			name: "update static connector",
			action: func() error {
				updater := func(c storage.Connector) (storage.Connector, error) {
					c.Name = "New"
					return c, nil
				}
				return s.UpdateConnector(ctx, c2.ID, updater)
			},
			wantErr: true,
		},
		{
			name: "update non-static connector",
			action: func() error {
				updater := func(c storage.Connector) (storage.Connector, error) {
					c.Name = "New"
					return c, nil
				}
				return s.UpdateConnector(ctx, c1.ID, updater)
			},
		},
		{
			name: "list connectors",
			action: func() error {
				connectors, err := s.ListConnectors(ctx)
				if err != nil {
					return err
				}
				if n := len(connectors); n != 2 {
					return fmt.Errorf("expected 2 connectors got %d", n)
				}
				return nil
			},
		},
		{
			name: "create connector",
			action: func() error {
				return s.CreateConnector(ctx, c3)
			},
		},
	}

	for _, tc := range tests {
		err := tc.action()
		if err != nil && !tc.wantErr {
			t.Errorf("%s: %v", tc.name, err)
		}
		if err == nil && tc.wantErr {
			t.Errorf("%s: expected error, didn't get one", tc.name)
		}
	}
}

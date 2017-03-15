package memory

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/dex/storage"
)

func TestStaticClients(t *testing.T) {
	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: &logrus.TextFormatter{DisableColors: true},
		Level:     logrus.DebugLevel,
	}
	backing := New(logger)

	c1 := storage.Client{ID: "foo", Secret: "foo_secret"}
	c2 := storage.Client{ID: "bar", Secret: "bar_secret"}
	c3 := storage.Client{ID: "spam", Secret: "spam_secret"}

	backing.CreateClient(c1)
	s := storage.WithStaticClients(backing, []storage.Client{c2})

	tests := []struct {
		name    string
		action  func() error
		wantErr bool
	}{
		{
			name: "get client from static storage",
			action: func() error {
				_, err := s.GetClient(c2.ID)
				return err
			},
		},
		{
			name: "get client from backing storage",
			action: func() error {
				_, err := s.GetClient(c1.ID)
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
				return s.UpdateClient(c2.ID, updater)
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
				return s.UpdateClient(c1.ID, updater)
			},
		},
		{
			name: "list clients",
			action: func() error {
				clients, err := s.ListClients()
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
				return s.CreateClient(c3)
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
	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: &logrus.TextFormatter{DisableColors: true},
		Level:     logrus.DebugLevel,
	}
	backing := New(logger)

	p1 := storage.Password{Email: "foo@example.com", Username: "foo_secret"}
	p2 := storage.Password{Email: "bar@example.com", Username: "bar_secret"}
	p3 := storage.Password{Email: "spam@example.com", Username: "spam_secret"}

	backing.CreatePassword(p1)
	s := storage.WithStaticPasswords(backing, []storage.Password{p2})

	tests := []struct {
		name    string
		action  func() error
		wantErr bool
	}{
		{
			name: "get password from static storage",
			action: func() error {
				_, err := s.GetPassword(p2.Email)
				return err
			},
		},
		{
			name: "get password from backing storage",
			action: func() error {
				_, err := s.GetPassword(p1.Email)
				return err
			},
		},
		{
			name: "get password from static storage with casing",
			action: func() error {
				_, err := s.GetPassword(strings.ToUpper(p2.Email))
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
				return s.UpdatePassword(p2.Email, updater)
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
				return s.UpdatePassword(p1.Email, updater)
			},
		},
		{
			name: "list passwords",
			action: func() error {
				passwords, err := s.ListPasswords()
				if err != nil {
					return err
				}
				if n := len(passwords); n != 2 {
					return fmt.Errorf("expected 2 passwords got %d", n)
				}
				return nil
			},
		},
		{
			name: "create password",
			action: func() error {
				return s.CreatePassword(p3)
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

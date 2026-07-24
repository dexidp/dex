package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/bcrypt"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/passwords"
	"github.com/dexidp/dex/storage"
)

// LocalConnector is the local passwordDB connector: an internal connector,
// backed by the password store, that is not part of the injected config map.
const LocalConnector = "local"

// ConnectorConfig is a configuration that can open a connector.
type ConnectorConfig interface {
	Open(id string, logger *slog.Logger) (connector.Connector, error)
}

// Resolver returns a ResolveFunc that builds the underlying implementation for a
// stored connector: the built-in local password DB (backed by storage), or a
// connector from the given config map. The map is injected by the caller so this
// package need not import any connector implementation — a library consumer can
// pass its own set of connectors.
func Resolver(store storage.Storage, logger *slog.Logger, configs map[string]func() ConnectorConfig) ResolveFunc {
	return func(conn storage.Connector) (connector.Connector, error) {
		if conn.Type == LocalConnector {
			return NewPasswordDB(store), nil
		}
		return openConnector(logger, configs, conn)
	}
}

// openConnector parses the stored config and opens the connector named by its type.
func openConnector(logger *slog.Logger, configs map[string]func() ConnectorConfig, conn storage.Connector) (connector.Connector, error) {
	var c connector.Connector

	f, ok := configs[conn.Type]
	if !ok {
		return c, fmt.Errorf("unknown connector type %q", conn.Type)
	}

	connConfig := f()
	if len(conn.Config) != 0 {
		data := []byte(string(conn.Config))
		if err := json.Unmarshal(data, connConfig); err != nil {
			return c, fmt.Errorf("parse connector config: %v", err)
		}
	}

	c, err := connConfig.Open(conn.ID, logger)
	if err != nil {
		return c, fmt.Errorf("failed to create connector %s: %v", conn.ID, err)
	}

	return c, nil
}

// NewPasswordDB returns the built-in local password connector backed by the
// password store. Resolver uses it for LocalConnector; it is exported so a
// custom ResolveFunc can reuse it.
func NewPasswordDB(s storage.Storage) interface {
	connector.Connector
	connector.PasswordConnector
} {
	return passwordDB{s}
}

type passwordDB struct {
	s storage.Storage
}

func resolvePasswordName(p storage.Password) string {
	if p.Name != "" {
		return p.Name
	}
	return p.Username
}

func resolvePasswordEmailVerified(p storage.Password) bool {
	if p.EmailVerified != nil {
		return *p.EmailVerified
	}
	return true
}

func (db passwordDB) Login(ctx context.Context, s connector.Scopes, email, password string) (connector.Identity, bool, error) {
	p, err := db.s.GetPassword(ctx, email)
	if err != nil {
		if err != storage.ErrNotFound {
			return connector.Identity{}, false, fmt.Errorf("get password: %v", err)
		}
		return connector.Identity{}, false, nil
	}
	// This check prevents dex users from logging in using static passwords
	// configured with hash costs that are too high or low.
	if err := passwords.CheckCost(p.Hash); err != nil {
		return connector.Identity{}, false, err
	}
	if err := bcrypt.CompareHashAndPassword(p.Hash, []byte(password)); err != nil {
		return connector.Identity{}, false, nil
	}
	return connector.Identity{
		UserID:            p.UserID,
		Username:          resolvePasswordName(p),
		PreferredUsername: p.PreferredUsername,
		Email:             p.Email,
		EmailVerified:     resolvePasswordEmailVerified(p),
		Groups:            p.Groups,
	}, true, nil
}

func (db passwordDB) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	// If the user has been deleted, the refresh token will be rejected.
	p, err := db.s.GetPassword(ctx, identity.Email)
	if err != nil {
		if err == storage.ErrNotFound {
			return connector.Identity{}, errors.New("user not found")
		}
		return connector.Identity{}, fmt.Errorf("get password: %v", err)
	}

	// User removed but a new user with the same email exists.
	if p.UserID != identity.UserID {
		return connector.Identity{}, errors.New("user not found")
	}

	// If a user has updated their username, that will be reflected in the
	// refreshed token.
	//
	// No other fields are expected to be refreshable as email is effectively used
	// as an ID.
	identity.Username = resolvePasswordName(p)
	identity.PreferredUsername = p.PreferredUsername
	identity.EmailVerified = resolvePasswordEmailVerified(p)
	identity.Groups = p.Groups

	return identity, nil
}

func (db passwordDB) Prompt() string {
	return "Email Address"
}

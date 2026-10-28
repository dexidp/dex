package connectors

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/passwords"
	"github.com/dexidp/dex/storage"
)

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

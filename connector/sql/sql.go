package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	// import third party drivers
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/log"
)

// Config holds the configuration for a mock connector which prompts for the supplied
// username and password.
type Config struct {
	Database   string `json:"database"`
	Connection string `json:"connection"`
	Prompt     string `json:"prompt"`

	// Required queries
	Login       string `json:"login"`
	GetIdentity string `json:"get-identity"`

	// Optional queries
	GetGroups string `json:"groups"`
}

// Open returns an authentication strategy which prompts for a predefined username and password.
func (c *Config) Open(id string, logger log.Logger) (connector.Connector, error) {
	if c.Database == "" {
		return nil, errors.New("no 'database' name supplied")
	}
	if c.Connection == "" {
		return nil, errors.New("no 'connection' string supplied")
	}

	// Connect
	db, err := sql.Open(c.Database, c.Connection)
	if err != nil {
		return &sqlConnector{}, err
	}

	db.SetConnMaxLifetime(0)
	db.SetMaxOpenConns(5)

	logger.Info("SQL Connector: Open")

	return &sqlConnector{
		db:     db,
		config: c,
		logger: logger,
	}, nil
}

type sqlConnector struct {
	db     *sql.DB
	config *Config
	logger log.Logger
}

func (p sqlConnector) Close() error {
	p.logger.Info("SQL Connector: Closing")
	return p.db.Close()
}

func (p sqlConnector) Login(ctx context.Context, s connector.Scopes, username, password string) (identity connector.Identity, validPassword bool, err error) {
	p.logger.Infof("SQL Connector: Login scopes=%v username=%s", s, username)
	validPassword, err = p.checkUsernameAndPassword(ctx, s, username, password)
	p.logger.Infof("SQL Connector: Login password checked scopes=%v username=%s validPassword=%b", s, username, validPassword)
	if !validPassword {
		return connector.Identity{}, false, nil
	} else if err != nil {
		p.logger.Infof("SQL Connector: Error cheching password err=%s", err)
		return connector.Identity{}, false, err
	}

	id, err := p.createIdentity(ctx, s, username)
	p.logger.Infof("SQL Connector: Login create identity scopes=%v username=%s id=%v", s, username, validPassword, id)
	if err != nil {
		p.logger.Infof("SQL Connector: Error creating identity err=%s", err)
	}
	return id, true, err
}

func (p sqlConnector) Prompt() string { return p.config.Prompt }

func (p sqlConnector) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	p.logger.Infof("SQL Connector: Refresh scopes=%v identity=%v", s, identity)
	return p.createIdentity(ctx, s, string(identity.ConnectorData))
}

func (p *sqlConnector) createIdentity(ctx context.Context, s connector.Scopes, username string) (connector.Identity, error) {
	// Refresh identity
	identity, err := p.getIdentityFromUsername(ctx, s, username)
	if err != nil {
		// getIdentityFromUsername() returns a nice error if the user isn't found
		return connector.Identity{}, err
	}

	// Set ConnectorData to username if we're gonna do OfflineAccess (i.e. set a refresh-token)
	if s.OfflineAccess {
		identity.ConnectorData = []byte(identity.Username)
	}

	// Get groups, if requested
	if s.Groups {
		groups, err := p.getGroups(ctx, s, identity.Username)
		if err == nil {
			identity.Groups = groups
		}
	} else {
		identity.Groups = []string{}
	}
	return identity, nil
}

func (p *sqlConnector) checkUsernameAndPassword(ctx context.Context, s connector.Scopes, username, password string) (bool, error) {
	if password == "" {
		return false, nil
	}

	validPassword := false

	// Check password is valid
	err := p.db.QueryRowContext(
		ctx,
		p.config.Login,
		sql.Named("username", username),
		sql.Named("password", password),
	).Scan(&validPassword)

	if err == sql.ErrNoRows || !validPassword {
		return false, nil
	}
	return validPassword, err
}

func (p *sqlConnector) getIdentityFromUsername(ctx context.Context, s connector.Scopes, username string) (connector.Identity, error) {
	id := connector.Identity{}
	err := p.db.QueryRowContext(
		ctx,
		p.config.GetIdentity,
		sql.Named("username", username),
	).Scan(
		&id.UserID,
		&id.Username,
		&id.PreferredUsername,
		&id.Email,
		&id.EmailVerified,
	)
	if err == sql.ErrNoRows {
		return id, fmt.Errorf("failed to find user: %s", username)
	}
	return id, err
}

func (p sqlConnector) getGroups(ctx context.Context, s connector.Scopes, username string) ([]string, error) {
	if p.config.GetGroups == "" {
		return []string{}, nil
	}

	rows, err := p.db.QueryContext(
		ctx,
		p.config.GetGroups,
		sql.Named("username", username),
	)
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()

	groups := []string{}
	for rows.Next() {
		var groupName string
		err := rows.Scan(&groupName)
		if err != nil {
			return []string{}, err
		}
		groups = append(groups, groupName)
	}
	return groups, nil
}

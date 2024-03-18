package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

// Fix Postgres query parameters - other parts of Dex really likes to replace
// environment-variables, messing with Postgres' use of $1, $2, ... $xxx as
// query parameters (github.com/lib/pq doesn't support named paremters yet).
//
// Thus, we'll make a valliant effort to fix up other interpolation-methods
// such as :xxx and @xxx to the postgres-syntax.
func (c *Config) FixPostgresQueryParameters() *Config {
	conf := *c
	if c.Database != "postgres" {
		return &conf
	}

	// TODO: We probably should use regexp.ReplaceAllFunc, but we'll setting
	// for stupid string stuff for now
	replacer := func(in string) string {
		tmp := strings.ReplaceAll(in, " @", " $")
		return strings.ReplaceAll(tmp, " :", " $")
	}
	conf.Login = replacer(conf.Login)
	conf.GetIdentity = replacer(conf.GetIdentity)
	conf.GetGroups = replacer(conf.GetGroups)

	return &conf
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
	logger.Infof("SQL Connector: Opening config %+v", c)
	db, err := sql.Open(c.Database, c.Connection)
	if err != nil {
		logger.Errorf("SQL Connector: Failed to open database: %s", err)
		return &sqlConnector{}, err
	}

	// Do a ping to check connection works
	err = db.Ping()
	if err != nil {
		logger.Errorf("SQL Connector: Failed to ping database: %s", err)
		return nil, err
	}

	db.SetConnMaxLifetime(0)
	db.SetMaxOpenConns(5)

	config := c.FixPostgresQueryParameters()

	logger.Infof("SQL Connector: Connection successful, config is: %+v", config)

	return &sqlConnector{
		db:     db,
		config: config,
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
	p.logger.Infof("SQL Connector: Login password checked scopes=%v username=%s validPassword=%t", s, username, validPassword)
	if !validPassword {
		return connector.Identity{}, false, nil
	} else if err != nil {
		p.logger.Infof("SQL Connector: Error cheching password err=%s", err)
		return connector.Identity{}, false, err
	}

	id, err := p.createIdentity(ctx, s, username)
	p.logger.Infof("SQL Connector: Login create identity scopes=%v username=%s id=%v", s, username, id)
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
		p.logger.Info("SQL Connector: Checking password: rejecting empty password")
		return false, nil
	}

	validPassword := false

	p.logger.Infof("SQL Connector: Checking password scopes=%v username=%s", s, username)

	// Check password is valid
	err := p.db.QueryRowContext(
		ctx,
		p.config.Login,
		username,
		password,
	).Scan(&validPassword)

	if err == sql.ErrNoRows || !validPassword {
		return false, nil
	}

	if err != nil {
		p.logger.Error("SQL Connector: Could not check password err=%s", err)
	}

	return validPassword, err
}

func (p *sqlConnector) getIdentityFromUsername(ctx context.Context, s connector.Scopes, username string) (connector.Identity, error) {
	id := connector.Identity{}
	err := p.db.QueryRowContext(
		ctx,
		p.config.GetIdentity,
		username,
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
		username,
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

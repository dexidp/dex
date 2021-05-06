// Package sql implements authentication using SQL queries.
package sql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	// Crypt support
	"github.com/al45tair/passlib"

	// Database driver
	_ "github.com/go-sql-driver/mysql"

	// Improved Go SQL support
	"github.com/jmoiron/sqlx"

	// Database driver
	_ "github.com/lib/pq"

	// Database driver
	_ "github.com/mattn/go-sqlite3"

	// Used to do case-folding
	"golang.org/x/text/cases"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/log"
)

// The SQL connector requires two queries, very much like the LDAP connector.
// The first finds the user based on the username and password given to the
// connector.  The second uses the user entry to search for groups.
//
// There are two versions of the first query, one keyed on username and one
// keyed on user ID.  The latter is used to implement refresh.
//
// The refresh logic assumes that user IDs (whatever they may be) are
// immutable and are never reused.

type Config struct {
	// The name of the database driver to use.  We pull in support for
	// PostgreSQL, MySQL, Sqlite3 and SQL Server.
	Driver string `json:"driver"`

	// The DSN; this is specific to the database driver.
	DSN string `json:"dsn"`

	// UsernamePrompt allows users to override the username attribute
	// (displayed in the username/password prompt).  If unset, the handler
	// wil use "Username".
	UsernamePrompt string `json:"usernamePrompt,omitempty"`

	// User queries, used to fetch user information
	UserQuery UserQuery `json:"userQuery"`

	// User groups query, used to fetch groups for a user
	UserGroupsQuery UserGroupsQuery `json:"userGroupsQuery,omitempty"`

	// SQL statements to run on login (successful or not); these can be used
	// to implement tracking of last login times/attempts, automatic lockout
	// and other similar features.
	Hooks Hooks `json:"hooks,omitempty"`

	// Hash schemes
	HashSchemes []string `json:"hashSchemes,omitempty"`

	// Only used if HashSchemes is not set
	PasslibDefaultsDate string `json:"passlibDefaultsDate,omitempty"`
}

type UserQuery struct {
	// SQL queries

	// QueryByName will substitute a ":username" argument, e.g.
	//
	//   SELECT id, username, email, name, password
	//   FROM Users
	//   WHERE username=:username OR email=:username
	//
	// It will also make the following additional substitutions:
	//
	//   :username_lower    - the username, transformed to lowercase
	//   :username_upper    - the username, transformed to UPPERCASE
	//   :username_casefold - the case-folded version of username
	//
	// These are Unicode aware.  The latter is NOT suitable for display to
	// the user, but it *is* the right way to make usernames case-insensitive;
	// that is, for proper Unicode case-sensitivity, you should store and
	// search by *case-folded* username, but always display a non-case-folded
	// username.
	//
	// A case-insensitive example is therefore:
	//
	//   SELECT id, username, email, name, password
	//   FROM Users
	//   WHERE folded_username=:username_casefold OR email=:username
	//
	// Note the extra column "folded_username" in addition to the normal
	// "username" column.
	//
	QueryByName string `json:"queryByUsername"`

	// QueryByID will substitute a ":userid" argument, e.g.
	//
	//   SELECT id, username, email, name, password
	//   FROM Users
	//   WHERE id=:userid
	//
	QueryByID string `json:"queryById"`

	// UpdatePassword updates the password; ":userid" and ":password"
	// are substituted, e.g.
	//
	//   UPDATE Users SET password=:password WHERE id=:userid
	//
	// This is only used if it is non-empty, and allows passlib to
	// automatically update passwords that have weak hashes.
	//
	// **NOTE THAT SETTING THIS WILL RESULT IN PASSLIB UPDATING YOUR HASHES
	//   TO ITS PREFERRED CRYPT ALGORITHM**
	//
	// If you use any software besides passlib to do password hashing,
	// you will very likely want to configure HashSchemes explicitly to
	// ensure that passlib doesn't update the hashes to an algorithm the
	// other software you are using does not support.
	//
	UpdatePassword string `json:"updatePassword,omitempty"`

	// The names of various columns
	IDColumn       string `json:"idColumn"`
	UsernameColumn string `json:"usernameColumn,omitempty"`
	EmailColumn    string `json:"emailColumn,omitempty"`
	NameColumn     string `json:"nameColumn,omitempty"`
	PasswordColumn string `json:"password"`
}

type UserGroupsQuery struct {
	// SQL queries

	// QueryByUserID will substitute a ":userid" argument
	QueryByUserID string `json:"queryByUserId"`

	// Column names in the result
	NameColumn string `json:"nameColumn"`
}

type Hooks struct {
	// If set, this SQL statement is executed on successful login, e.g.:
	//
	//   UPDATE Users SET lastLoginTime=NOW() WHERE id=:userid
	//
	OnSuccess string `json:"onSuccess,omitempty"`

	// If set, this SQL statement is executed on failed login, e.g.:
	//
	//   UPDATE Users
	//   SET lastFailedLoginTime=NOW(),
	//       failedLoginAttempts=failedLoginAttempts+1
	//   WHERE id=:userid
	//
	OnFailure string `json:"onFailure,omitempty"`
}

func (c *Config) Open(id string, logger log.Logger) (connector.Connector, error) {
	conn, err := c.OpenConnector(logger)
	if err != nil {
		return nil, err
	}
	return connector.Connector(conn), nil
}

func (c *Config) OpenConnector(logger log.Logger) (interface {
	connector.Connector
	connector.PasswordConnector
	connector.RefreshConnector
}, error) {
	return c.openConnector(logger)
}

func (c *Config) openConnector(logger log.Logger) (*sqlConnector, error) {
	db, err := sqlx.Open(c.Driver, c.DSN)
	if err != nil {
		logger.Errorf("sql: cannot connect to %q with driver %q: %s",
			c.DSN, c.Driver, err)
		return nil, err
	}

	ctx := &passlib.Context{}

	if c.HashSchemes != nil {
		ctx.Schemes, err = passlib.SchemesFromNames(c.HashSchemes)
		if err != nil {
			logger.Errorf("sql: %s", err)
			return nil, err
		}
	} else {
		date := c.PasslibDefaultsDate
		if date == "" {
			date = passlib.Defaults20180601
		}
		ctx.Schemes, err = passlib.DefaultSchemesFromDate(date)
		if err != nil {
			logger.Errorf("sql: %s", err)
			return nil, err
		}
	}

	return &sqlConnector{*c, ctx, db, logger}, nil
}

type sqlConnector struct {
	Config

	passlib *passlib.Context

	db *sqlx.DB

	logger log.Logger
}

type sqlRefreshData struct {
	UserID string `json:"userid"`
}

var (
	_ connector.PasswordConnector = (*sqlConnector)(nil)
	_ connector.RefreshConnector  = (*sqlConnector)(nil)
)

func (c *sqlConnector) identityFromRow(row map[string]interface{}) (ident connector.Identity, err error) {
	var ok bool

	id := row[c.UserQuery.IDColumn]
	if idint, ok := id.(int); ok {
		ident.UserID = fmt.Sprintf("%d", idint)
	} else if idstr, ok := id.(string); ok {
		ident.UserID = idstr
	} else {
		err = fmt.Errorf("IDColumn %s must be a string or int",
			c.UserQuery.IDColumn)
		return connector.Identity{}, err
	}

	if c.UserQuery.UsernameColumn != "" {
		data := row[c.UserQuery.UsernameColumn]
		if data != nil {
			ident.PreferredUsername, ok = data.(string)
			if !ok {
				err = fmt.Errorf("sql: UsernameColumn %s must be a string",
					c.UserQuery.UsernameColumn)
				return connector.Identity{}, err
			}
		}
	}

	if c.UserQuery.EmailColumn != "" {
		data := row[c.UserQuery.EmailColumn]
		if data != nil {
			ident.Email, ok = data.(string)
			if !ok {
				err = fmt.Errorf("sql: EmailColumn %s must be a string",
					c.UserQuery.EmailColumn)
				return connector.Identity{}, err
			}
		}
	}

	if c.UserQuery.NameColumn != "" {
		data := row[c.UserQuery.NameColumn]
		if data != nil {
			ident.Username, ok = data.(string)
			if !ok {
				err = fmt.Errorf("sql: NameColumn %s must be a string",
					c.UserQuery.NameColumn)
				return connector.Identity{}, err
			}
		}
	}

	return ident, nil
}

var caseFolder = cases.Fold()

func (c *sqlConnector) rowFromUsername(ctx context.Context,
	username string) (row map[string]interface{}, err error) {
	rows, err := c.db.NamedQueryContext(ctx, c.UserQuery.QueryByName,
		map[string]interface{}{
			"username":          username,
			"username_lower":    strings.ToLower(username),
			"username_upper":    strings.ToUpper(username),
			"username_casefold": caseFolder.String(username),
		})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		err = rows.Err()
		return nil, err
	}

	row = map[string]interface{}{}
	err = rows.MapScan(row)
	if err != nil {
		return nil, err
	}

	return row, nil
}

func (c *sqlConnector) Login(ctx context.Context, s connector.Scopes,
	username, password string) (ident connector.Identity,
	validPass bool, err error) {
	row, err := c.rowFromUsername(ctx, username)
	if err != nil || row == nil {
		return connector.Identity{}, false, err
	}

	ident, err = c.identityFromRow(row)
	if err != nil {
		return connector.Identity{}, false, err
	}

	cryptPassword, ok := row[c.UserQuery.PasswordColumn].(string)
	if !ok {
		err = fmt.Errorf("sql: PasswordColumn %s must be a string",
			c.UserQuery.PasswordColumn)
		return connector.Identity{}, false, err
	}

	newPassword, err := c.passlib.Verify(password, cryptPassword)
	if err != nil {
		if c.Hooks.OnFailure != "" {
			_, err := c.db.NamedExecContext(ctx, c.Hooks.OnFailure,
				map[string]interface{}{
					"userid": ident.UserID,
				})
			if err != nil {
				c.logger.Warnf("sql: unable to execute failure hook: %v", err)
			}
		}

		c.logger.Warnf("sql: incorrect password for user %s", ident.UserID)
		return connector.Identity{}, false, nil
	}

	if newPassword != "" {
		if c.UserQuery.UpdatePassword == "" {
			c.logger.Warnf("sql: weak password hash for user %s", ident.UserID)
		} else {
			_, err = c.db.NamedExecContext(ctx, c.UserQuery.UpdatePassword,
				map[string]interface{}{
					"userid":   ident.UserID,
					"password": newPassword,
				})

			if err != nil {
				c.logger.Warnf("sql: unable to update weak password hash for user %s: %v", ident.UserID, err)
			} else {
				c.logger.Infof("sql: updating password hash for user %s", ident.UserID)
			}
		}
	}

	if c.Hooks.OnSuccess != "" {
		_, err := c.db.NamedExecContext(ctx, c.Hooks.OnSuccess,
			map[string]interface{}{
				"userid": ident.UserID,
			})
		if err != nil {
			c.logger.Warnf("sql: unable to execute success hook: %v", err)
		}
	}

	if s.Groups {
		groups, err := c.groups(ctx, ident.UserID)
		if err != nil {
			return connector.Identity{}, false,
				fmt.Errorf("sql: failed to query groups: %v", err)
		}
		ident.Groups = groups
	}

	if s.OfflineAccess {
		refresh := sqlRefreshData{
			UserID: ident.UserID,
		}

		if ident.ConnectorData, err = json.Marshal(refresh); err != nil {
			return connector.Identity{}, false,
				fmt.Errorf("sql: failed to marshal refresh data: %v", err)
		}
	}

	return ident, true, nil
}

func (c *sqlConnector) groups(ctx context.Context, userID string) ([]string,
	error) {
	if c.UserGroupsQuery.QueryByUserID == "" {
		c.logger.Warnf("sql: asked for groups but no UserGroupsQuery configured")
		return []string{}, nil
	}

	rows, err := c.db.NamedQueryContext(ctx, c.UserGroupsQuery.QueryByUserID,
		map[string]interface{}{
			"userid": userID,
		})
	if err != nil {
		return nil, fmt.Errorf("sql: failed to query groups: %v", err)
	}
	defer rows.Close()

	result := []string{}
	for rows.Next() {
		row := map[string]interface{}{}

		err = rows.MapScan(row)
		if err != nil {
			return nil, fmt.Errorf("sql: failed to read groups: %v", err)
		}

		name, ok := row[c.UserGroupsQuery.NameColumn].(string)
		if !ok {
			return nil,
				fmt.Errorf("sql: groups query returned non-string results")
		}

		result = append(result, name)
	}

	return result, nil
}

func (c *sqlConnector) Refresh(ctx context.Context, s connector.Scopes,
	ident connector.Identity) (newIdent connector.Identity, err error) {
	var refreshData sqlRefreshData
	if err := json.Unmarshal(ident.ConnectorData, &refreshData); err != nil {
		return ident,
			fmt.Errorf("sql: failed to unmarshal internal data: %v", err)
	}

	rows, err := c.db.NamedQueryContext(ctx, c.UserQuery.QueryByID,
		map[string]interface{}{
			"userid": refreshData.UserID,
		})
	if err != nil {
		return connector.Identity{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err == nil {
			err = fmt.Errorf("sql: user %q not found during refresh",
				refreshData.UserID)
		} else {
			err = rows.Err()
		}

		return connector.Identity{}, err
	}

	row := map[string]interface{}{}

	err = rows.MapScan(row)
	if err != nil {
		return connector.Identity{}, err
	}

	newIdent, err = c.identityFromRow(row)
	if err != nil {
		return connector.Identity{}, err
	}
	newIdent.ConnectorData = ident.ConnectorData

	if s.Groups {
		groups, err := c.groups(ctx, newIdent.UserID)
		if err != nil {
			return connector.Identity{},
				fmt.Errorf("sql: failed to query groups: %v", err)
		}
		newIdent.Groups = groups
	}

	return newIdent, nil
}

func (c *sqlConnector) Prompt() string {
	return c.UsernamePrompt
}

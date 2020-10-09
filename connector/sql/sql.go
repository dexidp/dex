// Package sql implements authentication using SQL queries.
package sql

import (
	"context"
	"encoding/json"
	"fmt"

	// Crypt
	"golang.org/x/crypto/bcrypt"

	// Database drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/jmoiron/sqlx"

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
	UsernamePrompt string `json:"usernamePrompt"`

	// User query, e.g.
	//
	//   SELECT id, username, email, name, password
	//   FROM Users
	//   WHERE username=:username OR email=:username
	//
	// The identity column must be an integer or a string.  The other
	// columns must all be strings.
	//
	// The password column is expected to be a crypt password.
	//
	UserQuery UserQuery `json:"userQuery"`

	// User groups query, e.g.
	//
	//   SELECT name
	//   FROM Groups INNER JOIN UserGroups
	//   ON Groups.id=UserGroups.group
	//   WHERE UserGroups.user=:userid
	//
	UserGroupsQuery UserGroupsQuery `json:"userGroupsQuery"`
}

type UserQuery struct {
	// The actual SQL
	QueryByName string `json:"queryByUsername"`
	QueryByID   string `json:"queryById"`

	// The names of various columns
	IDColumn       string `json:"idColumn"`
	UsernameColumn string `json:"usernameColumn"`
	EmailColumn    string `json:"emailColumn"`
	NameColumn     string `json:"nameColumn"`
	PasswordColumn string `json:"password"`
}

type UserGroupsQuery struct {
	// The actual SQL
	QueryByUserID string `json:"queryByUserId"`

	// Column names in the result
	NameColumn string `json:"nameColumn"`
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
	return &sqlConnector{*c, db, logger}, nil
}

type sqlConnector struct {
	Config

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
		ident.PreferredUsername, ok = row[c.UserQuery.UsernameColumn].(string)
		if !ok {
			err = fmt.Errorf("sql: UsernameColumn %s must be a string",
				c.UserQuery.UsernameColumn)
			return connector.Identity{}, err
		}
	}

	if c.UserQuery.EmailColumn != "" {
		ident.Email, ok = row[c.UserQuery.EmailColumn].(string)
		if !ok {
			err = fmt.Errorf("sql: EmailColumn %s must be a string",
				c.UserQuery.EmailColumn)
			return connector.Identity{}, err
		}
	}

	if c.UserQuery.NameColumn != "" {
		ident.Username, ok = row[c.UserQuery.NameColumn].(string)
		if !ok {
			err = fmt.Errorf("sql: NameColumn %s must be a string",
				c.UserQuery.NameColumn)
			return connector.Identity{}, err
		}
	}

	return ident, nil
}

func (c *sqlConnector) Login(ctx context.Context, s connector.Scopes,
	username, password string) (ident connector.Identity,
	validPass bool, err error) {

	rows, err := c.db.NamedQueryContext(ctx, c.UserQuery.QueryByName,
		map[string]interface{}{
			"username": username,
		})
	if err != nil {
		return connector.Identity{}, false, err
	}
	defer rows.Close()

	if !rows.Next() {
		err = rows.Err()
		return connector.Identity{}, false, err
	}

	row := map[string]interface{}{}

	err = rows.MapScan(row)
	if err != nil {
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

	if !c.validatePassword(password, cryptPassword) {
		c.logger.Warnf("sql: incorrect password for user %s", ident.UserID)
		return connector.Identity{}, false, nil
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

func (c *sqlConnector) validatePassword(password, cryptPassword string) bool {
	// Extract the algorithm identifier from cryptPassword
	pwLen := len(cryptPassword)
	if pwLen < 3 || cryptPassword[0] != '$' {
		c.logger.Warnf("sql: password missing algorithm identifier")
		return false
	}

	algEnd := 1
	for algEnd < pwLen && cryptPassword[algEnd] != '$' {
		algEnd += 1
	}

	if algEnd >= pwLen {
		c.logger.Warnf("sql: password has unterminated algorithm identifier")
		return false
	}

	algorithm := cryptPassword[1:algEnd]

	switch algorithm {
	case "2", "2a", "2b", "2x", "2y": // bcrypt
		return bcrypt.CompareHashAndPassword([]byte(cryptPassword),
			[]byte(password)) == nil
	default:
		c.logger.Warnf("sql: unsupported password algorithm $%s$", algorithm)
		return false
	}
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

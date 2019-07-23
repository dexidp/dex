// Package sql provides SQL implementations of the storage interface.
package sql

import (
	"database/sql"
	"regexp"
	"time"

	// import third party drivers
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/dexidp/dex/pkg/log"
)

// flavor represents a specific SQL implementation, and is used to translate query strings
// between different drivers. Flavors shouldn't aim to translate all possible SQL statements,
// only the specific queries used by the SQL storages.
type flavor struct {
	queryReplacers []replacer

	// Optional function to create and finish a transaction.
	executeTx func(db *sql.DB, fn func(*sql.Tx) error) error

	// Does the flavor support timezones?
	supportsTimezones bool
}

// A regexp with a replacement string.
type replacer struct {
	re   *regexp.Regexp
	with string
}

// Match a postgres query binds. E.g. "$1", "$12", etc.
var bindRegexp = regexp.MustCompile(`\$\d+`)

func matchLiteral(s string) *regexp.Regexp {
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(s) + `\b`)
}

var (
	// The "github.com/lib/pq" driver is the default flavor. All others are
	// translations of this.
	flavorPostgres = flavor{
		// The default behavior for Postgres transactions is consistent reads, not consistent writes.
		// For each transaction opened, ensure it has the correct isolation level.
		//
		// See: https://www.postgresql.org/docs/9.3/static/sql-set-transaction.html
		//
		// NOTE(ericchiang): For some reason using `SET SESSION CHARACTERISTICS AS TRANSACTION` at a
		// session level didn't work for some edge cases. Might be something worth exploring.
		executeTx: func(db *sql.DB, fn func(sqlTx *sql.Tx) error) error {
			tx, err := db.Begin()
			if err != nil {
				return err
			}
			defer tx.Rollback()

			if _, err := tx.Exec(`SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;`); err != nil {
				return err
			}
			if err := fn(tx); err != nil {
				return err
			}
			return tx.Commit()
		},

		supportsTimezones: true,
	}

	flavorSQLite3 = flavor{
		queryReplacers: []replacer{
			{bindRegexp, "?"},
			// Translate for booleans to integers.
			{matchLiteral("true"), "1"},
			{matchLiteral("false"), "0"},
			{matchLiteral("boolean"), "integer"},
			// Translate other types.
			{matchLiteral("bytea"), "blob"},
			{matchLiteral("timestamptz"), "timestamp"},
			// SQLite doesn't have a "now()" method, replace with "date('now')"
			{regexp.MustCompile(`\bnow\(\)`), "date('now')"},
		},
	}

	flavorMySQL = flavor{
		queryReplacers: []replacer{
			{bindRegexp, "?"},
			// Translate types.
			{matchLiteral("bytea"), "blob"},
			{matchLiteral("timestamptz"), "datetime(3)"},
			// MySQL doesn't support indicies on text fields w/o
			// specifying key length. Use varchar instead (767 byte
			// is the max key length for InnoDB with 4k pages).
			// For compound indexes (with two keys) even less.
			{matchLiteral("text"), "varchar(384)"},
			// Quote keywords and reserved words used as identifiers.
			{regexp.MustCompile(`\b(keys)\b`), "`$1`"},
			// Change default timestamp to fit datetime.
			{regexp.MustCompile(`0001-01-01 00:00:00 UTC`), "1000-01-01 00:00:00"},
		},
	}
)

func (f flavor) translate(query string) string {
	// TODO(ericchiang): Heavy cashing.
	for _, r := range f.queryReplacers {
		query = r.re.ReplaceAllString(query, r.with)
	}
	return query
}

// translateArgs translates query parameters that may be unique to
// a specific SQL flavor. For example, standardizing "time.Time"
// types to UTC for clients that don't provide timezone support.
func (c *conn) translateArgs(args []interface{}) []interface{} {
	if c.flavor.supportsTimezones {
		return args
	}

	for i, arg := range args {
		if t, ok := arg.(time.Time); ok {
			args[i] = t.UTC()
		}
	}
	return args
}

// conn is the main database connection.
type conn struct {
	db                 *sql.DB
	flavor             flavor
	logger             log.Logger
	alreadyExistsCheck func(err error) bool
}

func (c *conn) Close() error {
	return c.db.Close()
}

// conn implements the same method signatures as encoding/sql.DB.

func (c *conn) Exec(query string, args ...interface{}) (sql.Result, error) {
	query = c.flavor.translate(query)
	return c.db.Exec(query, c.translateArgs(args)...)
}

func (c *conn) Query(query string, args ...interface{}) (*sql.Rows, error) {
	query = c.flavor.translate(query)
	return c.db.Query(query, c.translateArgs(args)...)
}

func (c *conn) QueryRow(query string, args ...interface{}) *sql.Row {
	query = c.flavor.translate(query)
	return c.db.QueryRow(query, c.translateArgs(args)...)
}

// ExecTx runs a method which operates on a transaction.
func (c *conn) ExecTx(fn func(tx *trans) error) error {
	if c.flavor.executeTx != nil {
		return c.flavor.executeTx(c.db, func(sqlTx *sql.Tx) error {
			return fn(&trans{sqlTx, c})
		})
	}

	sqlTx, err := c.db.Begin()
	if err != nil {
		return err
	}
	if err := fn(&trans{sqlTx, c}); err != nil {
		sqlTx.Rollback()
		return err
	}
	return sqlTx.Commit()
}

type trans struct {
	tx *sql.Tx
	c  *conn
}

// trans implements the same method signatures as encoding/sql.Tx.

func (t *trans) Exec(query string, args ...interface{}) (sql.Result, error) {
	query = t.c.flavor.translate(query)
	return t.tx.Exec(query, t.c.translateArgs(args)...)
}

func (t *trans) Query(query string, args ...interface{}) (*sql.Rows, error) {
	query = t.c.flavor.translate(query)
	return t.tx.Query(query, t.c.translateArgs(args)...)
}

func (t *trans) QueryRow(query string, args ...interface{}) *sql.Row {
	query = t.c.flavor.translate(query)
	return t.tx.QueryRow(query, t.c.translateArgs(args)...)
}

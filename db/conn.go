package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"
	_ "github.com/lib/pq"

	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/repo"
)

type table struct {
	name    string
	model   interface{}
	autoinc bool
	pkey    []string

	// unique are non-primary key fields which should have uniqueness constraints.
	unique []string
}

var (
	tables []table
)

func register(t table) {
	tables = append(tables, t)
}

type Config struct {
	// Connection string in the format: <driver>://<username>:<password>@<host>:<port>/<database>
	DSN string
	// The maximum number of open connections to the database. The default is 0 (unlimited).
	// For more details see: http://golang.org/pkg/database/sql/#DB.SetMaxOpenConns
	MaxOpenConnections int
	// The maximum number of connections in the idle connection pool. The default is 0 (unlimited).
	// For more details see: http://golang.org/pkg/database/sql/#DB.SetMaxIdleConns
	MaxIdleConnections int
}

func NewConnection(cfg Config) (*gorp.DbMap, error) {
	if !strings.HasPrefix(cfg.DSN, "postgres://") {
		return nil, errors.New("unrecognized database driver")
	}

	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(cfg.MaxIdleConnections)
	db.SetMaxOpenConns(cfg.MaxOpenConnections)

	dbm := gorp.DbMap{
		Db:      db,
		Dialect: gorp.PostgresDialect{},
	}

	for _, t := range tables {
		tm := dbm.AddTableWithName(t.model, t.name).SetKeys(t.autoinc, t.pkey...)
		for _, unique := range t.unique {
			cm := tm.ColMap(unique)
			if cm == nil {
				return nil, fmt.Errorf("no such column: %q", unique)
			}
			cm.SetUnique(true)
		}
	}

	return &dbm, nil
}

func TransactionFactory(conn *gorp.DbMap) repo.TransactionFactory {
	return func() (repo.Transaction, error) {
		return conn.Begin()
	}
}

func rollback(tx *gorp.Transaction) {
	err := tx.Rollback()
	if err != nil {
		log.Errorf("unable to rollback: %v", err)
	}
}

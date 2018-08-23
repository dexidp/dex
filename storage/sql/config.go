package sql

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/lib/pq"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"

	"github.com/dexidp/dex/storage"
)

const (
	// postgres error codes
	pgErrUniqueViolation = "23505" // unique_violation
)

// SQLite3 options for creating an SQL db.
type SQLite3 struct {
	// File to
	File string `json:"file"`
}

// Open creates a new storage implementation backed by SQLite3
func (s *SQLite3) Open(logger logrus.FieldLogger) (storage.Storage, error) {
	conn, err := s.open(logger)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (s *SQLite3) open(logger logrus.FieldLogger) (*conn, error) {
	db, err := sql.Open("sqlite3", s.File)
	if err != nil {
		return nil, err
	}
	if s.File == ":memory:" {
		// sqlite3 uses file locks to coordinate concurrent access. In memory
		// doesn't support this, so limit the number of connections to 1.
		db.SetMaxOpenConns(1)
	}

	errCheck := func(err error) bool {
		sqlErr, ok := err.(sqlite3.Error)
		if !ok {
			return false
		}
		return sqlErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey
	}

	c := &conn{db, flavorSQLite3, logger, errCheck}
	if _, err := c.migrate(); err != nil {
		return nil, fmt.Errorf("failed to perform migrations: %v", err)
	}
	return c, nil
}

const (
	sslDisable    = "disable"
	sslRequire    = "require"
	sslVerifyCA   = "verify-ca"
	sslVerifyFull = "verify-full"
)

// PostgresSSL represents SSL options for Postgres databases.
type PostgresSSL struct {
	Mode   string
	CAFile string
	// Files for client auth.
	KeyFile  string
	CertFile string
}

// Postgres options for creating an SQL db.
type Postgres struct {
	Database string
	User     string
	Password string
	Host     string
	Port     uint16

	SSL PostgresSSL `json:"ssl" yaml:"ssl"`

	ConnectionTimeout int // Seconds
}

// Open creates a new storage implementation backed by Postgres.
func (p *Postgres) Open(logger logrus.FieldLogger) (storage.Storage, error) {
	conn, err := p.open(logger, p.CreateDataSourceName())
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// CreateDataSourceName takes the configuration provided via the Postgres
// struct to create a data-source name that Go's database/sql package can
// make use of.
func (p *Postgres) CreateDataSourceName() string {
	parameters := []string{
		"connect_timeout=" + strconv.Itoa(p.ConnectionTimeout),
	}

	if p.Host != "" {
		parameters = append(parameters, "host='"+p.Host+"'")
	}

	if p.Port != 0 {
		parameters = append(parameters, "port='"+strconv.Itoa(int(p.Port))+"'")
	}

	if p.User != "" {
		parameters = append(parameters, "user='"+p.User+"'")
	}

	if p.Password != "" {
		parameters = append(parameters, "password='"+p.Password+"'")
	}

	if p.Database != "" {
		parameters = append(parameters, "dbname='"+p.Database+"'")
	}

	if p.SSL.Mode == "" {
		// Assume the strictest mode if unspecified.
		p.SSL.Mode = sslVerifyFull
	}

	parameters = append(parameters, "sslmode='"+p.SSL.Mode+"'")

	if p.SSL.Mode != "disable" {
		parameters = append(parameters, "sslcert='"+p.SSL.CertFile+"'")
		parameters = append(parameters, "sslkey='"+p.SSL.KeyFile+"'")
		parameters = append(parameters, "sslrootcert='"+p.SSL.CAFile+"'")
	}

	return strings.Join(parameters, " ")
}

func (p *Postgres) open(logger logrus.FieldLogger, dataSourceName string) (*conn, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}

	errCheck := func(err error) bool {
		sqlErr, ok := err.(*pq.Error)
		if !ok {
			return false
		}
		return sqlErr.Code == pgErrUniqueViolation
	}

	c := &conn{db, flavorPostgres, logger, errCheck}
	if _, err := c.migrate(); err != nil {
		return nil, fmt.Errorf("failed to perform migrations: %v", err)
	}
	return c, nil
}

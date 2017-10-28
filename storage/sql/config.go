package sql

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"

	"github.com/coreos/dex/storage"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

const (
	// postgres error codes
	pgErrUniqueViolation = "23505" // unique_violation
)

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

	SSL PostgresSSL `json:"ssl" yaml:"ssl"`

	ConnectionTimeout int // Seconds
}

// Open creates a new storage implementation backed by Postgres.
func (p *Postgres) Open(logger logrus.FieldLogger) (storage.Storage, error) {
	conn, err := p.open(logger)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (p *Postgres) open(logger logrus.FieldLogger) (*conn, error) {
	v := url.Values{}
	set := func(key, val string) {
		if val != "" {
			v.Set(key, val)
		}
	}
	set("connect_timeout", strconv.Itoa(p.ConnectionTimeout))
	set("sslkey", p.SSL.KeyFile)
	set("sslcert", p.SSL.CertFile)
	set("sslrootcert", p.SSL.CAFile)
	if p.SSL.Mode == "" {
		// Assume the strictest mode if unspecified.
		p.SSL.Mode = sslVerifyFull
	}
	set("sslmode", p.SSL.Mode)

	u := url.URL{
		Scheme:   "postgres",
		Host:     p.Host,
		Path:     "/" + p.Database,
		RawQuery: v.Encode(),
	}

	if p.User != "" {
		if p.Password != "" {
			u.User = url.UserPassword(p.User, p.Password)
		} else {
			u.User = url.User(p.User)
		}
	}
	db, err := sql.Open("postgres", u.String())
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

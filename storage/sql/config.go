package sql

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/url"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/dex/storage"
	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	sqlite3 "github.com/mattn/go-sqlite3"
)

const (
	// postgres error codes
	pgErrUniqueViolation = "23505" // unique_violation
)

const (
	// MySQL error codes
	mysqlErrDupEntry            = 1062
	mysqlErrDupEntryWithKeyName = 1586
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
	// postgres SSL modes
	pgSSLDisable    = "disable"
	pgSSLRequire    = "require"
	pgSSLVerifyCA   = "verify-ca"
	pgSSLVerifyFull = "verify-full"
)

const (
	// MySQL SSL modes
	mysqlSSLTrue       = "true"
	mysqlSSLFalse      = "false"
	mysqlSSLSkipVerify = "skip-verify"
	mysqlSSLCustom     = "custom"
)

// NetworkDB contains options common to SQL databases accessed over network.
type NetworkDB struct {
	Database string
	User     string
	Password string
	Host     string

	ConnectionTimeout int // Seconds
}

// SSL represents SSL options for network databases.
type SSL struct {
	Mode   string
	CAFile string
	// Files for client auth.
	KeyFile  string
	CertFile string
}

// Postgres options for creating an SQL db.
type Postgres struct {
	NetworkDB

	SSL SSL `json:"ssl" yaml:"ssl"`
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
		p.SSL.Mode = pgSSLVerifyFull
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

// MySQL options for creating a MySQL db.
type MySQL struct {
	NetworkDB

	SSL SSL `json:"ssl" yaml:"ssl"`

	// TODO(pborzenkov): used by tests to reduce lock wait timeout. Should
	// we make it exported and allow users to provide arbitrary params?
	params map[string]string
}

// Open creates a new storage implementation backed by MySQL.
func (s *MySQL) Open(logger logrus.FieldLogger) (storage.Storage, error) {
	conn, err := s.open(logger)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (s *MySQL) open(logger logrus.FieldLogger) (*conn, error) {
	cfg := mysql.Config{
		User:   s.User,
		Passwd: s.Password,
		DBName: s.Database,

		Timeout: time.Second * time.Duration(s.ConnectionTimeout),

		ParseTime: true,
		Params: map[string]string{
			"tx_isolation": "'SERIALIZABLE'",
		},
	}
	if s.Host != "" {
		if s.Host[0] != '/' {
			cfg.Net = "tcp"
			cfg.Addr = s.Host
		} else {
			cfg.Net = "unix"
			cfg.Addr = s.Host
		}
	}
	if s.SSL.CAFile != "" || s.SSL.CertFile != "" || s.SSL.KeyFile != "" {
		if err := s.makeTLSConfig(); err != nil {
			return nil, fmt.Errorf("failed to make TLS config: %v", err)
		}
		cfg.TLSConfig = mysqlSSLCustom
	} else {
		cfg.TLSConfig = s.SSL.Mode
	}
	for k, v := range s.params {
		cfg.Params[k] = v
	}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}

	errCheck := func(err error) bool {
		sqlErr, ok := err.(*mysql.MySQLError)
		if !ok {
			return false
		}
		return sqlErr.Number == mysqlErrDupEntry ||
			sqlErr.Number == mysqlErrDupEntryWithKeyName
	}

	c := &conn{db, flavorMySQL, logger, errCheck}
	if _, err := c.migrate(); err != nil {
		return nil, fmt.Errorf("failed to perform migrations: %v", err)
	}
	return c, nil
}

func (s *MySQL) makeTLSConfig() error {
	cfg := &tls.Config{}
	if s.SSL.CAFile != "" {
		rootCertPool := x509.NewCertPool()
		pem, err := ioutil.ReadFile(s.SSL.CAFile)
		if err != nil {
			return err
		}
		if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
			return fmt.Errorf("failed to append PEM")
		}
		cfg.RootCAs = rootCertPool
	}
	if s.SSL.CertFile != "" && s.SSL.KeyFile != "" {
		clientCert := make([]tls.Certificate, 0, 1)
		certs, err := tls.LoadX509KeyPair(s.SSL.CertFile, s.SSL.KeyFile)
		if err != nil {
			return err
		}
		clientCert = append(clientCert, certs)
		cfg.Certificates = clientCert
	}

	mysql.RegisterTLSConfig(mysqlSSLCustom, cfg)
	return nil
}

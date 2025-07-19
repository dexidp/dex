package sql

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql/driver"
	"fmt"
	"log/slog"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/XSAM/otelsql"
	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"

	"github.com/dexidp/dex/storage"
)

const (
	// postgres error codes
	pgErrUniqueViolation = "23505" // unique_violation
)

const (
	// MySQL error codes
	mysqlErrDupEntry            = 1062
	mysqlErrDupEntryWithKeyName = 1586
	mysqlErrUnknownSysVar       = 1193
)

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
	Port     uint16

	ConnectionTimeout int // Seconds

	// database/sql tunables, see
	// https://golang.org/pkg/database/sql/#DB.SetConnMaxLifetime and below
	// Note: defaults will be set if these are 0
	MaxOpenConns    int // default: 5
	MaxIdleConns    int // default: 5
	ConnMaxLifetime int // Seconds, default: not set
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
func (p *Postgres) Open(logger *slog.Logger) (storage.Storage, error) {
	conn, err := p.open(logger)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

var strEsc = regexp.MustCompile(`([\\'])`)

func dataSourceStr(str string) string {
	return "'" + strEsc.ReplaceAllString(str, `\$1`) + "'"
}

// createDataSourceName takes the configuration provided via the Postgres
// struct to create a data-source name that Go's database/sql package can
// make use of.
func (p *Postgres) createDataSourceName() string {
	parameters := []string{}

	addParam := func(key, val string) {
		parameters = append(parameters, fmt.Sprintf("%s=%s", key, val))
	}

	addParam("connect_timeout", strconv.Itoa(p.ConnectionTimeout))

	// detect host:port for backwards-compatibility
	host, port, err := net.SplitHostPort(p.Host)
	if err != nil {
		// not host:port, probably unix socket or bare address

		host = p.Host

		if p.Port != 0 {
			port = strconv.Itoa(int(p.Port))
		}
	}

	if host != "" {
		addParam("host", dataSourceStr(host))
	}

	if port != "" {
		addParam("port", port)
	}

	if p.User != "" {
		addParam("user", dataSourceStr(p.User))
	}

	if p.Password != "" {
		addParam("password", dataSourceStr(p.Password))
	}

	if p.Database != "" {
		addParam("dbname", dataSourceStr(p.Database))
	}

	if p.SSL.Mode == "" {
		// Assume the strictest mode if unspecified.
		addParam("sslmode", dataSourceStr(pgSSLVerifyFull))
	} else {
		addParam("sslmode", dataSourceStr(p.SSL.Mode))
	}

	if p.SSL.CAFile != "" {
		addParam("sslrootcert", dataSourceStr(p.SSL.CAFile))
	}

	if p.SSL.CertFile != "" {
		addParam("sslcert", dataSourceStr(p.SSL.CertFile))
	}

	if p.SSL.KeyFile != "" {
		addParam("sslkey", dataSourceStr(p.SSL.KeyFile))
	}

	return strings.Join(parameters, " ")
}

// Add this helper function to extract the operation name
func extractOperationName(query string) string {
	upperQuery := strings.ToUpper(query)
	// Regex to match the operation keyword at the start of the query (after optional whitespace)
	// Captures common SQL operations; extend the list as needed for your use case
	re := regexp.MustCompile(`^\s*(SELECT|INSERT|UPDATE|DELETE|REPLACE|UPSERT|MERGE|CALL|EXPLAIN|CREATE|ALTER|DROP|TRUNCATE|RENAME|SET|USE|GRANT|REVOKE)\b`)
	matches := re.FindStringSubmatch(upperQuery)
	if len(matches) == 2 {
		// Find indices to extract from original query (preserves case)
		idx := re.FindStringSubmatchIndex(upperQuery)
		if len(idx) >= 4 {
			start := idx[2]
			end := idx[3]
			return query[start:end]
		}
	}
	return ""
}

// Existing helper function to extract the table name (unchanged, but included for completeness)
func extractTableName(query string) string {
	upperQuery := strings.ToUpper(query)
	// Regex to match the table after common SQL keywords (uppercase for case-insensitive matching via ToUpper)
	// Captures quoted or unquoted table names, possibly with schema (e.g., public.users)
	re := regexp.MustCompile(`\b(FROM|INTO|UPDATE|DELETE\s+FROM)\s+(["]?[\w.]+\b["]?)\b`)
	matches := re.FindStringSubmatch(upperQuery)
	if len(matches) == 3 {
		// Find indices to extract from original query (preserves case)
		idx := re.FindStringSubmatchIndex(upperQuery)
		if len(idx) >= 6 {
			start := idx[4]
			end := idx[5]
			table := query[start:end]
			// Remove any surrounding quotes
			table = strings.Trim(table, "\"`")
			return table
		}
	}
	return ""
}

func (p *Postgres) open(logger *slog.Logger) (*conn, error) {
	dataSourceName := p.createDataSourceName()
	db, err := otelsql.Open("postgres", dataSourceName, otelsql.WithAttributes(
		semconv.DBSystemNamePostgreSQL,
		semconv.DBNamespace(p.Database),
		semconv.NetworkPeerPort(int(p.Port)),
	),
		otelsql.WithAttributesGetter(func(ctx context.Context, method otelsql.Method, query string, args []driver.NamedValue) []attribute.KeyValue {
			attrs := []attribute.KeyValue{}
			operation := extractOperationName(query)
			if operation != "" {
				attrs = append(attrs, semconv.DBOperationName(operation))
			}
			table := extractTableName(query)
			if table != "" {
				attrs = append(attrs, semconv.DBCollectionName(table))
			}
			if len(attrs) > 0 {
				return attrs
			}
			// If nothing extracted, return empty (or fallback to other attributes if needed)
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}
	err = otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(
		semconv.DBSystemNamePostgreSQL,
		semconv.DBNamespace(p.Database),
		semconv.NetworkPeerPort(int(p.Port)),
	),
		otelsql.WithAttributesGetter(func(ctx context.Context, method otelsql.Method, query string, args []driver.NamedValue) []attribute.KeyValue {
			attrs := []attribute.KeyValue{}
			operation := extractOperationName(query)
			if operation != "" {
				attrs = append(attrs, semconv.DBOperationName(operation))
			}
			table := extractTableName(query)
			if table != "" {
				attrs = append(attrs, semconv.DBCollectionName(table))
			}
			if len(attrs) > 0 {
				return attrs
			}
			// If nothing extracted, return empty (or fallback to other attributes if needed)
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}

	// set database/sql tunables if configured
	if p.ConnMaxLifetime != 0 {
		db.SetConnMaxLifetime(time.Duration(p.ConnMaxLifetime) * time.Second)
	}

	if p.MaxIdleConns == 0 {
		db.SetMaxIdleConns(5)
	} else {
		db.SetMaxIdleConns(p.MaxIdleConns)
	}

	if p.MaxOpenConns == 0 {
		db.SetMaxOpenConns(5)
	} else {
		db.SetMaxOpenConns(p.MaxOpenConns)
	}

	errCheck := func(err error) bool {
		sqlErr, ok := err.(*pq.Error)
		if !ok {
			return false
		}
		return sqlErr.Code == pgErrUniqueViolation
	}

	c := &conn{db, &flavorPostgres, logger, errCheck}
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
func (s *MySQL) Open(logger *slog.Logger) (storage.Storage, error) {
	conn, err := s.open(logger)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (s *MySQL) open(logger *slog.Logger) (*conn, error) {
	cfg := mysql.Config{
		User:                 s.User,
		Passwd:               s.Password,
		DBName:               s.Database,
		AllowNativePasswords: true,

		Timeout: time.Second * time.Duration(s.ConnectionTimeout),

		ParseTime: true,
		Params: map[string]string{
			"transaction_isolation": "'SERIALIZABLE'",
		},
	}
	if s.Host != "" {
		if s.Host[0] != '/' {
			cfg.Net = "tcp"
			cfg.Addr = s.Host

			if s.Port != 0 {
				cfg.Addr = net.JoinHostPort(s.Host, strconv.Itoa(int(s.Port)))
			}
		} else {
			cfg.Net = "unix"
			cfg.Addr = s.Host
		}
	}

	switch {
	case s.SSL.CAFile != "" || s.SSL.CertFile != "" || s.SSL.KeyFile != "":
		if err := s.makeTLSConfig(); err != nil {
			return nil, fmt.Errorf("failed to make TLS config: %v", err)
		}
		cfg.TLSConfig = mysqlSSLCustom
	case s.SSL.Mode == "":
		cfg.TLSConfig = mysqlSSLTrue
	default:
		cfg.TLSConfig = s.SSL.Mode
	}

	for k, v := range s.params {
		cfg.Params[k] = v
	}
	db, err := otelsql.Open("mysql", cfg.FormatDSN(), otelsql.WithAttributes(
		semconv.DBSystemNameMySQL,
	))
	if err != nil {
		return nil, err
	}
	err = otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(
		semconv.DBSystemNameMySQL,
	))
	if err != nil {
		return nil, err
	}

	if s.MaxIdleConns == 0 {
		/*Override default behaviour to fix https://github.com/dexidp/dex/issues/1608*/
		db.SetMaxIdleConns(0)
	} else {
		db.SetMaxIdleConns(s.MaxIdleConns)
	}

	err = db.Ping()
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == mysqlErrUnknownSysVar {
			logger.Info("reconnecting with MySQL pre-5.7.20 compatibility mode")

			// MySQL 5.7.20 introduced transaction_isolation and deprecated tx_isolation.
			// MySQL 8.0 doesn't have tx_isolation at all.
			// https://dev.mysql.com/doc/refman/5.7/en/server-system-variables.html#sysvar_transaction_isolation
			delete(cfg.Params, "transaction_isolation")
			cfg.Params["tx_isolation"] = "'SERIALIZABLE'"

			db, err = otelsql.Open("mysql", cfg.FormatDSN(), otelsql.WithAttributes(
				semconv.DBSystemNameMySQL,
			))
			if err != nil {
				return nil, err
			}
			err = otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(
				semconv.DBSystemNameMySQL,
			))
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	errCheck := func(err error) bool {
		sqlErr, ok := err.(*mysql.MySQLError)
		if !ok {
			return false
		}
		return sqlErr.Number == mysqlErrDupEntry ||
			sqlErr.Number == mysqlErrDupEntryWithKeyName
	}

	c := &conn{db, &flavorMySQL, logger, errCheck}
	if _, err := c.migrate(); err != nil {
		return nil, fmt.Errorf("failed to perform migrations: %v", err)
	}
	return c, nil
}

func (s *MySQL) makeTLSConfig() error {
	cfg := &tls.Config{}
	if s.SSL.CAFile != "" {
		rootCertPool := x509.NewCertPool()
		pem, err := os.ReadFile(s.SSL.CAFile)
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

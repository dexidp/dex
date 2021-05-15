package ent

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	entSQL "entgo.io/ent/dialect/sql"

	// Register postgres driver.
	_ "github.com/lib/pq"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/ent/client"
	"github.com/dexidp/dex/storage/ent/db"
)

// nolint
const (
	// postgres SSL modes
	pgSSLDisable    = "disable"
	pgSSLRequire    = "require"
	pgSSLVerifyCA   = "verify-ca"
	pgSSLVerifyFull = "verify-full"
)

// Postgres options for creating an SQL db.
type Postgres struct {
	NetworkDB

	SSL SSL `json:"ssl"`
}

// Open always returns a new in sqlite3 storage.
func (p *Postgres) Open(logger log.Logger) (storage.Storage, error) {
	logger.Debug("experimental ent-based storage driver is enabled")
	drv, err := p.driver()
	if err != nil {
		return nil, err
	}

	databaseClient := client.NewDatabase(
		client.WithClient(db.NewClient(db.Driver(drv))),
		client.WithHasher(sha256.New),
		// The default behavior for Postgres transactions is consistent reads, not consistent writes.
		// For each transaction opened, ensure it has the correct isolation level.
		//
		// See: https://www.postgresql.org/docs/9.3/static/sql-set-transaction.html
		client.WithTxIsolationLevel(sql.LevelSerializable),
	)

	if err := databaseClient.Schema().Create(context.TODO()); err != nil {
		return nil, err
	}

	return databaseClient, nil
}

func (p *Postgres) driver() (*entSQL.Driver, error) {
	drv, err := entSQL.Open("postgres", p.dsn())
	if err != nil {
		return nil, err
	}

	// set database/sql tunables if configured
	if p.ConnMaxLifetime != 0 {
		drv.DB().SetConnMaxLifetime(time.Duration(p.ConnMaxLifetime) * time.Second)
	}

	if p.MaxIdleConns == 0 {
		drv.DB().SetMaxIdleConns(5)
	} else {
		drv.DB().SetMaxIdleConns(p.MaxIdleConns)
	}

	if p.MaxOpenConns == 0 {
		drv.DB().SetMaxOpenConns(5)
	} else {
		drv.DB().SetMaxOpenConns(p.MaxOpenConns)
	}

	return drv, nil
}

func (p *Postgres) dsn() string {
	// detect host:port for backwards-compatibility
	host, port, err := net.SplitHostPort(p.Host)
	if err != nil {
		// not host:port, probably unix socket or bare address
		host = p.Host
		if p.Port != 0 {
			port = strconv.Itoa(int(p.Port))
		}
	}

	var parameters []string
	addParam := func(key, val string) {
		parameters = append(parameters, fmt.Sprintf("%s=%s", key, val))
	}

	addParam("connect_timeout", strconv.Itoa(p.ConnectionTimeout))

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

var strEsc = regexp.MustCompile(`([\\'])`)

func dataSourceStr(str string) string {
	return "'" + strEsc.ReplaceAllString(str, `\$1`) + "'"
}

package db

import (
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/connector"
	dexdb "github.com/coreos/dex/db"
	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/refresh"
	"github.com/coreos/dex/repo"
	"github.com/coreos/dex/session"
	"github.com/coreos/dex/user"
	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oidc"
	"github.com/go-gorp/gorp"
	"github.com/jonboulle/clockwork"
)

const (
	DriverName string = "postgresql"
)

var (
	dbURL          *string
	dbMaxIdleConns *int
	dbMaxOpenConns *int
)

func init() {
	dexdb.Register(DriverName, &dexdb.RegisteredDriver{
		New:        newPostgresqlDriver,
		InitFlags:  initFlags,
		NewWithMap: newPostgresqlDriverWithMap,
	})
}

func initFlags(fs *flag.FlagSet) {
	//old flag names are used
	dbURL = fs.String("db-url", "", "DSN-formatted database connection string")
	dbMaxIdleConns = fs.Int("db-max-idle-conns", 0, "maximum number of connections in the idle connection pool")
	dbMaxOpenConns = fs.Int("db-max-open-conns", 0, "maximum number of open connections to the database")
}

func newPostgresqlDriver() (dexdb.Driver, error) {
	if *dbMaxIdleConns == 0 {
		log.Warning("Running with no limit on: database idle connections")
	}
	if *dbMaxOpenConns == 0 {
		log.Warning("Running with no limit on: database open connections")
	}

	config := Config{
		DSN:                *dbURL,
		MaxIdleConnections: *dbMaxIdleConns,
		MaxOpenConnections: *dbMaxOpenConns,
	}

	if config.DSN == "" {
		return nil, errors.New("missing database connection string")
	}

	dbc, err := NewConnection(config)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize database connection: %v", err)
	}
	p := &postgresqlDriver{
		conn: dbc,
	}
	return p, nil
}

func newPostgresqlDriverWithMap(cnf map[string]interface{}) (dexdb.Driver, error) {
	url := ""
	maxIdleConns := 0
	maxOpenConns := 0

	if u, ok := cnf["url"]; ok {
		url = u.(string)
	}

	if maxIdleConns == 0 {
		log.Warning("Running with no limit on: database idle connections")
	}
	if maxOpenConns == 0 {
		log.Warning("Running with no limit on: database open connections")
	}

	config := Config{
		DSN:                url,
		MaxIdleConnections: maxIdleConns,
		MaxOpenConnections: maxOpenConns,
	}

	if config.DSN == "" {
		return nil, errors.New("missing database connection string")
	}

	dbc, err := NewConnection(config)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize database connection: %v", err)
	}
	p := &postgresqlDriver{
		conn: dbc,
	}
	return p, nil
}

type postgresqlDriver struct {
	conn *gorp.DbMap
}

func (m *postgresqlDriver) Name() string {
	return DriverName
}

func (m *postgresqlDriver) DoesNeedGarbageCollecting() bool {
	return true
}

func (m *postgresqlDriver) NewConnectorConfigRepo() connector.ConnectorConfigRepo {
	return NewConnectorConfigRepo(m.conn)
}

func (m *postgresqlDriver) NewClientIdentityRepo() client.ClientIdentityRepo {
	return &clientIdentityRepo{dbMap: m.conn}
}

func (m *postgresqlDriver) NewPasswordInfoRepo() user.PasswordInfoRepo {
	return &passwordInfoRepo{dbMap: m.conn}
}

func (m *postgresqlDriver) NewSessionRepo() session.SessionRepo {
	return NewSessionRepo(m.conn)
}

func (m *postgresqlDriver) NewSessionKeyRepo() session.SessionKeyRepo {
	return NewSessionKeyRepo(m.conn)
}

func (m *postgresqlDriver) NewUserRepo() user.UserRepo {
	return &userRepo{dbMap: m.conn}
}

func (m *postgresqlDriver) NewRefreshTokenRepo() refresh.RefreshTokenRepo {
	return NewRefreshTokenRepo(m.conn)
}

func (m *postgresqlDriver) NewPrivateKeySetRepo(useOldFormatKeySecrets bool, keySecrets ...[]byte) (key.PrivateKeySetRepo, error) {
	if len(keySecrets) == 0 {
		return nil, errors.New("missing key secret")
	}

	kRepo, err := NewPrivateKeySetRepo(m.conn, useOldFormatKeySecrets, keySecrets...)
	if err != nil {
		return nil, fmt.Errorf("unable to create PrivateKeySetRepo: %v", err)
	}
	return kRepo, nil
}

func (m *postgresqlDriver) GetTransactionFactory() repo.TransactionFactory {
	return TransactionFactory(m.conn)
}

func (m *postgresqlDriver) DropTablesIfExists() error {
	return m.conn.DropTablesIfExists()
}

func (m *postgresqlDriver) DropMigrationsTable() error {
	return DropMigrationsTable(m.conn)
}

func (m *postgresqlDriver) MigrateToLatest() (int, error) {
	return MigrateToLatest(m.conn)
}

func (m *postgresqlDriver) NewGarbageCollector(interval time.Duration) dexdb.GarbageCollector {
	return NewGarbageCollector(m.conn, interval)
}

func (m *postgresqlDriver) NewUserRepoFromUsers(users []user.UserWithRemoteIdentities) (user.UserRepo, error) {
	return NewUserRepoFromUsers(m.conn, users)
}

func (m *postgresqlDriver) NewClientIdentityRepoFromClients(clients []oidc.ClientIdentity) (client.ClientIdentityRepo, error) {
	return NewClientIdentityRepoFromClients(m.conn, clients)
}

func (m *postgresqlDriver) NewSessionRepoWithClock(clock clockwork.Clock) session.SessionRepo {
	return NewSessionRepoWithClock(m.conn, clock)
}

func (m *postgresqlDriver) NewSessionKeyRepoWithClock(clock clockwork.Clock) session.SessionKeyRepo {
	return NewSessionKeyRepoWithClock(m.conn, clock)
}

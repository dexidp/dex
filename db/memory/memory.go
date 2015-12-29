package memory

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/db"
	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/refresh"
	"github.com/coreos/dex/repo"
	"github.com/coreos/dex/session"
	"github.com/coreos/dex/user"
	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oidc"
	"github.com/jonboulle/clockwork"
)

const (
	MemoryDriverName     string = "memory"
	MemoryConnectorsFlag string = "memory-connectors"
	MemoryClientsFlag    string = "memory-clients"
	MemoryUsersFlag      string = "memory-users"
)

var (
	connectors *string
	clients    *string
	users      *string
)

func init() {
	db.Register(MemoryDriverName, &db.RegisteredDriver{
		New:       newMemoryDriver,
		InitFlags: initFlags,
	})
}

func initFlags(fs *flag.FlagSet) {
	connectors = fs.String(MemoryConnectorsFlag, "./static/fixtures/connectors.json", "JSON file containg set of IDPC configs")
	clients = fs.String(MemoryClientsFlag, "./static/fixtures/clients.json", "json file containing set of clients")
	users = fs.String(MemoryUsersFlag, "./static/fixtures/users.json", "json file containing set of users")
}
func newMemoryDriver() (db.Driver, error) {
	m := &MemoryDB{}

	log.Warning("Running in-process without external database or key rotation")

	k, err := key.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}

	ks := key.NewPrivateKeySet([]*key.PrivateKey{k}, time.Now().Add(24*time.Hour))
	kRepo := key.NewPrivateKeySetRepo()
	if err = kRepo.Set(ks); err != nil {
		return nil, err
	}

	ccr, err := connector.NewConnectorConfigRepoFromFile(*connectors)
	if err != nil {
		return nil, err
	}
	cf, err := os.Open(*clients)
	if err != nil {
		return nil, fmt.Errorf("unable to read clients from file %s: %v", *clients, err)
	}
	defer cf.Close()
	ciRepo, err := client.NewClientIdentityRepoFromReader(cf)
	if err != nil {
		return nil, fmt.Errorf("unable to read client identities from file %s: %v", *clients, err)
	}
	userRepo, err := user.NewUserRepoFromFile(*users)
	if err != nil {
		return nil, fmt.Errorf("unable to read users from file: %v", err)
	}
	m.privateKeySetRepo = kRepo
	m.connectorConfigRepo = ccr
	m.clientIdentityRepo = ciRepo
	m.sessionRepo = session.NewSessionRepo()
	m.sessionKeyRepo = session.NewSessionKeyRepo()
	m.userRepo = userRepo
	m.passwordInfoRepo = user.NewPasswordInfoRepo()
	m.refreshTokenRepo = refresh.NewRefreshTokenRepo()
	return m, nil
}

type MemoryDB struct {
	connectorConfigRepo connector.ConnectorConfigRepo
	clientIdentityRepo  client.ClientIdentityRepo
	sessionRepo         session.SessionRepo
	sessionKeyRepo      session.SessionKeyRepo
	passwordInfoRepo    user.PasswordInfoRepo
	refreshTokenRepo    refresh.RefreshTokenRepo
	userRepo            user.UserRepo
	privateKeySetRepo   key.PrivateKeySetRepo
}

func (m *MemoryDB) Name() string {
	return MemoryDriverName
}

func (m *MemoryDB) DoesNeedGarbageCollecting() bool {
	return false
}

func (m *MemoryDB) NewConnectorConfigRepo() connector.ConnectorConfigRepo {
	return m.connectorConfigRepo
}

func (m *MemoryDB) NewClientIdentityRepo() client.ClientIdentityRepo {
	return m.clientIdentityRepo
}

func (m *MemoryDB) NewSessionRepo() session.SessionRepo {
	return m.sessionRepo
}
func (m *MemoryDB) NewPasswordInfoRepo() user.PasswordInfoRepo {
	return m.passwordInfoRepo
}

func (m *MemoryDB) NewSessionKeyRepo() session.SessionKeyRepo {
	return m.sessionKeyRepo
}

func (m *MemoryDB) NewPrivateKeySetRepo(useOldFormatKeySecrets bool, keySecrets ...[]byte) (key.PrivateKeySetRepo, error) {
	return m.privateKeySetRepo, nil
}

func (m *MemoryDB) GetTransactionFactory() repo.TransactionFactory {
	return repo.InMemTransactionFactory
}

func (m *MemoryDB) NewRefreshTokenRepo() refresh.RefreshTokenRepo {
	return m.refreshTokenRepo
}

func (m *MemoryDB) NewUserRepo() user.UserRepo {
	return m.userRepo
}

func (m *MemoryDB) DropTablesIfExists() error {
	return nil
}

func (m *MemoryDB) DropMigrationsTable() error {
	return nil
}

func (m *MemoryDB) MigrateToLatest() (int, error) {
	return 0, nil
}

func (m *MemoryDB) NewGarbageCollector(interval time.Duration) db.GarbageCollector {
	return nil
}

func (m *MemoryDB) NewUserRepoFromUsers(users []user.UserWithRemoteIdentities) (user.UserRepo, error) {
	return user.NewUserRepoFromUsers(users), nil
}

func (m *MemoryDB) NewClientIdentityRepoFromClients(clients []oidc.ClientIdentity) (client.ClientIdentityRepo, error) {
	return client.NewClientIdentityRepo(clients), nil
}

func (m *MemoryDB) NewSessionRepoWithClock(clock clockwork.Clock) session.SessionRepo {
	return session.NewSessionRepoWithClock(clock)
}

func (m *MemoryDB) NewSessionKeyRepoWithClock(clock clockwork.Clock) session.SessionKeyRepo {
	return session.NewSessionKeyRepoWithClock(clock)
}

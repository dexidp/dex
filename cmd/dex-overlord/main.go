package main

import (
	"expvar"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/coreos/go-oidc/key"
	"github.com/go-gorp/gorp"

	"github.com/coreos/dex/admin"
	clientmanager "github.com/coreos/dex/client/manager"
	"github.com/coreos/dex/db"
	pflag "github.com/coreos/dex/pkg/flag"
	"github.com/coreos/dex/pkg/log"
	ptime "github.com/coreos/dex/pkg/time"
	"github.com/coreos/dex/server"
	"github.com/coreos/dex/user/manager"
)

var version = "DEV"

func init() {
	expvar.NewString("dex.version").Set(version)
}

func main() {
	fs := flag.NewFlagSet("dex-overlord", flag.ExitOnError)

	keySecrets := pflag.NewBase64List(32)
	fs.Var(keySecrets, "key-secrets", "A comma-separated list of base64 encoded 32 byte strings used as symmetric keys used to encrypt/decrypt signing key data in DB. The first key is considered the active key and used for encryption, while the others are used to decrypt.")

	useOldFormat := fs.Bool("use-deprecated-secret-format", false, "In prior releases, the database used AES-CBC to encrypt keys. New deployments should use the default AES-GCM encryption.")

	dbURL := fs.String("db-url", "", "DSN-formatted database connection string")

	dbMigrate := fs.Bool("db-migrate", true, "perform database migrations when starting up overlord. This includes the initial DB objects creation.")

	keyPeriod := fs.Duration("key-period", 24*time.Hour, "length of time for-which a given key will be valid")
	gcInterval := fs.Duration("gc-interval", time.Hour, "length of time between garbage collection runs")

	adminListen := fs.String("admin-listen", "http://127.0.0.1:5557", "scheme, host and port for listening for administrative operation requests ")

	adminAPISecret := pflag.NewBase64(server.AdminAPISecretLength)
	fs.Var(adminAPISecret, "admin-api-secret", fmt.Sprintf("A base64-encoded %d byte string which is used to protect the Admin API.", server.AdminAPISecretLength))

	localConnectorID := fs.String("local-connector", "local", "ID of the local connector")
	logDebug := fs.Bool("log-debug", false, "log debug-level information")
	logTimestamps := fs.Bool("log-timestamps", false, "prefix log lines with timestamps")

	printVersion := fs.Bool("version", false, "Print the version and exit")

	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if err := pflag.SetFlagsFromEnv(fs, "DEX_OVERLORD"); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if *printVersion {
		fmt.Printf("dex version %s\ngo version %s\n", strings.TrimPrefix(version, "v"), strings.TrimPrefix(runtime.Version(), "go"))
		os.Exit(0)
	}

	if *logDebug {
		log.EnableDebug()
	}
	if *logTimestamps {
		log.EnableTimestamps()
	}

	adminURL, err := url.Parse(*adminListen)
	if err != nil {
		log.Fatalf("Unable to use --admin-listen flag: %v", err)
	}

	if len(keySecrets.BytesSlice()) == 0 {
		log.Fatalf("Must specify at least one key secret")
	}

	dbCfg := db.Config{
		DSN:                *dbURL,
		MaxIdleConnections: 1,
		MaxOpenConnections: 1,
	}
	dbc, err := db.NewConnection(dbCfg)
	if err != nil {
		log.Fatalf(err.Error())
	}
	if _, ok := dbc.Dialect.(gorp.PostgresDialect); !ok {
		log.Fatal("only postgres backend supported for multi server configurations")
	}

	if *dbMigrate {
		var sleep time.Duration
		for {
			var err error
			var migrations int
			if migrations, err = db.MigrateToLatest(dbc); err == nil {
				log.Infof("Performed %d db migrations", migrations)
				break
			}
			sleep = ptime.ExpBackoff(sleep, time.Minute)
			log.Errorf("Unable to migrate database, retrying in %v: %v", sleep, err)
			time.Sleep(sleep)
		}
	}
	userRepo := db.NewUserRepo(dbc)
	pwiRepo := db.NewPasswordInfoRepo(dbc)
	orgRepo := db.NewOrganizationRepo(dbc)
	connCfgRepo := db.NewConnectorConfigRepo(dbc)
	clientRepo := db.NewClientRepo(dbc)
	userManager := manager.NewUserManager(userRepo, pwiRepo, orgRepo, connCfgRepo, db.TransactionFactory(dbc), manager.ManagerOptions{})
	clientManager := clientmanager.NewClientManager(clientRepo, db.TransactionFactory(dbc), clientmanager.ManagerOptions{})
	connectorConfigRepo := db.NewConnectorConfigRepo(dbc)

	adminAPI := admin.NewAdminAPI(userRepo, pwiRepo, clientRepo, connectorConfigRepo, userManager, clientManager, *localConnectorID)
	kRepo, err := db.NewPrivateKeySetRepo(dbc, *useOldFormat, keySecrets.BytesSlice()...)
	if err != nil {
		log.Fatalf(err.Error())
	}

	var sleep time.Duration
	for {
		var done bool
		_, err := kRepo.Get()
		switch err {
		case nil:
			done = true
		case key.ErrorNoKeys:
			done = true
		case db.ErrorCannotDecryptKeys:
			log.Fatalf("Cannot decrypt keys using any of the given key secrets. The key secrets must be changed to include one that can decrypt the existing keys, or the existing keys must be deleted.")
		}

		if done {
			break
		}
		sleep = ptime.ExpBackoff(sleep, time.Minute)
		log.Errorf("Unable to get keys from repository, retrying in %v: %v", sleep, err)
		time.Sleep(sleep)
	}

	krot := key.NewPrivateKeyRotator(kRepo, *keyPeriod)
	s := server.NewAdminServer(adminAPI, krot, adminAPISecret.String())
	h := s.HTTPHandler()
	httpsrv := &http.Server{
		Addr:    adminURL.Host,
		Handler: h,
	}

	gc := db.NewGarbageCollector(dbc, *gcInterval)

	log.Infof("Binding to %s...", httpsrv.Addr)
	go func() {
		log.Fatal(httpsrv.ListenAndServe())
	}()

	gc.Run()
	<-krot.Run()
}

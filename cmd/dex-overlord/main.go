package main

import (
	"expvar"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/coreos/go-oidc/key"

	"github.com/coreos/dex/admin"
	"github.com/coreos/dex/db"
	pflag "github.com/coreos/dex/pkg/flag"
	"github.com/coreos/dex/pkg/log"
	ptime "github.com/coreos/dex/pkg/time"
	"github.com/coreos/dex/server"
	"github.com/coreos/dex/user"
)

var version = "DEV"

func init() {
	expvar.NewString("dex.version").Set(version)
}

func main() {
	fs := flag.NewFlagSet("dex-overlord", flag.ExitOnError)

	keySecrets := pflag.NewBase64List(32)
	fs.Var(keySecrets, "key-secrets", "A comma-separated list of base64 encoded 32 byte strings used as symmetric keys used to encrypt/decrypt signing key data in DB. The first key is considered the active key and used for encryption, while the others are used to decrypt.")

	dbURL := fs.String("db-url", "", "DSN-formatted database connection string")

	dbMigrate := fs.Bool("db-migrate", true, "perform database migrations when starting up overlord. This includes the initial DB objects creation.")

	keyPeriod := fs.Duration("key-period", 24*time.Hour, "length of time for-which a given key will be valid")
	gcInterval := fs.Duration("gc-interval", time.Hour, "length of time between garbage collection runs")

	adminListen := fs.String("admin-listen", "http://0.0.0.0:5557", "scheme, host and port for listening for administrative operation requests ")

	localConnectorID := fs.String("local-connector", "local", "ID of the local connector")
	logDebug := fs.Bool("log-debug", false, "log debug-level information")
	logTimestamps := fs.Bool("log-timestamps", false, "prefix log lines with timestamps")

	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if err := pflag.SetFlagsFromEnv(fs, "DEX_OVERLORD"); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
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

	dbCfg := db.Config{
		DSN:                *dbURL,
		MaxIdleConnections: 1,
		MaxOpenConnections: 1,
	}
	dbc, err := db.NewConnection(dbCfg)
	if err != nil {
		log.Fatalf(err.Error())
	}

	if *dbMigrate {
		var sleep time.Duration
		for {
			if migrations, err := db.MigrateToLatest(dbc); err == nil {
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
	userManager := user.NewManager(userRepo,
		pwiRepo, db.TransactionFactory(dbc), user.ManagerOptions{})
	adminAPI := admin.NewAdminAPI(userManager, userRepo, pwiRepo, *localConnectorID)
	kRepo, err := db.NewPrivateKeySetRepo(dbc, keySecrets.BytesSlice()...)
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
	s := server.NewAdminServer(adminAPI, krot)
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

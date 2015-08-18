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
	"github.com/coreos/dex/server"
	"github.com/coreos/dex/user"
)

var version = "DEV"

func init() {
	expvar.NewString("dex.version").Set(version)
}

func main() {
	fs := flag.NewFlagSet("dex-overlord", flag.ExitOnError)
	secret := fs.String("key-secret", "", "symmetric key used to encrypt/decrypt signing key data in DB")
	dbURL := fs.String("db-url", "", "DSN-formatted database connection string")
	keyPeriod := fs.Duration("key-period", 24*time.Hour, "length of time for-which a given key will be valid")
	gcInterval := fs.Duration("gc-interval", time.Hour, "length of time between garbage collection runs")

	adminListen := fs.String("admin-listen", "http://0.0.0.0:5557", "scheme, host and port for listening for administrative operation requests ")

	logDebug := fs.Bool("log-debug", false, "log debug-level information")
	logTimestamps := fs.Bool("log-timestamps", false, "prefix log lines with timestamps")
	localConnectorID := fs.String("local-connector", "local", "ID of the local connector")

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

	if len(*secret) == 0 {
		log.Fatalf("--key-secret unset")
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

	userRepo := db.NewUserRepo(dbc)
	pwiRepo := db.NewPasswordInfoRepo(dbc)
	userManager := user.NewManager(userRepo,
		pwiRepo, db.TransactionFactory(dbc), user.ManagerOptions{})
	adminAPI := admin.NewAdminAPI(userManager, userRepo, pwiRepo, *localConnectorID)
	kRepo, err := db.NewPrivateKeySetRepo(dbc, *secret)
	if err != nil {
		log.Fatalf(err.Error())
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

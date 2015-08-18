package main

import (
	"expvar"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/handlers"

	"github.com/coreos/dex/db"
	pflag "github.com/coreos/dex/pkg/flag"
	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/server"

	"github.com/coreos/pkg/flagutil"
)

var version = "DEV"

func init() {
	versionVar := expvar.NewString("dex.version")
	versionVar.Set(version)
}

func main() {
	fs := flag.NewFlagSet("dex-worker", flag.ExitOnError)
	listen := fs.String("listen", "http://0.0.0.0:5556", "")
	issuer := fs.String("issuer", "http://127.0.0.1:5556", "")
	templates := fs.String("html-assets", "./static/html", "directory of html template files")

	emailTemplateDirs := flagutil.StringSliceFlag{"./static/email"}
	fs.Var(&emailTemplateDirs, "email-templates", "comma separated list of directories of email template files")

	emailFrom := fs.String("email-from", "no-reply@coreos.com", "emails sent from dex will come from this address")
	emailConfig := fs.String("email-cfg", "./static/fixtures/emailer.json", "configures emailer.")

	noDB := fs.Bool("no-db", false, "manage entities in-process w/o any encryption, used only for single-node testing")

	// ignored if --no-db is set
	dbURL := fs.String("db-url", "", "DSN-formatted database connection string")
	keySecret := fs.String("key-secret", "", "symmetric key used to encrypt/decrypt signing key data in DB")
	dbMaxIdleConns := fs.Int("db-max-idle-conns", 0, "maximum number of connections in the idle connection pool")
	dbMaxOpenConns := fs.Int("db-max-open-conns", 0, "maximum number of open connections to the database")

	// used only if --no-db is set
	connectors := fs.String("connectors", "./static/fixtures/connectors.json", "JSON file containg set of IDPC configs")
	clients := fs.String("clients", "./static/fixtures/clients.json", "json file containing set of clients")
	users := fs.String("users", "./static/fixtures/users.json", "json file containing set of users")

	logDebug := fs.Bool("log-debug", false, "log debug-level information")
	logTimestamps := fs.Bool("log-timestamps", false, "prefix log lines with timestamps")

	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if err := pflag.SetFlagsFromEnv(fs, "DEX_WORKER"); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if *logDebug {
		log.EnableDebug()
		log.Infof("Debug logging enabled.")
		log.Debugf("Debug logging enabled.")
	}
	if *logTimestamps {
		log.EnableTimestamps()
	}

	lu, err := url.Parse(*listen)
	if err != nil {
		log.Fatalf("Unable to use --listen flag: %v", err)
	}

	if lu.Scheme != "http" {
		log.Fatalf("Unable to listen using scheme %s", lu.Scheme)
	}

	scfg := server.ServerConfig{
		IssuerURL:         *issuer,
		TemplateDir:       *templates,
		EmailTemplateDirs: emailTemplateDirs,
		EmailFromAddress:  *emailFrom,
		EmailerConfigFile: *emailConfig,
	}

	if *noDB {
		log.Warning("Running in-process without external database or key rotation")
		scfg.StateConfig = &server.SingleServerConfig{
			ClientsFile:    *clients,
			ConnectorsFile: *connectors,
			UsersFile:      *users,
		}
	} else {
		if *dbMaxIdleConns == 0 {
			log.Warning("Running with no limit on: database idle connections")
		}
		if *dbMaxOpenConns == 0 {
			log.Warning("Running with no limit on: database open connections")
		}
		dbCfg := db.Config{
			DSN:                *dbURL,
			MaxIdleConnections: *dbMaxIdleConns,
			MaxOpenConnections: *dbMaxOpenConns,
		}
		scfg.StateConfig = &server.MultiServerConfig{
			KeySecret:      *keySecret,
			DatabaseConfig: dbCfg,
		}
	}

	srv, err := scfg.Server()
	if err != nil {
		log.Fatalf("Unable to build Server: %v", err)
	}

	cfgs, err := srv.ConnectorConfigRepo.All()
	if err != nil {
		log.Fatalf("Unable to fetch connector configs from repo: %v", err)
	}

	for _, cfg := range cfgs {
		cfg := cfg
		if err = srv.AddConnector(cfg); err != nil {
			log.Fatalf("Failed registering connector: %v", err)
		}
	}

	h := srv.HTTPHandler()

	h = handlers.LoggingHandler(log.InfoWriter(), h)

	httpsrv := &http.Server{
		Addr:    lu.Host,
		Handler: h,
	}

	log.Infof("Binding to %s...", httpsrv.Addr)
	go func() {
		log.Fatal(httpsrv.ListenAndServe())
	}()

	<-srv.Run()
}

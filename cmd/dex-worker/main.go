package main

import (
	"expvar"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/db"
	_ "github.com/coreos/dex/db/memory"
	_ "github.com/coreos/dex/db/postgresql"
	pflag "github.com/coreos/dex/pkg/flag"
	"github.com/coreos/dex/pkg/log"
	ptime "github.com/coreos/dex/pkg/time"
	"github.com/coreos/dex/server"
	"github.com/coreos/pkg/flagutil"
	"github.com/gorilla/handlers"
)

var version = "DEV"

func init() {
	versionVar := expvar.NewString("dex.version")
	versionVar.Set(version)
}

func main() {
	fs := flag.NewFlagSet("dex-worker", flag.ExitOnError)
	listen := fs.String("listen", "http://127.0.0.1:5556", "the address that the server will listen on")

	issuer := fs.String("issuer", "http://127.0.0.1:5556", "the issuer's location")

	certFile := fs.String("tls-cert-file", "", "the server's certificate file for TLS connection")
	keyFile := fs.String("tls-key-file", "", "the server's private key file for TLS connection")

	templates := fs.String("html-assets", "./static/html", "directory of html template files")

	emailTemplateDirs := flagutil.StringSliceFlag{"./static/email"}
	fs.Var(&emailTemplateDirs, "email-templates", "comma separated list of directories of email template files")

	emailFrom := fs.String("email-from", "", "emails sent from dex will come from this address")
	emailConfig := fs.String("email-cfg", "./static/fixtures/emailer.json", "configures emailer.")

	enableRegistration := fs.Bool("enable-registration", false, "Allows users to self-register")

	dnames := db.GetDriverNames()
	dbName := fs.String("db", "memory", fmt.Sprintf("The database. Available drivers: %s", strings.Join(dnames, ", ")))
	for _, d := range dnames {
		db.GetDriver(d).InitFlags(fs)
	}
	// UI-related:
	issuerName := fs.String("issuer-name", "dex", "The name of this dex installation; will appear on most pages.")
	issuerLogoURL := fs.String("issuer-logo-url", "https://coreos.com/assets/images/brand/coreos-wordmark-135x40px.png", "URL of an image representing the issuer")

	keySecrets := pflag.NewBase64List(32)
	fs.Var(keySecrets, "key-secrets", "A comma-separated list of base64 encoded 32 byte strings used as symmetric keys used to encrypt/decrypt signing key data in DB. The first key is considered the active key and used for encryption, while the others are used to decrypt.")
	useOldFormat := fs.Bool("use-deprecated-secret-format", false, "In prior releases, the database used AES-CBC to encrypt keys. New deployments should use the default AES-GCM encryption.")

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

	rd := db.GetDriver(*dbName)
	if rd == nil {
		fmt.Fprintf(os.Stderr, "there is no '%s' named db driver\n", *dbName)
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

	// Validate listen address.
	lu, err := url.Parse(*listen)
	if err != nil {
		log.Fatalf("Invalid listen address %q: %v", *listen, err)
	}

	switch lu.Scheme {
	case "http":
	case "https":
		if *certFile == "" || *keyFile == "" {
			log.Fatalf("Must provide certificate file and private key file")
		}
	default:
		log.Fatalf("Only 'http' and 'https' schemes are supported")
	}

	// Validate issuer address.
	iu, err := url.Parse(*issuer)
	if err != nil {
		log.Fatalf("Invalid issuer URL %q: %v", *issuer, err)
	}

	if iu.Scheme != "http" && iu.Scheme != "https" {
		log.Fatalf("Only 'http' and 'https' schemes are supported")
	}
	dbDriver, err := rd.New()
	if err != nil {
		log.Fatalf("Error while initializing db %v", err)
	}

	scfg := server.ServerConfig{
		IssuerURL:          *issuer,
		TemplateDir:        *templates,
		EmailTemplateDirs:  emailTemplateDirs,
		EmailFromAddress:   *emailFrom,
		EmailerConfigFile:  *emailConfig,
		IssuerName:         *issuerName,
		IssuerLogoURL:      *issuerLogoURL,
		EnableRegistration: *enableRegistration,
		UseOldFormat:       *useOldFormat,
		KeySecrets:         keySecrets.BytesSlice(),
		DB:                 dbDriver,
	}

	srv, err := scfg.Server()
	if err != nil {
		log.Fatalf("Unable to build Server: %v", err)
	}

	var cfgs []connector.ConnectorConfig
	var sleep time.Duration
	for {
		var err error
		cfgs, err = srv.ConnectorConfigRepo.All()
		if len(cfgs) > 0 && err == nil {
			break
		}
		sleep = ptime.ExpBackoff(sleep, 8*time.Second)
		if err != nil {
			log.Errorf("Unable to load connectors, retrying in %v: %v", sleep, err)
		} else {
			log.Errorf("No connectors, will wait. Retrying in %v.", sleep)
		}
		time.Sleep(sleep)
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
		if lu.Scheme == "http" {
			log.Fatal(httpsrv.ListenAndServe())
		} else {
			log.Fatal(httpsrv.ListenAndServeTLS(*certFile, *keyFile))
		}
	}()

	<-srv.Run()
}

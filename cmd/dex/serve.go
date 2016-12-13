package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/coreos/dex/api"
	"github.com/coreos/dex/server"
	"github.com/coreos/dex/storage"
)

func commandServe() *cobra.Command {
	return &cobra.Command{
		Use:     "serve [ config file ]",
		Short:   "Connect to the storage and begin serving requests.",
		Long:    ``,
		Example: "dex serve config.yaml",
		RunE:    serve,
	}
}

func serve(cmd *cobra.Command, args []string) error {
	switch len(args) {
	default:
		return errors.New("surplus arguments")
	case 0:
		// TODO(ericchiang): Consider having a default config file location.
		return errors.New("no config file specified")
	case 1:
	}

	configFile := args[0]
	configData, err := ioutil.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("read config file %s: %v", configFile, err)
	}

	var c Config
	if err := yaml.Unmarshal(configData, &c); err != nil {
		return fmt.Errorf("parse config file %s: %v", configFile, err)
	}

	// Fast checks. Perform these first for a more responsive CLI.
	checks := []struct {
		bad    bool
		errMsg string
	}{
		{c.Issuer == "", "no issuer specified in config file"},
		{len(c.Connectors) == 0 && !c.EnablePasswordDB, "no connectors supplied in config file"},
		{!c.EnablePasswordDB && len(c.StaticPasswords) != 0, "cannot specify static passwords without enabling password db"},
		{c.Storage.Config == nil, "no storage suppied in config file"},
		{c.Web.HTTP == "" && c.Web.HTTPS == "", "must supply a HTTP/HTTPS  address to listen on"},
		{c.Web.HTTPS != "" && c.Web.TLSCert == "", "no cert specified for HTTPS"},
		{c.Web.HTTPS != "" && c.Web.TLSKey == "", "no private key specified for HTTPS"},
		{c.GRPC.TLSCert != "" && c.GRPC.Addr == "", "no address specified for gRPC"},
		{c.GRPC.TLSKey != "" && c.GRPC.Addr == "", "no address specified for gRPC"},
		{(c.GRPC.TLSCert == "") != (c.GRPC.TLSKey == ""), "must specific both a gRPC TLS cert and key"},
		{c.GRPC.TLSCert == "" && c.GRPC.TLSClientCA != "", "cannot specify gRPC TLS client CA without a gRPC TLS cert"},
	}

	for _, check := range checks {
		if check.bad {
			return errors.New(check.errMsg)
		}
	}

	var grpcOptions []grpc.ServerOption
	if c.GRPC.TLSCert != "" {
		if c.GRPC.TLSClientCA != "" {
			// Parse certificates from certificate file and key file for server.
			cert, err := tls.LoadX509KeyPair(c.GRPC.TLSCert, c.GRPC.TLSKey)
			if err != nil {
				return fmt.Errorf("parsing certificate file: %v", err)
			}

			// Parse certificates from client CA file to a new CertPool.
			cPool := x509.NewCertPool()
			clientCert, err := ioutil.ReadFile(c.GRPC.TLSClientCA)
			if err != nil {
				return fmt.Errorf("reading from client CA file: %v", err)
			}
			if cPool.AppendCertsFromPEM(clientCert) != true {
				return errors.New("failed to parse client CA")
			}

			tlsConfig := tls.Config{
				Certificates: []tls.Certificate{cert},
				ClientAuth:   tls.RequireAndVerifyClientCert,
				ClientCAs:    cPool,
			}
			grpcOptions = append(grpcOptions, grpc.Creds(credentials.NewTLS(&tlsConfig)))
		} else {
			opt, err := credentials.NewServerTLSFromFile(c.GRPC.TLSCert, c.GRPC.TLSKey)
			if err != nil {
				return fmt.Errorf("load grpc certs: %v", err)
			}
			grpcOptions = append(grpcOptions, grpc.Creds(opt))
		}
	}

	logger, _ := newLogger(c.Logger.Level, c.Logger.Format)

	connectors := make([]server.Connector, len(c.Connectors))
	for i, conn := range c.Connectors {
		if conn.ID == "" {
			return fmt.Errorf("no ID field for connector %d", i)
		}
		if conn.Config == nil {
			return fmt.Errorf("no config field for connector %q", conn.ID)
		}
		connectorLogger := logger.WithField("connector", conn.Name)
		c, err := conn.Config.Open(connectorLogger)
		if err != nil {
			return fmt.Errorf("open %s: %v", conn.ID, err)
		}
		connectors[i] = server.Connector{
			ID:          conn.ID,
			DisplayName: conn.Name,
			Connector:   c,
		}
	}

	s, err := c.Storage.Config.Open(logger)
	if err != nil {
		return fmt.Errorf("initializing storage: %v", err)
	}
	if len(c.StaticClients) > 0 {
		s = storage.WithStaticClients(s, c.StaticClients)
	}
	if len(c.StaticPasswords) > 0 {
		passwords := make([]storage.Password, len(c.StaticPasswords))
		for i, p := range c.StaticPasswords {
			passwords[i] = storage.Password(p)
		}
		s = storage.WithStaticPasswords(s, passwords)
	}

	serverConfig := server.Config{
		SupportedResponseTypes: c.OAuth2.ResponseTypes,
		SkipApprovalScreen:     c.OAuth2.SkipApprovalScreen,
		Issuer:                 c.Issuer,
		Connectors:             connectors,
		Storage:                s,
		Web:                    c.Frontend,
		EnablePasswordDB:       c.EnablePasswordDB,
		Logger:                 logger,
	}
	if c.Expiry.SigningKeys != "" {
		signingKeys, err := time.ParseDuration(c.Expiry.SigningKeys)
		if err != nil {
			return fmt.Errorf("parsing signingKeys expiry: %v", err)
		}
		serverConfig.RotateKeysAfter = signingKeys
	}
	if c.Expiry.IDTokens != "" {
		idTokens, err := time.ParseDuration(c.Expiry.IDTokens)
		if err != nil {
			return fmt.Errorf("parsing idTokens expiry: %v", err)
		}
		serverConfig.IDTokensValidFor = idTokens
	}

	serv, err := server.NewServer(context.Background(), serverConfig)
	if err != nil {
		return fmt.Errorf("initializing server: %v", err)
	}
	errc := make(chan error, 3)
	if c.Web.HTTP != "" {
		logger.Errorf("listening (http) on %s", c.Web.HTTP)
		go func() {
			errc <- http.ListenAndServe(c.Web.HTTP, serv)
		}()
	}
	if c.Web.HTTPS != "" {
		logger.Errorf("listening (https) on %s", c.Web.HTTPS)
		go func() {
			errc <- http.ListenAndServeTLS(c.Web.HTTPS, c.Web.TLSCert, c.Web.TLSKey, serv)
		}()
	}
	if c.GRPC.Addr != "" {
		logger.Errorf("listening (grpc) on %s", c.GRPC.Addr)
		go func() {
			errc <- func() error {
				list, err := net.Listen("tcp", c.GRPC.Addr)
				if err != nil {
					return fmt.Errorf("listen grpc: %v", err)
				}
				s := grpc.NewServer(grpcOptions...)
				api.RegisterDexServer(s, server.NewAPI(serverConfig.Storage, logger))
				return s.Serve(list)
			}()
		}()
	}

	return <-errc
}

func newLogger(level string, format string) (logrus.FieldLogger, error) {
	var logLevel logrus.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = logrus.DebugLevel
	case "", "info":
		logLevel = logrus.InfoLevel
	case "error":
		logLevel = logrus.ErrorLevel
	default:
		return nil, fmt.Errorf("unsupported logLevel: %s", level)
	}

	var formatter logrus.Formatter
	switch strings.ToLower(format) {
	case "", "text":
		formatter = &logrus.TextFormatter{DisableColors: true}
	case "json":
		formatter = &logrus.JSONFormatter{}
	default:
		return nil, fmt.Errorf("unsupported logger format: %s", format)
	}

	return &logrus.Logger{
		Out:       os.Stderr,
		Formatter: formatter,
		Level:     logLevel,
	}, nil
}

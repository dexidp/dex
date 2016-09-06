package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"

	"github.com/coreos/dex/server"
	"github.com/coreos/dex/storage"
)

func commandServe() *cobra.Command {
	return &cobra.Command{
		Use:     "serve [ config file ]",
		Short:   "Connect to the storage and begin serving requests.",
		Long:    ``,
		Example: "dex serve c.yaml",
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
		{len(c.Connectors) == 0, "no connectors supplied in config file"},
		{c.Storage.Config == nil, "no storage suppied in config file"},
		{c.Web.HTTP == "" && c.Web.HTTPS == "", "must supply a HTTP/HTTPS  address to listen on"},
		{c.Web.HTTPS != "" && c.Web.TLSCert == "", "no cert specified for HTTPS"},
		{c.Web.HTTPS != "" && c.Web.TLSKey == "", "no private key specified for HTTPS"},
	}

	for _, check := range checks {
		if check.bad {
			return errors.New(check.errMsg)
		}
	}

	connectors := make([]server.Connector, len(c.Connectors))
	for i, conn := range c.Connectors {
		if conn.Config == nil {
			return fmt.Errorf("no config field for connector %q", conn.ID)
		}
		c, err := conn.Config.Open()
		if err != nil {
			return fmt.Errorf("open %s: %v", conn.ID, err)
		}
		connectors[i] = server.Connector{
			ID:          conn.ID,
			DisplayName: conn.Name,
			Connector:   c,
		}
	}

	s, err := c.Storage.Config.Open()
	if err != nil {
		return fmt.Errorf("initializing storage: %v", err)
	}
	if len(c.StaticClients) > 0 {
		s = storage.WithStaticClients(s, c.StaticClients)
	}

	serverConfig := server.Config{
		SupportedResponseTypes: c.OAuth2.ResponseTypes,
		Issuer:                 c.Issuer,
		Connectors:             connectors,
		Storage:                s,
		TemplateConfig:         c.Templates,
	}

	serv, err := server.New(serverConfig)
	if err != nil {
		return fmt.Errorf("initializing server: %v", err)
	}
	errc := make(chan error, 2)
	if c.Web.HTTP != "" {
		go func() {
			log.Printf("listening on %s", c.Web.HTTP)
			errc <- http.ListenAndServe(c.Web.HTTP, serv)
		}()
	}
	if c.Web.HTTPS != "" {
		go func() {
			log.Printf("listening on %s", c.Web.HTTPS)
			errc <- http.ListenAndServeTLS(c.Web.HTTPS, c.Web.TLSCert, c.Web.TLSKey, serv)
		}()
	}
	return <-errc
}

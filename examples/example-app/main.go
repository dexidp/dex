package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dexidp/dex/examples/example-app/server"
)

func cmd() *cobra.Command {
	var (
		opts    server.Options
		listen  string
		tlsCert string
		tlsKey  string
	)

	c := cobra.Command{
		Use:   "example-app",
		Short: "An example OpenID Connect client",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New("surplus arguments provided")
			}

			s, err := server.New(opts)
			if err != nil {
				return err
			}
			return s.Run(listen, tlsCert, tlsKey)
		},
	}

	c.Flags().StringVar(&opts.ClientID, "client-id", "example-app", "OAuth2 client ID of this application.")
	c.Flags().StringVar(&opts.ClientSecret, "client-secret", "ZXhhbXBsZS1hcHAtc2VjcmV0", "OAuth2 client secret of this application.")
	c.Flags().BoolVar(&opts.PKCE, "pkce", true, "Use PKCE flow for the code exchange.")
	c.Flags().StringVar(&opts.RedirectURI, "redirect-uri", "http://127.0.0.1:5555/callback", "Callback URL for OAuth2 responses.")
	c.Flags().StringVar(&opts.IssuerURL, "issuer", "http://127.0.0.1:5556/dex", "URL of the OpenID Connect issuer.")
	c.Flags().StringVar(&listen, "listen", "http://127.0.0.1:5555", "HTTP(S) address to listen at.")
	c.Flags().StringVar(&tlsCert, "tls-cert", "", "X509 cert file to present when serving HTTPS.")
	c.Flags().StringVar(&tlsKey, "tls-key", "", "Private key for the HTTPS cert.")
	c.Flags().StringVar(&opts.RootCAs, "issuer-root-ca", "", "Root certificate authorities for the issuer. Defaults to host certs.")
	c.Flags().BoolVar(&opts.Debug, "debug", false, "Print all request and responses from the OpenID Connect issuer.")
	c.Flags().BoolVar(&opts.SessionAware, "session-aware", false, "Check Dex session on index page via prompt=none and show logout button.")

	return &c
}

func main() {
	if err := cmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
}

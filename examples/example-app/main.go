package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/dexidp/dex/examples/example-app/server"
)

// defaultSessionCheck is short enough that signing out of dex in another tab
// shows up here while you are still looking at the page.
const defaultSessionCheck = 30 * time.Second

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
		Long: `An example OpenID Connect client for dex.

It runs the flows dex implements — authorization code, device code, refresh,
client credentials, password and token exchange — and shows what comes back.
Given --grpc-addr it also talks to dex's management API.`,
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

	flags := c.Flags()

	// This client.
	flags.StringVar(&opts.ClientID, "client-id", "example-app", "OAuth2 client ID of this application.")
	flags.StringVar(&opts.ClientSecret, "client-secret", "ZXhhbXBsZS1hcHAtc2VjcmV0", "OAuth2 client secret of this application.")
	flags.StringVar(&opts.RedirectURI, "redirect-uri", "http://127.0.0.1:5555/callback", "Callback URL for OAuth2 responses. Its path is where this app serves the callback.")
	flags.BoolVar(&opts.PKCE, "pkce", true, "Send a PKCE challenge with authorization requests.")

	// The provider.
	flags.StringVar(&opts.IssuerURL, "issuer", "http://127.0.0.1:5556/dex", "URL of the OpenID Connect issuer.")
	flags.StringVar(&opts.RootCAs, "issuer-root-ca", "", "Root certificate authorities for the issuer. Defaults to host certs.")
	flags.DurationVar(&opts.SessionCheckInterval, "session-check-interval", defaultSessionCheck,
		"How often to confirm with the provider, using prompt=none, that its session still exists. Zero stops checking, which lets this app show a user who has signed out of the provider.")

	// The management API.
	flags.StringVar(&opts.GRPCAddr, "grpc-addr", "", "Address of dex's gRPC API, e.g. 127.0.0.1:5557. Enables the API page; empty leaves it out.")
	flags.StringVar(&opts.GRPCCA, "grpc-ca", "", "CA certificate for the gRPC API. Without it the connection is plaintext.")
	flags.StringVar(&opts.GRPCClientCert, "grpc-client-cert", "", "Client certificate for the gRPC API.")
	flags.StringVar(&opts.GRPCClientKey, "grpc-client-key", "", "Client key for the gRPC API.")

	// This process.
	flags.StringVar(&listen, "listen", "http://127.0.0.1:5555", "HTTP(S) address to listen at.")
	flags.StringVar(&tlsCert, "tls-cert", "", "X509 cert file to present when serving HTTPS.")
	flags.StringVar(&tlsKey, "tls-key", "", "Private key for the HTTPS cert.")
	flags.BoolVar(&opts.Debug, "debug", false, "Print all requests and responses to and from the issuer.")

	return &c
}

func main() {
	if err := cmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
}

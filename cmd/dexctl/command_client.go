package main

import (
	"net/url"

	"github.com/coreos/go-oidc/oidc"
	"github.com/spf13/cobra"
	"os"
)

var (
	cmdNewClient = &cobra.Command{
		Use:     "new-client",
		Short:   "Create a new client with one or more redirect URLs.",
		Long:    "Create a new client with one or more redirect URLs,",
		Example: `  dexctl new-client --base-url=${OVER_LORD_URL} --api-key=${ADMIN_API_KEY} 'https://example.com/callback'`,
		Run:     wrapRun(runNewClient),
	}
)

func init() {
	rootCmd.AddCommand(cmdNewClient)
}

func runNewClient(cmd *cobra.Command, args []string) int {
	if len(args) < 1 {
		stderr("Provide at least one redirect URL.")
		return 2
	}

	redirectURLs := make([]url.URL, len(args))
	for i, ua := range args {
		u, err := url.Parse(ua)
		if err != nil {
			stderr("Malformed URL %q: %v", ua, err)
			return 1
		}
		redirectURLs[i] = *u
	}
	var clientCredential *oidc.ClientCredentials
	if useDBConnector, err := isDBURLPresent(); err != nil {
		stderr("Unable to configure dexctl : %v", err)
		os.Exit(1)
	} else {
		if useDBConnector {

			dbConnector, dbConnectorError := getDBConnector()
			if dbConnectorError != nil {
				stderr("Failed creating database connector: %v", dbConnectorError)
				return 1
			}
			if cc, err := dbConnector.NewClient(oidc.ClientMetadata{RedirectURIs: redirectURLs}); err != nil {
				stderr("Failed creating new client: %v", err)
				return 1
			} else {
				clientCredential = cc
			}

		} else {
			adminAPIConnector, adminAPIConnectorError := getAdminAPIConnector()
			if adminAPIConnectorError != nil {
				stderr("Failed creating admin api client : %v", adminAPIConnectorError)
				return 1
			}
			if cc, err := adminAPIConnector.NewClient(oidc.ClientMetadata{RedirectURIs: redirectURLs}); err != nil {
				stderr("Failed creating new client: %v", err)
				return 1
			} else {
				clientCredential = cc
			}
		}
	}

	stdout("# Added new client:")
	stdout("DEX_APP_CLIENT_ID=%s", clientCredential.ID)
	stdout("DEX_APP_CLIENT_SECRET=%s", clientCredential.Secret)
	for i, u := range redirectURLs {
		stdout("DEX_APP_REDIRECTURL_%d=%s", i, u.String())
	}

	return 0
}

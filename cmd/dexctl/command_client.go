package main

import (
	"net/url"

	"github.com/coreos/go-oidc/oidc"
	"github.com/spf13/cobra"
)

var (
	cmdNewClient = &cobra.Command{
		Use:     "new-client",
		Short:   "Create a new client with one or more redirect URLs.",
		Long:    "Create a new client with one or more redirect URLs,",
		Example: `  dexctl new-client --db-url=${DB_URL} 'https://example.com/callback'`,
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

	redirectURLs := make([]*url.URL, len(args))
	for i, ua := range args {
		u, err := url.Parse(ua)
		if err != nil {
			stderr("Malformed URL %q: %v", ua, err)
			return 1
		}
		redirectURLs[i] = u
	}

	cc, err := getDriver().NewClient(oidc.ClientMetadata{RedirectURIs: redirectURLs})
	if err != nil {
		stderr("Failed creating new client: %v", err)
		return 1
	}

	stdout("# Added new client:")
	stdout("DEX_APP_CLIENT_ID=%s", cc.ID)
	stdout("DEX_APP_CLIENT_SECRET=%s", cc.Secret)
	for i, u := range redirectURLs {
		stdout("DEX_APP_REDIRECTURL_%d=%s", i, u.String())
	}

	return 0
}

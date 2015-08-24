package main

import (
	"net/url"

	"github.com/coreos/go-oidc/oidc"
)

var (
	cmdNewClient = &command{
		Name:    "new-client",
		Summary: "Create a new client with the provided redirect URL(s)",
		Usage:   "<URL>...",
		Run:     runNewClient,
	}
)

func init() {
	commands = append(commands, cmdNewClient)
}

func runNewClient(args []string) int {
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

	cc, err := getDriver().NewClient(oidc.ClientMetadata{RedirectURLs: redirectURLs})
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

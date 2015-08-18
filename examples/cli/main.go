package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	pflag "github.com/coreos/dex/pkg/flag"
	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/go-oidc/oidc"
)

func main() {
	fs := flag.NewFlagSet("example-cli", flag.ExitOnError)
	clientID := fs.String("client-id", "", "")
	clientSecret := fs.String("client-secret", "", "")
	discovery := fs.String("discovery", "http://localhost:5556", "")
	logDebug := fs.Bool("log-debug", false, "log debug-level information")
	logTimestamps := fs.Bool("log-timestamps", false, "prefix log lines with timestamps")

	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if err := pflag.SetFlagsFromEnv(fs, "EXAMPLE_CLI"); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if *logDebug {
		log.EnableDebug()
	}
	if *logTimestamps {
		log.EnableTimestamps()
	}

	if *clientID == "" {
		fmt.Println("--client-id must be set")
		os.Exit(2)
	}

	if *clientSecret == "" {
		fmt.Println("--client-secret must be set")
		os.Exit(2)
	}

	cc := oidc.ClientCredentials{
		ID:     *clientID,
		Secret: *clientSecret,
	}

	// NOTE: A real CLI would cache this config, or provide it via flags/config file.
	var cfg oidc.ProviderConfig
	var err error
	for {
		cfg, err = oidc.FetchProviderConfig(http.DefaultClient, *discovery)
		if err == nil {
			break
		}

		sleep := 1 * time.Second
		fmt.Printf("Failed fetching provider config, trying again in %v: %v\n", sleep, err)
		time.Sleep(sleep)
	}

	fmt.Printf("Fetched provider config from %s: %#v\n\n", *discovery, cfg)

	ccfg := oidc.ClientConfig{
		ProviderConfig: cfg,
		Credentials:    cc,
	}

	client, err := oidc.NewClient(ccfg)
	if err != nil {
		log.Fatalf("Unable to create Client: %v", err)
	}

	tok, err := client.ClientCredsToken([]string{"openid"})
	if err != nil {
		fmt.Printf("unable to verify auth code with issuer: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("got jwt: %v\n\n", tok.Encode())

	claims, err := tok.Claims()
	if err != nil {
		fmt.Printf("unable to construct claims: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("got claims %#v...\n", claims)
}

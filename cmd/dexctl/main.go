package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"net/http"
	"os"

	pflag "github.com/coreos/dex/pkg/flag"
	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/go-oidc/oidc"
)

var (
	cliName        = "dexctl"
	cliDescription = "A tool for interacting with the dex system"

	commands []*command
	globalFS = flag.NewFlagSet(cliName, flag.ExitOnError)

	global struct {
		endpoint string
		creds    oidc.ClientCredentials
		dbURL    string
		help     bool
		logDebug bool
	}
)

func init() {
	log.EnableTimestamps()

	globalFS.StringVar(&global.endpoint, "endpoint", "", "URL of dex API")
	globalFS.StringVar(&global.creds.ID, "client-id", "", "dex API user ID")
	globalFS.StringVar(&global.creds.Secret, "client-secret", "", "dex API user password")
	globalFS.StringVar(&global.dbURL, "db-url", "", "DSN-formatted database connection string")
	globalFS.BoolVar(&global.help, "help", false, "Print usage information and exit")
	globalFS.BoolVar(&global.help, "h", false, "Print usage information and exit")
	globalFS.BoolVar(&global.logDebug, "log-debug", false, "Log debug-level information")
}

func main() {
	err := parseFlags()
	if err != nil {
		stderr(err.Error())
		os.Exit(2)
	}

	if global.logDebug {
		log.EnableDebug()
	}

	args := globalFS.Args()
	if len(args) < 1 || global.help {
		args = []string{"help"}
	}

	var cmd *command
	for _, c := range commands {
		if c.Name == args[0] {
			cmd = c
			// Don't print the default help message.
			c.Flags.SetOutput(ioutil.Discard)
			if err := c.Flags.Parse(args[1:]); err != nil {
				if err == flag.ErrHelp {
					printCommandUsage(c)
				} else {
					stderr("%v", err)
				}
				os.Exit(2)
			}
			break
		}
	}

	if cmd == nil {
		stderr("%v: unknown subcommand: %q", cliName, args[0])
		stderr("Run '%v help' for usage.", cliName)
		os.Exit(2)
	}

	os.Exit(cmd.Run(cmd.Flags.Args()))
}

type command struct {
	Name        string       // Name of the command and the string to use to invoke it
	Summary     string       // One-sentence summary of what the command does
	Usage       string       // Usage options/arguments
	Description string       // Detailed description of command
	Flags       flag.FlagSet // Set of flags associated with this command

	Run func(args []string) int // Run a command with the given arguments, return exit status

}

func parseFlags() error {
	if err := globalFS.Parse(os.Args[1:]); err != nil {
		return err
	}

	return pflag.SetFlagsFromEnv(globalFS, "DEXCTL")
}

func getDriver() (drv driver) {
	var err error
	switch {
	case len(global.dbURL) > 0:
		drv, err = newDBDriver(global.dbURL)
	case len(global.endpoint) > 0:
		if len(global.creds.ID) == 0 || len(global.creds.Secret) == 0 {
			err = errors.New("--client-id/--client-secret flags unset")
			break
		}
		pcfg, err := oidc.FetchProviderConfig(http.DefaultClient, global.endpoint)
		if err != nil {
			stderr("Unable to fetch provider config: %v", err)
			os.Exit(1)
		}
		drv, err = newAPIDriver(pcfg, global.creds)
	default:
		err = errors.New("--endpoint/--db-url flags unset")
	}

	if err != nil {
		stderr("Unable to configure dexctl driver: %v", err)
		os.Exit(1)
	}

	return
}

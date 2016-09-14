package main

import (
	"errors"
	"os"
	"strings"

	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/go-oidc/oidc"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	rootCmd = &cobra.Command{
		Use:   "dexctl",
		Short: "A command line tool for interacting with the dex system",
		Long:  "",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// initialize flags from environment
			fs := cmd.Flags()

			// don't override flags set by command line flags
			alreadySet := make(map[string]bool)
			fs.Visit(func(f *pflag.Flag) { alreadySet[f.Name] = true })

			var err error
			fs.VisitAll(func(f *pflag.Flag) {
				if err != nil || alreadySet[f.Name] {
					return
				}
				key := "DEXCTL_" + strings.ToUpper(strings.Replace(f.Name, "-", "_", -1))
				if val := os.Getenv(key); val != "" {
					err = fs.Set(f.Name, val)
				}
			})
			return err
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			os.Exit(2)
		},
	}

	global struct {
		creds    oidc.ClientCredentials
		baseURL  string
		apiKey   string
		help     bool
		logDebug bool
	}
)

func init() {
	log.EnableTimestamps()

	rootCmd.PersistentFlags().StringVar(&global.baseURL, "base-url", "", "DSN-formatted dex-overlord base URL")
	rootCmd.PersistentFlags().StringVar(&global.apiKey, "api-key", "", "API key for Admin API")
	rootCmd.PersistentFlags().BoolVar(&global.logDebug, "log-debug", false, "Log debug-level information")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(2)
	}
}

func wrapRun(run func(cmd *cobra.Command, args []string) int) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		os.Exit(run(cmd, args))
	}
}

func getDriver() (drv driver) {
	var err error
	switch {
	case len(global.baseURL) < 1:
		err = errors.New("--db-url flag unset")
	case len(global.apiKey) < 1:
		err = errors.New("--api-key flag unset")
	default:
		drv, err = newAdminAPIDriver(global.baseURL, global.apiKey)
	}

	if err != nil {
		stderr("Unable to configure dexctl driver: %v", err)
		os.Exit(1)
	}

	return
}

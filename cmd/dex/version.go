package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/dexidp/dex/server"
)

var version = "DEV"

// nolint:gochecknoinits
func init() {
	// inject version for API endpoint
	server.Version = version
}

func commandVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version and exit",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf(
				"Dex Version: %s\nGo Version: %s\nGo OS/ARCH: %s %s\n",
				version,
				runtime.Version(),
				runtime.GOOS,
				runtime.GOARCH,
			)
		},
	}
}

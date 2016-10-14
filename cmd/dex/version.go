package main

import (
	"fmt"
	"runtime"

	"github.com/coreos/dex/version"
	"github.com/spf13/cobra"
)

func commandVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version and exit",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf(`dex Version: %s
Go Version: %s
Go OS/ARCH: %s %s
`, version.Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		},
	}
}

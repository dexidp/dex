package main

import (
	"fmt"
	"runtime"

	"github.com/coreos/poke/version"
	"github.com/spf13/cobra"
)

func commandVersion() *cobra.Command {
	return &cobra.Command{
		Use: "version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf(`v%s %s %s %s
`, version.Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		},
	}
}

package main

import (
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// set by the top level build script
	version = ""

	cmdVersion = &cobra.Command{
		Use:   "version",
		Short: "Print the dexctl version.",
		Long:  "Print the dexctl version.",
		Run: func(cmd *cobra.Command, args []string) {
			stdout("dex version %s\ngo version %s", strings.TrimPrefix(version, "v"), strings.TrimPrefix(runtime.Version(), "go"))
		},
	}
)

func init() {
	rootCmd.AddCommand(cmdVersion)
}

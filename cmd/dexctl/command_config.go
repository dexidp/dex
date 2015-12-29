package main

import (
	"fmt"

	"github.com/coreos/dex/connector"
	"github.com/spf13/cobra"
)

var (
	cmdGetConnectorConfigs = &cobra.Command{
		Use:     "get-connector-configs",
		Short:   "Enumerate current IdP connector configs.",
		Long:    "Enumerate current IdP connector configs.",
		Example: `  dexctl get-connector-configs --db-url=${DB_URL}`,
		Run:     wrapRun(runGetConnectorConfigs),
	}

	cmdSetConnectorConfigs = &cobra.Command{
		Use:     "set-connector-configs",
		Short:   "Overwrite the current IdP connector configs with those from a local file.",
		Long:    "Overwrite the current IdP connector configs with those from a local file.",
		Example: `  dexctl set-connector-configs --db-url=${DB_URL} ./static/conn_conf.json`,
		Run:     wrapRun(runSetConnectorConfigs),
	}
)

func init() {
	rootCmd.AddCommand(cmdGetConnectorConfigs)
	rootCmd.AddCommand(cmdSetConnectorConfigs)
}

func runSetConnectorConfigs(cmd *cobra.Command, args []string) int {
	if len(args) != 1 {
		stderr("Provide a single argument.")
		return 2
	}

	rf, err := connector.NewConnectorConfigRepoFromFile(args[0])
	if err != nil {
		stderr("Unable to retrieve connector configs from file: %v", err)
		return 1
	}

	cfgs, err := rf.All()
	if err != nil {
		stderr("Unable to retrieve connector configs from file: %v", err)
		return 1
	}

	if err := getDriver().SetConnectorConfigs(cfgs); err != nil {
		stderr(err.Error())
		return 1
	}

	fmt.Printf("Saved %d connector config(s)\n", len(cfgs))

	return 0
}

func runGetConnectorConfigs(cmd *cobra.Command, args []string) int {
	if len(args) != 0 {
		stderr("Provide zero arguments.")
		return 2
	}

	cfgs, err := getDriver().ConnectorConfigs()
	if err != nil {
		stderr("Unable to retrieve connector configs: %v", err)
		return 1
	}

	fmt.Printf("Found %d connector config(s)\n", len(cfgs))

	for _, cfg := range cfgs {
		fmt.Println()
		fmt.Printf("ID:   %v\n", cfg.ConnectorID())
		fmt.Printf("Type: %v\n", cfg.ConnectorType())
	}

	return 0
}

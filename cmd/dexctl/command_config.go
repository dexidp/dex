package main

import (
	"fmt"

	"github.com/coreos/dex/connector"
)

var (
	cmdGetConnectorConfigs = &command{
		Name:    "get-connector-configs",
		Summary: "Enumerate current IdP connector configs.",
		Usage:   "",
		Run:     runGetConnectorConfigs,
	}

	cmdSetConnectorConfigs = &command{
		Name:    "set-connector-configs",
		Summary: "Overwrite the current IdP connector configs with those from a local file.",
		Usage:   "<FILE>",
		Run:     runSetConnectorConfigs,
	}
)

func init() {
	commands = append(commands, cmdSetConnectorConfigs)
	commands = append(commands, cmdGetConnectorConfigs)
}

func runSetConnectorConfigs(args []string) int {
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

func runGetConnectorConfigs(args []string) int {
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

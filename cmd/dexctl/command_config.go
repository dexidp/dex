package main

import (
	"fmt"
	"io"
	"os"

	"encoding/json"
	"github.com/coreos/dex/connector"
	"github.com/spf13/cobra"
)

var (
	cmdGetConnectorConfigs = &cobra.Command{
		Use:     "get-connector-configs",
		Short:   "Enumerate current IdP connector configs.",
		Long:    "Enumerate current IdP connector configs.",
		Example: `  dexctl get-connector-configs --base-url=${OVER_LORD_URL} --api-key=${ADMIN_API_KEY}`,
		Run:     wrapRun(runGetConnectorConfigs),
	}

	cmdSetConnectorConfigs = &cobra.Command{
		Use:     "set-connector-configs",
		Short:   "Overwrite the current IdP connector configs with those from a local file. Provide the argument '-' to read from stdin.",
		Long:    "Overwrite the current IdP connector configs with those from a local file. Provide the argument '-' to read from stdin.",
		Example: `  dexctl set-connector-configs --base-url=${OVER_LORD_URL} --api-key=${ADMIN_API_KEY} ./static/fixtures/connectors.json.sample`,
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

	var r io.Reader
	if from := args[0]; from == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(from)
		if err != nil {
			stderr("Unable to open specified file: %v", err)
			return 1
		}
		defer f.Close()
		r = f
	}
	if useDBConnector, err := isDBURLPresent(); err != nil {
		stderr("Unable to configure dexctl : %v", err)
		os.Exit(1)
	} else {
		if useDBConnector {
			cfgs, err := connector.ReadConfigs(r)
			if err != nil {
				stderr("Unable to decode connector configs: %v", err)
				return 1
			}

			dbConnector, dbConnectorError := getDBConnector()
			if dbConnectorError != nil {
				stderr("Failed creating database connector: %v", dbConnectorError)
				return 1
			}
			if err := dbConnector.SetConnectorConfigs(cfgs); err != nil {
				stderr(err.Error())
				return 1
			}

			fmt.Printf("Saved %d connector config(s)\n", len(cfgs))
		} else {
			cfgs, err := readConfigs(r)
			if err != nil {
				stderr("Unable to decode connector configs: %v", err)
				return 1
			}
			adminAPIConnector, adminAPIConnectorError := getAdminAPIConnector()
			if adminAPIConnectorError != nil {
				stderr("Failed creating admin api client : %v", adminAPIConnectorError)
				return 1
			}

			if err := adminAPIConnector.SetConnectorConfigs(cfgs); err != nil {
				stderr(err.Error())
				return 1
			}

			fmt.Printf("Saved %d connector config(s)\n", len(cfgs))
		}
	}

	return 0
}

func runGetConnectorConfigs(cmd *cobra.Command, args []string) int {
	if len(args) != 0 {
		stderr("Provide zero arguments.")
		return 2
	}
	if useDBConnector, err := isDBURLPresent(); err != nil {
		stderr("Unable to configure dexctl : %v", err)
		os.Exit(1)
	} else {
		if useDBConnector {
			dbConnector, dbConnectorError := getDBConnector()
			if dbConnectorError != nil {
				stderr("Failed creating database connector: %v", dbConnectorError)
				return 1
			}
			cfgs, err := dbConnector.ConnectorConfigs()
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
		} else {
			adminAPIConnector, adminAPIConnectorError := getAdminAPIConnector()
			if adminAPIConnectorError != nil {
				stderr("Failed creating admin api client : %v", adminAPIConnectorError)
				return 1
			}
			cfgs, err := adminAPIConnector.ConnectorConfigs()
			if err != nil {
				stderr("Unable to retrieve connector configs: %v", err)
				return 1
			}

			fmt.Printf("Found %d connector config(s)\n", len(cfgs))
			for _, cfg := range cfgs {
				cfgMap := cfg.(map[string]interface{})
				fmt.Println()
				fmt.Printf("ID:   %v\n", cfgMap["id"])
				for configKey, configVal := range cfgMap {
					if configKey != "id" {
						fmt.Printf("%s: %s\n", configKey, configVal)
					}
				}
			}
		}
	}

	return 0
}

/**
Reads connector config as array of interfaces.
It is useful when using dex overload api client to add configs
*/
func readConfigs(r io.Reader) ([]interface{}, error) {
	var connectorConfigAsMap []map[string]interface{}
	if err := json.NewDecoder(r).Decode(&connectorConfigAsMap); err != nil {
		return nil, err
	}
	connectorConfigs := make([]interface{}, len(connectorConfigAsMap))
	for i, connectorConfig := range connectorConfigAsMap {

		connectorConfigs[i] = connectorConfig
	}
	return connectorConfigs, nil
}

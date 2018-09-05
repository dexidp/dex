// Copyright © 2018 Aaron Donovan <amdonov@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"

	"github.com/coreos/dex/cmd/dex/cmd"
	"github.com/coreos/dex/cmd/extend-example/connector/custom"
	"github.com/coreos/dex/server"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "extend-example",
	Short: "Shows how to add a connector to dex",
}

func main() {
	server.ConnectorsConfig["custom"] = func() server.ConnectorConfig { return new(custom.Config) }
	rootCmd.AddCommand(cmd.CommandServe())
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

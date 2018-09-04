// Copyright © 2017 Aaron Donovan <amdonov@gmail.com>
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

package cmd

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/theherk/viper"

	"github.com/amdonov/lite-idp/idp"

	"github.com/amdonov/lite-idp/saml"
	"github.com/spf13/cobra"
)

// serviceProviderCmd represents the serviceProvider command
var serviceProviderCmd = &cobra.Command{
	Use:   "service-provider metadata",
	Short: "add a service provider to the IdP",
	Long: `Parses the service provider's metadata to create an entry in the 
	configuration file.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		metadata, err := getReader(args[0])
		if err != nil {
			return err
		}
		defer metadata.Close()
		spMeta, err := readMetadata(metadata)
		if err != nil {
			return err
		}
		sp := idp.ConvertMetadata(spMeta)
		// Get the existing sps
		sps := []*idp.ServiceProvider{}
		if err = viper.UnmarshalKey("sps", &sps); err != nil {
			return err
		}
		found := false
		for i, client := range sps {
			if client.EntityID == sp.EntityID {
				sps[i] = sp
				found = true
				break
			}
		}
		if !found {
			sps = append(sps, sp)
		}
		viper.Set("sps", sps)
		return viper.WriteConfig()
	},
}

func readMetadata(metadata io.Reader) (*saml.SPEntityDescriptor, error) {
	decoder := xml.NewDecoder(metadata)
	sp := &saml.SPEntityDescriptor{}
	if err := decoder.Decode(sp); err != nil {
		return nil, err
	}
	return sp, nil
}

func getReader(fileOrUrl string) (io.ReadCloser, error) {
	url, err := url.Parse(fileOrUrl)
	if err != nil {
		return nil, err
	}
	if url.IsAbs() {
		resp, err := http.Get(fileOrUrl)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code, %d, when requesting metadata", resp.StatusCode)
		}
		return resp.Body, nil
	}
	// Just treat as a file
	return os.Open(fileOrUrl)
}

func init() {
	addCmd.AddCommand(serviceProviderCmd)
}

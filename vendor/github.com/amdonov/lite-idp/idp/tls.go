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

package idp

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"github.com/theherk/viper"
)

// ConfigureTLS not requiring users to present client certificates.
func ConfigureTLS() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(viper.GetString("tls-certificate"), viper.GetString("tls-private-key"))
	ca := viper.GetString("tls-ca")
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		//Some but not all operations will require a client cert
		ClientAuth: tls.VerifyClientCertIfGiven,
		MinVersion: tls.VersionTLS12,
	}
	if ca != "" {
		caCert, err := ioutil.ReadFile(ca)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
		tlsConfig.ClientCAs = caCertPool
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig, nil
}

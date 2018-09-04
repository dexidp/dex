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
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/amdonov/xmlsig"
)

func getCertFromRequest(r *http.Request) (*x509.Certificate, error) {
	if len(r.TLS.PeerCertificates) == 0 {
		return nil, errors.New("no certificate provided")
	}
	return r.TLS.PeerCertificates[0], nil
}

func getCertFromXML(cd *xmlsig.X509Data) (*x509.Certificate, error) {
	if cd == nil {
		return nil, errors.New("x509 data is nil")
	}
	certData, err := base64.StdEncoding.DecodeString(cd.X509Certificate)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certData)
}

func getSubjectDN(subject pkix.Name) string {
	rdns := []string{}
	names := subject.Names
	// Reverse the order
	for i := len(names) - 1; i >= 0; i-- {
		t := names[i].Type
		if len(t) == 4 && t[0] == 2 && t[1] == 5 && t[2] == 4 {
			var rdnName string
			switch t[3] {
			case 3:
				rdnName = "CN"
			case 6:
				rdnName = "C"
			case 7:
				rdnName = "L"
			case 8:
				rdnName = "ST"
			case 9:
				rdnName = "STREET"
			case 10:
				rdnName = "O"
			case 11:
				rdnName = "OU"
			default:
				panic("RFC 2253 implementation is incomplete")
			}
			rdnValue, _ := names[i].Value.(string)
			rdns = append(rdns, fmt.Sprintf("%s=%s", rdnName, rdnValue))
		}
	}
	return strings.Join(rdns, ", ")
}

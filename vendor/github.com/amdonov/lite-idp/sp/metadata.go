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

package sp

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"net/http"

	"github.com/amdonov/lite-idp/saml"
	"github.com/amdonov/xmlsig"
)

func (sp *serviceProvider) MetadataFunc() (http.HandlerFunc, error) {
	certData := sp.configuration.TLSConfig.Certificates[0].Certificate[0]
	// build EntityDescriptor
	ed := &saml.SPEntityDescriptor{
		EntityDescriptor: saml.EntityDescriptor{
			ID:       saml.NewID(),
			EntityID: sp.configuration.EntityID,
		},
		SPSSODescriptor: saml.SPSSODescriptor{
			AuthnRequestsSigned:        true,
			ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
			AssertionConsumerService: []saml.AssertionConsumerService{
				saml.AssertionConsumerService{
					IsDefault: true,
					Service: saml.Service{
						Binding:  "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Artifact",
						Location: sp.configuration.AssertionConsumerServiceURL,
					},
				},
			},
			KeyDescriptor: saml.KeyDescriptor{
				Use: "signing",
				KeyInfo: xmlsig.KeyInfo{
					X509Data: &xmlsig.X509Data{
						X509Certificate: base64.StdEncoding.EncodeToString(certData),
					},
				},
			},
		},
	}
	sig, err := sp.signer.CreateSignature(ed)
	if err != nil {
		return nil, err
	}
	ed.Signature = sig

	// save it into byte slice
	var b bytes.Buffer
	b.Write([]byte(xml.Header))
	encoder := xml.NewEncoder(&b)
	err = encoder.Encode(ed)
	if err != nil {
		return nil, err
	}
	metadata := b.Bytes()

	// return handler
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write(metadata)
	}, nil
}

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
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"net/http"

	"github.com/amdonov/lite-idp/saml"
	"github.com/amdonov/xmlsig"
)

func (i *IDP) DefaultMetadataHandler() (http.HandlerFunc, error) {
	certData := i.TLSConfig.Certificates[0].Certificate[0]
	keyDescriptor := saml.KeyDescriptor{
		Use: "signing",
		KeyInfo: xmlsig.KeyInfo{
			X509Data: &xmlsig.X509Data{
				X509Certificate: base64.StdEncoding.EncodeToString(certData),
			},
		},
	}

	// build EntityDescriptor
	ed := &saml.IDPEntityDescriptor{
		EntityDescriptor: saml.EntityDescriptor{
			ID:       saml.NewID(),
			EntityID: i.entityID,
		},
		IDPSSODescriptor: saml.IDPSSODescriptor{
			ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
			KeyDescriptor:              keyDescriptor,
			WantAuthnRequestsSigned:    true,
			ArtifactResolutionService: saml.ArtifactResolutionService{
				Service: saml.Service{
					Binding:  "urn:oasis:names:tc:SAML:2.0:bindings:SOAP",
					Location: i.artifactResolutionServiceLocation,
				},
				Index: 1,
			},
			NameIDFormat: "urn:oasis:names:tc:SAML:1.1:nameid-format:X509SubjectName",
			SingleSignOnService: saml.SingleSignOnService{
				Service: saml.Service{
					Binding:  "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect",
					Location: i.singleSignOnServiceLocation,
				},
			},
		},
		AttributeAuthorityDescriptor: saml.AttributeAuthorityDescriptor{
			ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
			KeyDescriptor:              keyDescriptor,
			AttributeService: saml.AttributeService{
				Service: saml.Service{
					Binding:  "urn:oasis:names:tc:SAML:2.0:bindings:SOAP",
					Location: i.attributeServiceLocation,
				},
			},
			NameIDFormat: "urn:oasis:names:tc:SAML:1.1:nameid-format:X509SubjectName",
		},
	}
	sig, err := i.signer.CreateSignature(ed)
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

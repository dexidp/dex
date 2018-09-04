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

package saml

import (
	"encoding/xml"

	"github.com/amdonov/xmlsig"
)

type EntityDescriptor struct {
	XMLName   xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:metadata EntityDescriptor"`
	ID        string   `xml:",attr"`
	EntityID  string   `xml:"entityID,attr"`
	Signature *xmlsig.Signature
}

type SPEntityDescriptor struct {
	EntityDescriptor
	SPSSODescriptor SPSSODescriptor
}

type IDPEntityDescriptor struct {
	EntityDescriptor
	IDPSSODescriptor             IDPSSODescriptor
	AttributeAuthorityDescriptor AttributeAuthorityDescriptor
}

type IDPSSODescriptor struct {
	XMLName                    xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:metadata IDPSSODescriptor"`
	ProtocolSupportEnumeration string   `xml:"protocolSupportEnumeration,attr"`
	WantAuthnRequestsSigned    bool     `xml:",attr"`
	KeyDescriptor              KeyDescriptor
	ArtifactResolutionService  ArtifactResolutionService
	NameIDFormat               string `xml:"NameIDFormat"`
	SingleSignOnService        SingleSignOnService
}

type Service struct {
	Binding  string `xml:",attr"`
	Location string `xml:",attr"`
}

type AttributeService struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:metadata AttributeService"`
	Service
}

type AttributeAuthorityDescriptor struct {
	XMLName                    xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:metadata AttributeAuthorityDescriptor"`
	ProtocolSupportEnumeration string   `xml:"protocolSupportEnumeration,attr"`
	KeyDescriptor              KeyDescriptor
	AttributeService           AttributeService
	NameIDFormat               string `xml:"NameIDFormat"`
}

type SingleSignOnService struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:metadata SingleSignOnService"`
	Service
}

type ArtifactResolutionService struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:metadata ArtifactResolutionService"`
	Service
	Index uint `xml:"index,attr"`
}

type SPSSODescriptor struct {
	XMLName                    xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:metadata SPSSODescriptor"`
	AuthnRequestsSigned        bool     `xml:",attr"`
	WantAssertionsSigned       bool     `xml:",attr"`
	ProtocolSupportEnumeration string   `xml:"protocolSupportEnumeration,attr"`
	AssertionConsumerService   []AssertionConsumerService
	KeyDescriptor              KeyDescriptor
}

type AssertionConsumerService struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:metadata AssertionConsumerService"`
	Service
	IsDefault bool   `xml:"isDefault,attr"`
	Index     uint32 `xml:"index,attr"`
}

type KeyDescriptor struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:metadata KeyDescriptor"`
	Use     string   `xml:"use,attr,omitempty"`
	KeyInfo xmlsig.KeyInfo
}

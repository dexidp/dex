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
	"github.com/amdonov/lite-idp/saml"
)

//ServiceProvider stores the Service Provider metadata required by the IdP
type ServiceProvider struct {
	EntityID                  string
	AssertionConsumerServices []AssertionConsumerService
	Certificate               string
	// Could be an RSA or DSA public key
	publicKey interface{}
}

type AssertionConsumerService struct {
	Index     uint32
	IsDefault bool
	Binding   string
	Location  string
}

func ConvertMetadata(spMeta *saml.SPEntityDescriptor) *ServiceProvider {
	sp := &ServiceProvider{
		Certificate: spMeta.SPSSODescriptor.KeyDescriptor.KeyInfo.X509Data.X509Certificate,
		EntityID:    spMeta.EntityDescriptor.EntityID,
	}
	sp.AssertionConsumerServices = make([]AssertionConsumerService, len(spMeta.SPSSODescriptor.AssertionConsumerService))
	for i, val := range spMeta.SPSSODescriptor.AssertionConsumerService {
		sp.AssertionConsumerServices[i] = AssertionConsumerService{
			Index:     val.Index,
			IsDefault: val.IsDefault,
			Binding:   val.Binding,
			Location:  val.Location,
		}
	}
	return sp
}

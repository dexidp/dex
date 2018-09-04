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

// The IdP needs to hold some state information either in memory or in an
// external store such as redis. In benchmarks, protocol
// buffers were much faster and smaller than json or gob marshalling,
// and they used far less space.

//go:generate protoc model.proto --go_out=.

package model

import (
	"github.com/amdonov/lite-idp/saml"
	"github.com/golang/protobuf/ptypes"
)

func (u *User) AppendAttributes(atts []*Attribute) {
	if u.Attributes == nil {
		u.Attributes = atts
		return
	}
	u.Attributes = append(u.Attributes, atts...)
}

func (u *User) AttributeStatement() *saml.AttributeStatement {
	if u.Attributes == nil {
		return nil
	}
	stmt := &saml.AttributeStatement{}
	for _, val := range u.Attributes {
		attVals := make([]saml.AttributeValue, len(val.Value))
		for i := range val.Value {
			attVals[i] = saml.AttributeValue{Value: val.Value[i]}
		}
		att := saml.Attribute{
			FriendlyName:   val.Name,
			Name:           val.Name,
			NameFormat:     "urn:oasis:names:tc:SAML:2.0:attrname-format:basic",
			AttributeValue: attVals,
		}
		stmt.Attribute = append(stmt.Attribute, att)
	}
	return stmt
}

func NewAuthnRequest(src *saml.AuthnRequest, relayState string) (*AuthnRequest, error) {
	t, err := ptypes.TimestampProto(src.IssueInstant)
	if err != nil {
		return nil, err
	}
	return &AuthnRequest{
		AssertionConsumerServiceURL:   src.AssertionConsumerServiceURL,
		AssertionConsumerServiceIndex: src.AssertionConsumerServiceIndex,
		Destination:                   src.Destination,
		ID:                            src.ID,
		ProtocolBinding:               src.ProtocolBinding,
		RelayState:                    relayState,
		IssueInstant:                  t,
		Issuer:                        src.Issuer,
	}, nil
}

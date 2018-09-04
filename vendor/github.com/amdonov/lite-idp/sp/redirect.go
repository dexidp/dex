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
	"compress/flate"
	"encoding/base64"
	"net/url"
	"time"

	"github.com/amdonov/lite-idp/saml"
	"github.com/google/uuid"
)

// GetRedirect stores the provided state and returns a URL suitable for redirecting the SAML IdP (redirect binding)
func (sp *serviceProvider) GetRedirect(state []byte) (string, error) {
	data := struct {
		ID       string
		Time     string
		IDP      string
		ACSURL   string
		EntityID string
	}{
		saml.NewID(),
		time.Now().Format(time.RFC3339),
		sp.configuration.IDPRedirectEndpoint,
		sp.configuration.AssertionConsumerServiceURL,
		sp.configuration.EntityID,
	}
	var b bytes.Buffer
	writer, err := flate.NewWriter(&b, flate.DefaultCompression)
	if err != nil {
		return "", err
	}
	err = sp.requestTemplate.Execute(writer, data)
	if err != nil {
		return "", err
	}
	err = writer.Flush()
	if err != nil {
		return "", err
	}

	// Don't need to encode this - is OK
	stateID := uuid.New().String()

	if sp.stateCache == nil {
		// Caller doesn't want state managed by the SP
		// Directly pass along the state provided
		stateID = url.QueryEscape(string(state))
	} else {
		// Store the provided state
		sp.stateCache.Set(stateID, state)
	}
	samlRequest := url.QueryEscape(base64.StdEncoding.EncodeToString(b.Bytes()))
	requestToSign := "SAMLRequest=" + samlRequest + "&RelayState=" + stateID + "&SigAlg=" + url.QueryEscape(sp.signer.Algorithm())

	signature, err := sp.signer.Sign([]byte(requestToSign))
	if err != nil {
		return "", err
	}
	query := requestToSign + "&Signature=" + url.QueryEscape(signature)
	return sp.configuration.IDPRedirectEndpoint + "?" + query, err
}

const requestTemplate = `<?xml version="1.0"?><samlp:AuthnRequest ID="{{ .ID }}" Version="2.0" IssueInstant="{{ .Time }}" ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Artifact" AssertionConsumerServiceURL="{{ .ACSURL }}" Destination="{{ .IDP }}" xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"><saml:Issuer>{{ .EntityID }}</saml:Issuer></samlp:AuthnRequest>`

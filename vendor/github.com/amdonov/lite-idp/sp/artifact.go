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
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/amdonov/lite-idp/saml"
	log "github.com/sirupsen/logrus"
)

type ArtifactCallback func(w http.ResponseWriter, r *http.Request, state []byte, assertion *saml.Assertion)

func (sp *serviceProvider) ArtifactFunc(callback ArtifactCallback) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// retrieve the relayState from our cache
		stateID := r.Form.Get("RelayState")
		if stateID == "" {
			log.Info("received a request without RelayState")
			http.Error(w, "identity provider did not return RelayState", http.StatusUnauthorized)
			return
		}
		var state []byte
		if sp.stateCache == nil {
			state = []byte(stateID)
		} else {
			state, err = sp.stateCache.Get(stateID)
			if err != nil {
				log.Info("RelayState not found in cache")
				http.Error(w, "identity provider did not return RelayState", http.StatusUnauthorized)
				return
			}
		}

		// call the IdP to get the SAML assertion
		artifact := r.Form.Get("SAMLart")

		assertion, err := sp.resolveArtifact(artifact)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// allow the application to write the response
		callback(w, r, state, assertion)
	}
}

func (sp *serviceProvider) resolveArtifact(artifact string) (*saml.Assertion, error) {
	request, err := sp.buildResolveRequest(artifact)
	if err != nil {
		return nil, err
	}
	post, err := http.NewRequest(http.MethodPost, sp.configuration.IDPArtifactEndpoint, request)
	if err != nil {
		return nil, err
	}
	post.Header.Add("Content-Type", "text/xml")
	post.Header.Add("SOAPAction", "http://www.oasis-open.org/committees/security")
	resp, err := sp.client.Do(post)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from artifact resolve request %d", resp.StatusCode)
	}
	decoder := xml.NewDecoder(resp.Body)
	response := &saml.ArtifactResponseEnvelope{}
	err = decoder.Decode(response)
	if err != nil {
		return nil, err
	}
	assertion := response.Body.ArtifactResponse.Response.Assertion
	if assertion == nil {
		// TODO check the rest of the response for an error
		// Write it out for now until we know what we're looking
		encoder := xml.NewEncoder(os.Stderr)
		encoder.Encode(response)
		return nil, errors.New("check logs assertion was nil")
	}
	assertion.RawXML = response.Body.ArtifactResponse.Response.RawAssertion
	return assertion, nil
}

func (sp *serviceProvider) buildResolveRequest(artifact string) (io.Reader, error) {
	resolve := saml.ArtifactResolveEnvelope{
		Body: saml.ArtifactResolveBody{
			ArtifactResolve: saml.ArtifactResolve{
				RequestAbstractType: saml.RequestAbstractType{
					ID:           saml.NewID(),
					IssueInstant: time.Now(),
					Issuer:       sp.configuration.EntityID,
					Version:      "2.0",
				},
				Artifact: artifact,
			},
		},
	}
	signature, err := sp.signer.CreateSignature(resolve.Body.ArtifactResolve)
	if err != nil {
		return nil, err
	}
	resolve.Body.ArtifactResolve.Signature = signature
	var b bytes.Buffer
	encoder := xml.NewEncoder(&b)
	err = encoder.Encode(resolve)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b.Bytes()), nil
}

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
	"encoding/xml"
	"net/http"
	"net/url"
	"time"

	"github.com/amdonov/lite-idp/model"
	"github.com/amdonov/lite-idp/saml"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
)

func (i *IDP) DefaultArtifactResolveHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// We require transport authentication rather than message authentication
		tlsCert, err := getCertFromRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		log.Infof("received artifact resolution request from %s", getSubjectDN(tlsCert.Subject))
		decoder := xml.NewDecoder(r.Body)
		var resolveEnv saml.ArtifactResolveEnvelope
		err = decoder.Decode(&resolveEnv)
		// TODO confirm appropriate error response for this service
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		artifact := resolveEnv.Body.ArtifactResolve.Artifact
		data, err := i.TempCache.Get(artifact)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		artifactResponse := &model.ArtifactResponse{}
		err = proto.Unmarshal(data, artifactResponse)
		// TODO confirm appropriate error response for this service
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response := i.makeResponse(artifactResponse.Request, artifactResponse.User)
		artResponseEnv := saml.ArtifactResponseEnvelope{}
		artResponse := &artResponseEnv.Body.ArtifactResponse
		artResponse.ID = saml.NewID()
		now := time.Now()
		artResponse.IssueInstant = now
		artResponse.InResponseTo = resolveEnv.Body.ArtifactResolve.ID
		artResponse.Version = "2.0"
		artResponse.Issuer = &saml.Issuer{
			Value: i.entityID,
		}
		artResponse.Status = &saml.Status{
			StatusCode: saml.StatusCode{
				Value: "urn:oasis:names:tc:SAML:2.0:status:Success",
			},
		}
		artResponse.Response = *response

		signature, err := i.signer.CreateSignature(response.Assertion)
		// TODO confirm appropriate error response for this service
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response.Assertion.Signature = signature
		// TODO handle these errors. Probably can't do anything besides log, as we've already started to write the
		// response.
		_, err = w.Write([]byte(xml.Header))
		encoder := xml.NewEncoder(w)
		err = encoder.Encode(artResponseEnv)
		err = encoder.Flush()
	}
}

func (i *IDP) sendArtifactResponse(authRequest *model.AuthnRequest, user *model.User,
	w http.ResponseWriter, r *http.Request) error {
	target, err := url.Parse(authRequest.AssertionConsumerServiceURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	parameters := url.Values{}
	artifact := getArtifact(i.entityID)
	// Store required data in the cache
	response := &model.ArtifactResponse{
		User:    user,
		Request: authRequest,
	}
	data, err := proto.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	i.TempCache.Set(artifact, data)
	parameters.Add("SAMLart", artifact)
	parameters.Add("RelayState", authRequest.RelayState)
	target.RawQuery = parameters.Encode()
	// Don't send temporary redirect. We don't want the post resent
	http.Redirect(w, r, target.String(), http.StatusFound)

	return nil
}

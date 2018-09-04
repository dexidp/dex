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
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/amdonov/lite-idp/model"
	"github.com/amdonov/lite-idp/saml"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
)

func (i *IDP) respond(authRequest *model.AuthnRequest, user *model.User,
	w http.ResponseWriter, r *http.Request) error {
	// Save user information and set session cookie
	data, err := proto.Marshal(user)
	if err != nil {
		return err
	}
	session := uuid.New().String()
	err = i.UserCache.Set(session, data)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     i.cookieName,
		Path:     "/",
		Value:    session,
		Secure:   true,
		HttpOnly: true,
	})
	switch authRequest.ProtocolBinding {
	case "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Artifact":
		return i.sendArtifactResponse(authRequest, user, w, r)
	case "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST":
		return i.sendPostResponse(authRequest, user, w, r)
	default:
		return errors.New("unsupported protocol binding")
	}
}

func (i *IDP) makeResponse(authRequest *model.AuthnRequest, user *model.User) *saml.Response {
	now := time.Now()
	fiveFromNow := now.Add(5 * time.Minute)
	s := &saml.Response{
		StatusResponseType: saml.StatusResponseType{
			Version:      "2.0",
			ID:           saml.NewID(),
			IssueInstant: now,
			Status: &saml.Status{
				StatusCode: saml.StatusCode{
					Value: "urn:oasis:names:tc:SAML:2.0:status:Success",
				},
			},
			InResponseTo: authRequest.ID,
			Issuer:       saml.NewIssuer(i.entityID),
		},
		Assertion: &saml.Assertion{
			ID:           saml.NewID(),
			IssueInstant: now,
			Issuer:       saml.NewIssuer(i.entityID),
			Version:      "2.0",
			Subject: &saml.Subject{
				NameID: &saml.NameID{
					Format:          user.Format,
					NameQualifier:   i.entityID,
					SPNameQualifier: authRequest.Issuer,
					Value:           user.Name,
				},
				SubjectConfirmation: &saml.SubjectConfirmation{
					Method: "urn:oasis:names:tc:SAML:2.0:cm:bearer",
					SubjectConfirmationData: &saml.SubjectConfirmationData{
						Address:      net.ParseIP(user.IP),
						InResponseTo: authRequest.ID,
						Recipient:    authRequest.AssertionConsumerServiceURL,
						NotOnOrAfter: fiveFromNow,
					},
				},
			},
			AuthnStatement: &saml.AuthnStatement{
				AuthnInstant: now,
				SessionIndex: saml.NewID(),
				SubjectLocality: &saml.SubjectLocality{
					DNSName: i.serverName,
				},
				AuthnContext: &saml.AuthnContext{
					AuthnContextClassRef: user.Context,
				},
			},
			AttributeStatement: user.AttributeStatement(),
			Conditions: &saml.Conditions{
				NotOnOrAfter: fiveFromNow,
				NotBefore:    now,
				AudienceRestriction: &saml.AudienceRestriction{
					Audience: authRequest.Issuer,
				},
			},
		},
	}
	return s
}

func getArtifact(entityId string) string {
	// The artifact isn't just a random session id. It's a base64-encoded byte array
	// that's 44 bytes in length. The first two bytes must be 04 for SAML 2. The second
	// two bytes are the index of the artifact resolution endpoint in the IdP metadata. Something like 02
	// The next 20 bytes are the sha1 hash of the IdP's entity ID
	// The last 20 bytes are unique to the request
	artifact := make([]byte, 44)
	// Use SAML 2
	artifact[1] = byte(4)
	// Index 1
	artifact[3] = byte(1)
	// Hash of entity ID
	source := sha1.Sum([]byte(entityId))
	for i := 4; i < 24; i++ {
		artifact[i] = source[i-4]
	}
	// Message ID
	message := sha1.Sum([]byte(uuid.New().String()))
	for i := 24; i < 44; i++ {
		artifact[i] = message[i-24]
	}
	return base64.StdEncoding.EncodeToString(artifact)
}

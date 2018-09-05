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
	"crypto/tls"
	"net/http"
	"text/template"
	"time"

	"github.com/amdonov/lite-idp/store"
	"github.com/amdonov/xmlsig"
)

type ServiceProvider interface {
	GetRedirect([]byte) (string, error)
	MetadataFunc() (http.HandlerFunc, error)
	ArtifactFunc(callback ArtifactCallback) http.HandlerFunc
}

type Configuration struct {
	EntityID                    string
	AssertionConsumerServiceURL string
	IDPRedirectEndpoint         string
	IDPArtifactEndpoint         string
	// Optional override of client added for testing
	// but may have other uses
	Client    *http.Client
	Timeout   time.Duration
	TLSConfig *tls.Config
	Cache     store.Cache
}

func New(conf Configuration) (ServiceProvider, error) {
	templ, err := template.New("redirect").Parse(requestTemplate)
	if err != nil {
		return nil, err
	}
	cert := conf.TLSConfig.Certificates[0]

	signer, err := xmlsig.NewSigner(cert)
	if err != nil {
		return nil, err
	}
	client := conf.Client
	if client == nil {
		client = &http.Client{
			Timeout: conf.Timeout,
			Transport: &http.Transport{
				TLSClientConfig: conf.TLSConfig,
			}}
	}
	serviceProvider := &serviceProvider{
		configuration:   conf,
		requestTemplate: templ,
		signer:          signer,
		client:          client,
		stateCache:      conf.Cache,
	}

	return serviceProvider, nil
}

type serviceProvider struct {
	configuration   Configuration
	requestTemplate *template.Template
	signer          xmlsig.Signer
	client          *http.Client
	stateCache      store.Cache
}

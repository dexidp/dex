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
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"text/template"

	"github.com/amdonov/lite-idp/model"
	"github.com/amdonov/lite-idp/store"
	"github.com/amdonov/lite-idp/ui"
	"github.com/amdonov/xmlsig"
	"github.com/julienschmidt/httprouter"
	"github.com/theherk/viper"
)

type IDP struct {
	// You can include other routes by providing a router or
	// one will be created. Alternatively, you can add routes and
	// middleware to the Handler
	Router *httprouter.Router
	// Short term cache for saving state during authentication
	TempCache store.Cache
	// Longer term cache of authenticated users
	UserCache              store.Cache
	TLSConfig              *tls.Config
	PasswordValidator      PasswordValidator
	AttributeSources       []AttributeSource
	MetadataHandler        http.HandlerFunc
	ArtifactResolveHandler http.HandlerFunc
	RedirectSSOHandler     http.HandlerFunc
	PasswordLoginHandler   http.HandlerFunc

	handler http.Handler
	signer  xmlsig.Signer

	// properties set or derived from configuration settings
	cookieName                        string
	serverName                        string
	entityID                          string
	artifactResolutionServiceLocation string
	attributeServiceLocation          string
	singleSignOnServiceLocation       string
	postTemplate                      *template.Template
	sps                               map[string]ServiceProvider
}

func (i *IDP) Handler() (http.Handler, error) {
	if i.handler == nil {
		if err := i.configureConstants(); err != nil {
			return nil, err
		}
		if err := i.configureSPs(); err != nil {
			return nil, err
		}
		if err := i.configureCrypto(); err != nil {
			return nil, err
		}
		if err := i.configureStores(); err != nil {
			return nil, err
		}
		if err := i.configureValidator(); err != nil {
			return nil, err
		}
		if err := i.configureAttributeSources(); err != nil {
			return nil, err
		}
		if err := i.buildRoutes(); err != nil {
			return nil, err
		}
		i.handler = i.Router
	}
	return i.handler, nil
}

func (i *IDP) configureConstants() error {
	templ, err := template.New("post").Parse(postTemplate)
	if err != nil {
		return err
	}
	i.postTemplate = templ
	i.cookieName = viper.GetString("cookie-name")
	serverName := viper.GetString("server-name")
	i.entityID = viper.GetString("entity-id")
	if i.entityID == "" {
		i.entityID = fmt.Sprintf("https://%s/", serverName)
	}
	i.serverName = serverName
	i.artifactResolutionServiceLocation = fmt.Sprintf("https://%s%s", serverName, viper.GetString("artifact-service-path"))
	i.attributeServiceLocation = fmt.Sprintf("https://%s%s", serverName, viper.GetString("attribute-service-path"))
	i.singleSignOnServiceLocation = fmt.Sprintf("https://%s%s", serverName, viper.GetString("sso-service-path"))
	return nil
}

func (i *IDP) configureSPs() error {
	sps := []ServiceProvider{}
	if err := viper.UnmarshalKey("sps", &sps); err != nil {
		return err
	}
	i.sps = make(map[string]ServiceProvider, len(sps))
	for j, sp := range sps {
		block, err := base64.StdEncoding.DecodeString(sp.Certificate)
		if err != nil {
			return errors.New("failed to parse PEM block containing the public key")
		}
		cert, err := x509.ParseCertificate(block)
		if err != nil {
			return errors.New("failed to parse certificate: " + err.Error())
		}
		sps[j].publicKey = cert.PublicKey
		i.sps[sp.EntityID] = sps[j]
	}

	return nil
}

func (i *IDP) configureCrypto() error {
	if i.TLSConfig == nil {
		tlsConfig, err := ConfigureTLS()
		if err != nil {
			return err
		}
		i.TLSConfig = tlsConfig
	}
	if len(i.TLSConfig.Certificates) == 0 {
		return errors.New("tlsConfig does not contain a certificate")
	}
	cert := i.TLSConfig.Certificates[0]
	signer, err := xmlsig.NewSignerWithOptions(cert, xmlsig.SignerOptions{
		SignatureAlgorithm: viper.GetString("signature-algoritm"),
		DigestAlgorithm:    viper.GetString("digest-algorithm"),
	})
	i.signer = signer
	return err
}

func (i *IDP) configureStores() error {
	if i.TempCache == nil {
		cache, err := store.New(viper.GetDuration("temp-cache-duration"))
		if err != nil {
			return err
		}
		i.TempCache = cache
	}
	if i.UserCache == nil {
		cache, err := store.New(viper.GetDuration("user-cache-duration"))
		if err != nil {
			return err
		}
		i.UserCache = cache
	}
	return nil
}

func (i *IDP) configureValidator() error {
	if i.PasswordValidator == nil {
		validator, err := NewValidator()
		if err != nil {
			return err
		}
		i.PasswordValidator = validator
	}
	return nil
}

func (i *IDP) configureAttributeSources() error {
	if i.AttributeSources == nil {
		source, err := NewAttributeSource()
		if err != nil {
			return err
		}
		i.AttributeSources = []AttributeSource{source}
	}
	return nil
}

func (i *IDP) buildRoutes() error {
	if i.Router == nil {
		i.Router = httprouter.New()
	}
	r := i.Router

	// Handle requests for metadata
	if i.MetadataHandler == nil {
		metadata, err := i.DefaultMetadataHandler()
		if err != nil {
			return err
		}
		i.MetadataHandler = metadata
	}
	r.HandlerFunc("GET", viper.GetString("metadata-path"), i.MetadataHandler)

	// Handle artifact resolution
	if i.ArtifactResolveHandler == nil {
		i.ArtifactResolveHandler = i.DefaultArtifactResolveHandler()
	}
	r.HandlerFunc("POST", viper.GetString("artifact-service-path"), i.ArtifactResolveHandler)

	// Handle redirect SSO requests
	if i.RedirectSSOHandler == nil {
		i.RedirectSSOHandler = i.DefaultRedirectSSOHandler()
	}
	r.HandlerFunc("GET", viper.GetString("sso-service-path"), i.RedirectSSOHandler)

	// Handle password logins
	if i.PasswordLoginHandler == nil {
		i.PasswordLoginHandler = i.DefaultPasswordLoginHandler()
	}
	r.HandlerFunc("POST", "/ui/login.html", i.PasswordLoginHandler)

	// Serve up UI
	userInterface := ui.UI()
	r.Handler("GET", "/ui/*path", userInterface)
	r.Handler("GET", "/favicon.ico", userInterface)

	return nil
}

func getIP(request *http.Request) net.IP {
	addr := request.RemoteAddr
	if strings.Contains(addr, ":") {
		addr = strings.Split(addr, ":")[0]
	}
	return net.ParseIP(addr)
}

func (i *IDP) setUserAttributes(user *model.User) error {
	for _, source := range i.AttributeSources {
		if err := source.AddAttributes(user); err != nil {
			return err
		}
	}
	return nil
}

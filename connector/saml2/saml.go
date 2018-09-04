// Package saml2 implements strategies for authenticating using the SAML ArtifactBinding.
package saml2

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/amdonov/lite-idp/saml"
	"github.com/amdonov/lite-idp/sp"
	"github.com/coreos/dex/connector"
	"github.com/sirupsen/logrus"
)

// Config holds the configuration parameters for the SAML2 connector
//
// An example config:
//	config:
//		entityID: http://127.0.0.1:5556/dex/
//		assertionConsumerServiceURL: http://127.0.0.1:5556/dex/callback
//		redirectEndpoint: https://idp/SAML2/Redirect/SSO
//		artifactEndpoint: https://idp/SAML2/SOAP/ArtifactResolution
//		certificate: cert.pem
//		key: key.pem
//		ca: ca.crt
//		metadataAddress: 127.0.0.1:10000
//		emailAttr: email
//		nameAttr: name
//
type Config struct {
	EntityID                    string        `json:"entityID"`
	AssertionConsumerServiceURL string        `json:"assertionConsumerServiceURL"`
	IDPArtifactEndpoint         string        `json:"artifactEndpoint"`
	IDPRedirectEndpoint         string        `json:"redirectEndpoint"`
	Timeout                     time.Duration `json:"timeout"`
	Certificate                 string        `json:"certificate"`
	Key                         string        `json:"key"`
	CA                          string        `json:"ca"`
	MetadataAddress             string        `json:"metadataAddress"`
	EmailAttr                   string        `json:"emailAttr"`
	NameAttr                    string        `json:"nameAttr"`
}

// Open validates the config and returns a connector. It does not actually
// validate connectivity with the provider.
func (c *Config) Open(id string, logger logrus.FieldLogger) (connector.Connector, error) {
	requiredFields := []struct {
		name string
		val  string
	}{
		{"entityID", c.EntityID},
		{"assertionConsumerServiceURL", c.AssertionConsumerServiceURL},
		{"artifactEndpoint", c.IDPArtifactEndpoint},
		{"redirectEndpoint", c.IDPRedirectEndpoint},
		{"certificate", c.Certificate},
		{"key", c.Key},
		{"emailAttr", c.EmailAttr},
		{"nameAttr", c.NameAttr},
	}

	for _, field := range requiredFields {
		if field.val == "" {
			return nil, fmt.Errorf("saml2: missing required field %q", field.name)
		}
	}
	tlsConfigClient, err := configureTLS(c)
	if err != nil {
		return nil, err
	}
	serviceProvider, err := sp.New(sp.Configuration{
		EntityID:                    c.EntityID,
		AssertionConsumerServiceURL: c.AssertionConsumerServiceURL,
		IDPArtifactEndpoint:         c.IDPArtifactEndpoint,
		IDPRedirectEndpoint:         c.IDPRedirectEndpoint,
		Timeout:                     c.Timeout,
		TLSConfig:                   tlsConfigClient,
	})
	if c.MetadataAddress != "" {
		metadata, err := serviceProvider.MetadataFunc()
		if err != nil {
			return nil, err
		}
		logger.Info("saml metadata available at https://", c.MetadataAddress)
		go func() {
			logger.Fatal(http.ListenAndServeTLS(c.MetadataAddress, c.Certificate, c.Key, metadata))
		}()
	}
	return &samlConnector{c, serviceProvider}, nil
}

type samlConnector struct {
	config          *Config
	serviceProvider sp.ServiceProvider
}

// LoginURL uses the SAML RedirectBinding to make a request to the IdP
func (c *samlConnector) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
	return c.serviceProvider.GetRedirect([]byte(state))
}

// HandleBack implements the SAML AssertionConsumer service for the ArtifactBinding
// It's very similar to an oauth2 token exchange and uses the provided to ArtifactID to
// retrieve a SAML assertion directly from the IdP. It's much safer than a POST binding because
// Assertion is not retrieved via the user. In addition, It's easier for non-browser clients because
// they just have to follow redirects and don't need to post a form.
func (c *samlConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	// SP client library wants to return the response on error and call our callback on success
	// dex doesn't make the response writer available
	// buffer the output and if callback isn't called raise an error
	success := false
	ab := &artfactBuffer{}
	c.serviceProvider.ArtifactFunc(func(w http.ResponseWriter, r *http.Request, state []byte, assertion *saml.Assertion) {
		atts := make(map[string][]saml.AttributeValue)
		if assertion.AttributeStatement != nil {
			for att := range assertion.AttributeStatement.Attribute {
				atts[assertion.AttributeStatement.Attribute[att].Name] = assertion.AttributeStatement.Attribute[att].AttributeValue
			}
		}
		getAttribute := func(name string) string {
			if val, ok := atts[name]; ok && len(val) > 0 {
				return val[0].Value
			}
			return ""
		}
		identity = connector.Identity{
			Username:      getAttribute(c.config.NameAttr),
			UserID:        assertion.Subject.NameID.Value,
			Email:         getAttribute(c.config.EmailAttr),
			EmailVerified: true,
		}
		success = true
	}).ServeHTTP(ab, r)
	if !success {
		return identity, errors.New(ab.buffer.String())
	}
	return identity, nil
}

// create TLS Configuration used to present client certificates
// and optionally trust an alternate CA
func configureTLS(c *Config) (*tls.Config, error) {

	cert, err := tls.LoadX509KeyPair(c.Certificate, c.Key)
	ca := c.CA
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	if ca != "" {
		caCert, err := ioutil.ReadFile(ca)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig, nil
}

type artfactBuffer struct {
	buffer bytes.Buffer
}

func (*artfactBuffer) Header() http.Header {
	return make(http.Header)
}

func (ab *artfactBuffer) Write(data []byte) (int, error) {
	return ab.buffer.Write(data)
}

func (*artfactBuffer) WriteHeader(int) {}

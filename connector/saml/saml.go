package saml

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Calpicow/go-saml"
	"github.com/Sirupsen/logrus"
	"github.com/coreos/dex/connector"
)

// Config holds configuration options for SAML
type Config struct {
	IssuerID              string `json:"issuerID"`
	SignRequest           bool   `json:"signRequest"`
	NameIDPolicyFormat    string `json:"nameIDPolicyFormat"`
	RequestedAuthnContext string `json:"requestedAuthnContext"`
	PublicCertPath        string `json:"publicCertPath"`
	PrivateKeyPath        string `json:"privateKeyPath"`
	SSOUrl                string `json:"ssoURL"`
	IDPPublicCertPath     string `json:"idpPublicCertPath"`
	IDAttr                string `json:"idAttr"`
	UserAttr              string `json:"userAttr"`
	EmailAttr             string `json:"emailAttr"`
	GroupsAttr            string `json:"groupsAttr"`
	GroupsDelimiter       string `json:"groupsDelimiter"`
	RedirectURI           string `json:"redirectURI"`
}

// Open returns a connector which can be used to login users through an upstream
// SAML provider
func (c *Config) Open(logger logrus.FieldLogger) (conn connector.Connector, err error) {
	// Configure the app and account settings
	sp := saml.ServiceProviderSettings{
		PublicCertPath:              c.PublicCertPath,
		PrivateKeyPath:              c.PrivateKeyPath,
		IDPSSOURL:                   c.SSOUrl,
		IDPSSODescriptorURL:         c.IssuerID,
		IDPPublicCertPath:           c.IDPPublicCertPath,
		AssertionConsumerServiceURL: c.RedirectURI,
		SPSignRequest:               c.SignRequest,
	}

	err = sp.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %v", err)
	}

	idpc := &samlConnector{
		spSettings:              sp,
		spNameIDPolicyFormat:    c.NameIDPolicyFormat,
		spRequestedAuthnContext: c.RequestedAuthnContext,
		idAttr:                  c.IDAttr,
		userAttr:                c.UserAttr,
		emailAttr:               c.EmailAttr,
		groupsAttr:              c.GroupsAttr,
		groupsDelimiter:         c.GroupsDelimiter,
		logger:                  logger,
	}
	return idpc, nil
}

var (
	_ connector.CallbackConnector = (*samlConnector)(nil)
)

type samlConnector struct {
	spSettings              saml.ServiceProviderSettings
	spNameIDPolicyFormat    string
	spRequestedAuthnContext string
	idAttr                  string
	userAttr                string
	emailAttr               string
	groupsAttr              string
	groupsDelimiter         string
	logger                  logrus.FieldLogger
}

func (c *samlConnector) Close() error {
	return nil
}

func (c *samlConnector) State() string {
	return "RelayState"
}

func (c *samlConnector) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
	// Construct an AuthnRequest
	authnRequest := c.spSettings.GetAuthnRequest()

	if len(c.spNameIDPolicyFormat) != 0 {
		authnRequest.NameIDPolicy.Format = c.spNameIDPolicyFormat
	}

	if len(c.spRequestedAuthnContext) == 0 {
		authnRequest.RequestedAuthnContext = nil
	}

	b64XML, err := authnRequest.CompressedEncodedString()
	if err != nil {
		return "", err
	}

	// Get a URL formed with the SAMLRequest parameter
	url, err := saml.GetAuthnRequestURL(c.spSettings.IDPSSOURL, b64XML, state)
	if err != nil {
		return "", err
	}

	if c.spSettings.AssertionConsumerServiceURL != callbackURL {
		return "", fmt.Errorf("expected callback URL did not match the URL in the config")
	}
	return url, err
}

type samlError struct {
	error            string
	errorDescription string
}

func (e *samlError) Error() string {
	if e.errorDescription == "" {
		return e.error
	}
	return e.error + ": " + e.errorDescription
}

func (c *samlConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return identity, &samlError{errType, q.Get("error_description")}
	}

	encodedXML := r.FormValue("SAMLResponse")
	if encodedXML == "" {
		return identity, fmt.Errorf("saml: SAMLResponse form value missing")
	}

	response, err := saml.ParseEncodedResponse(encodedXML)
	if err != nil {
		return identity, fmt.Errorf("saml: SAMLResponse parse: %v", err.Error())
	}

	err = response.Validate(&c.spSettings)
	if err != nil {
		return identity, fmt.Errorf("saml: SAMLResponse validation: %v", err.Error())
	}

	userID := response.GetAttribute(c.idAttr)
	if userID == "" {
		return identity, fmt.Errorf("saml: SAML attribute identifier %s missing", c.idAttr)
	}

	username := response.GetAttribute(c.userAttr)
	if username == "" {
		return identity, fmt.Errorf("saml: SAML attribute identifier %s missing", c.userAttr)
	}

	email := response.GetAttribute(c.emailAttr)
	if email == "" {
		return identity, fmt.Errorf("saml: SAML attribute identifier %s missing", c.emailAttr)
	}

	identity = connector.Identity{
		UserID:   userID,
		Username: username,
		Email:    email,
	}

	if s.Groups {
		if c.groupsDelimiter == "" {
			identity.Groups = response.GetAttributeValues(c.groupsAttr)
		} else {
			identity.Groups = strings.Split(response.GetAttribute(c.groupsAttr), c.groupsDelimiter)
		}
	}

	return identity, nil
}

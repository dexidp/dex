package connector

import (
	"html/template"
	"net/url"
	"testing"

	"github.com/coreos/go-oidc/oidc"
)

var (
	ns        url.URL
	lf        oidc.LoginFunc
	nsf       NewSessionFunc
	templates *template.Template
)

func init() {
	templates = template.New(LDAPLoginPageTemplateName)
}

func TestLDAPConnectorConfigValidTLS(t *testing.T) {
	cc := LDAPConnectorConfig{
		ID:     "ldap",
		Host:   "example.com:636",
		UseTLS: true,
		UseSSL: false,
	}

	_, err := cc.Connector(ns, lf, nsf, templates)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLDAPConnectorConfigInvalidSSLandTLS(t *testing.T) {
	cc := LDAPConnectorConfig{
		ID:     "ldap",
		Host:   "example.com:636",
		UseTLS: true,
		UseSSL: true,
	}

	_, err := cc.Connector(ns, lf, nsf, templates)
	if err == nil {
		t.Fatal("Expected LDAPConnector initialization to fail when both TLS and SSL enabled.")
	}
}

func TestLDAPConnectorConfigValidSearchScope(t *testing.T) {
	cc := LDAPConnectorConfig{
		ID:          "ldap",
		Host:        "example.com:636",
		SearchScope: "one",
	}

	_, err := cc.Connector(ns, lf, nsf, templates)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLDAPConnectorConfigInvalidSearchScope(t *testing.T) {
	cc := LDAPConnectorConfig{
		ID:          "ldap",
		Host:        "example.com:636",
		SearchScope: "three",
	}

	_, err := cc.Connector(ns, lf, nsf, templates)
	if err == nil {
		t.Fatal("Expected LDAPConnector initialization to fail when invalid value provided for SearchScope.")
	}
}

func TestLDAPConnectorConfigInvalidCertFileNoKeyFile(t *testing.T) {
	cc := LDAPConnectorConfig{
		ID:       "ldap",
		Host:     "example.com:636",
		CertFile: "/tmp/ldap.crt",
	}

	_, err := cc.Connector(ns, lf, nsf, templates)
	if err == nil {
		t.Fatal("Expected LDAPConnector initialization to fail when CertFile specified without KeyFile.")
	}
}

func TestLDAPConnectorConfigValidCertFileAndKeyFile(t *testing.T) {
	cc := LDAPConnectorConfig{
		ID:       "ldap",
		Host:     "example.com:636",
		CertFile: "/tmp/ldap.crt",
		KeyFile:  "/tmp/ldap.key",
	}

	_, err := cc.Connector(ns, lf, nsf, templates)
	if err != nil {
		t.Fatal(err)
	}
}

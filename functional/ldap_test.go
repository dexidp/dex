package functional

import (
	"fmt"
	"html/template"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/repo"
	"github.com/coreos/go-oidc/oidc"
	"gopkg.in/ldap.v2"
)

var (
	ldapHost   string
	ldapPort   uint16
	ldapBindDN string
	ldapBindPw string
)

func init() {
	ldapuri := os.Getenv("DEX_TEST_LDAP_URI")
	if ldapuri == "" {
		fmt.Println("Unable to proceed with empty env var " +
			"DEX_TEST_LDAP_URI")
		os.Exit(1)
	}
	u, err := url.Parse(ldapuri)
	if err != nil {
		fmt.Println("Unable to parse DEX_TEST_LDAP_URI")
		os.Exit(1)
	}
	if strings.Index(u.RawQuery, "?") < 0 {
		fmt.Println("Unable to parse DEX_TEST_LDAP_URI")
		os.Exit(1)
	}
	extentions := make(map[string]string)
	kvs := strings.Split(strings.TrimLeft(u.RawQuery, "?"), ",")
	for i := range kvs {
		fmt.Println(kvs[i])
		kv := strings.Split(kvs[i], "=")
		if len(kv) < 2 {
			fmt.Println("Unable to parse DEX_TEST_LDAP_URI")
			os.Exit(1)
		}
		extentions[kv[0]] = kv[1]
	}
	hostport := strings.Split(u.Host, ":")
	port := 389
	if len(hostport) > 1 {
		port, _ = strconv.Atoi(hostport[1])
	}

	ldapHost = hostport[0]
	ldapPort = uint16(port)

	if len(extentions["bindname"]) > 0 {
		ldapBindDN, err = url.QueryUnescape(extentions["bindname"])
		if err != nil {
			fmt.Println("Unable to parse DEX_TEST_LDAP_URI")
			os.Exit(1)
		}
	}
	if len(extentions["X-BINDPW"]) > 0 {
		ldapBindPw = extentions["X-BINDPW"]
	}
}

func TestLDAPConnect(t *testing.T) {
	fmt.Println("ldapHost:   ", ldapHost)
	fmt.Println("ldapPort:   ", ldapPort)
	fmt.Println("ldapBindDN: ", ldapBindDN)
	fmt.Println("ldapBindPw: ", ldapBindPw)
	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", ldapHost, ldapPort))
	if err != nil {
		t.Fatal(err)
	}
	err = l.Bind(ldapBindDN, ldapBindPw)
	if err != nil {
		t.Fatal(err)
	}
	l.Close()
}

func TestConnectorLDAPConnectFail(t *testing.T) {
	var tx repo.Transaction
	var lf oidc.LoginFunc
	var ns url.URL

	templates := template.New(connector.LDAPLoginPageTemplateName)

	ccr := connector.NewConnectorConfigRepoFromConfigs(
		[]connector.ConnectorConfig{&connector.LDAPConnectorConfig{
			ID:         "ldap",
			ServerHost: ldapHost,
			ServerPort: ldapPort + 1,
		}},
	)
	cc, err := ccr.GetConnectorByID(tx, "ldap")
	if err != nil {
		t.Fatal(err)
	}
	c, err := cc.Connector(ns, lf, templates)
	if err != nil {
		t.Fatal(err)
	}
	err = c.Healthy()
	if err == nil {
		t.Fatal(fmt.Errorf("LDAPConnector.Healty() supposed to fail, but succeeded!"))
	}
}

func TestConnectorLDAPConnectSuccess(t *testing.T) {
	var tx repo.Transaction
	var lf oidc.LoginFunc
	var ns url.URL

	templates := template.New(connector.LDAPLoginPageTemplateName)

	ccr := connector.NewConnectorConfigRepoFromConfigs(
		[]connector.ConnectorConfig{&connector.LDAPConnectorConfig{
			ID:         "ldap",
			ServerHost: ldapHost,
			ServerPort: ldapPort,
		}},
	)
	cc, err := ccr.GetConnectorByID(tx, "ldap")
	if err != nil {
		t.Fatal(err)
	}
	c, err := cc.Connector(ns, lf, templates)
	if err != nil {
		t.Fatal(err)
	}
	err = c.Healthy()
	if err != nil {
		t.Fatal(err)
	}
}

func TestConnectorLDAPcaFilecertFileConnectTLS(t *testing.T) {
	var tx repo.Transaction
	var lf oidc.LoginFunc
	var ns url.URL

	templates := template.New(connector.LDAPLoginPageTemplateName)

	ccr := connector.NewConnectorConfigRepoFromConfigs(
		[]connector.ConnectorConfig{&connector.LDAPConnectorConfig{
			ID:         "ldap",
			ServerHost: ldapHost,
			ServerPort: ldapPort,
			UseTLS:     true,
			CertFile:   "/tmp/ldap.crt",
			KeyFile:    "/tmp/ldap.key",
			CaFile:     "/tmp/openldap-ca.pem",
		}},
	)
	cc, err := ccr.GetConnectorByID(tx, "ldap")
	if err != nil {
		t.Fatal(err)
	}
	c, err := cc.Connector(ns, lf, templates)
	if err != nil {
		t.Fatal(err)
	}
	err = c.Healthy()
	if err != nil {
		t.Fatal(err)
	}
}

func TestConnectorLDAPcaFilecertFileConnectSSL(t *testing.T) {
	var tx repo.Transaction
	var lf oidc.LoginFunc
	var ns url.URL

	templates := template.New(connector.LDAPLoginPageTemplateName)

	ccr := connector.NewConnectorConfigRepoFromConfigs(
		[]connector.ConnectorConfig{&connector.LDAPConnectorConfig{
			ID:         "ldap",
			ServerHost: ldapHost,
			ServerPort: ldapPort + 247, // 636
			UseSSL:     true,
			CertFile:   "/tmp/ldap.crt",
			KeyFile:    "/tmp/ldap.key",
			CaFile:     "/tmp/openldap-ca.pem",
		}},
	)
	cc, err := ccr.GetConnectorByID(tx, "ldap")
	if err != nil {
		t.Fatal(err)
	}
	c, err := cc.Connector(ns, lf, templates)
	if err != nil {
		t.Fatal(err)
	}
	err = c.Healthy()
	if err != nil {
		t.Fatal(err)
	}
}

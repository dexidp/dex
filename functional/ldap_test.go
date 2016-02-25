package functional

import (
	"fmt"
	"html/template"
	"net"
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/coreos/dex/connector"
	"gopkg.in/ldap.v2"
)

var (
	ldapHost   string
	ldapPort   uint16
	ldapBindDN string
	ldapBindPw string
)

type LDAPServer struct {
	Host   string
	Port   uint16
	BindDN string
	BindPw string
}

const (
	ldapEnvHost     = "DEX_TEST_LDAP_HOST"
	ldapEnvBindName = "DEX_TEST_LDAP_BINDNAME"
	ldapEnvBindPass = "DEX_TEST_LDAP_BINDPASS"
)

func ldapServer(t *testing.T) LDAPServer {
	host := os.Getenv(ldapEnvHost)
	if host == "" {
		t.Fatalf("%s not set", ldapEnvHost)
	}
	var port uint64 = 389
	if h, p, err := net.SplitHostPort(host); err == nil {
		port, err = strconv.ParseUint(p, 10, 16)
		if err != nil {
			t.Fatalf("failed to parse port: %v", err)
		}
		host = h
	}
	return LDAPServer{host, uint16(port), os.Getenv(ldapEnvBindName), os.Getenv(ldapEnvBindPass)}
}

func TestLDAPConnect(t *testing.T) {
	server := ldapServer(t)
	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", server.Host, server.Port))
	if err != nil {
		t.Fatal(err)
	}
	err = l.Bind(server.BindDN, server.BindPw)
	if err != nil {
		t.Fatal(err)
	}
	l.Close()
}

func TestConnectorLDAPHealthy(t *testing.T) {
	server := ldapServer(t)

	tests := []struct {
		config  connector.LDAPConnectorConfig
		wantErr bool
	}{
		{
			config: connector.LDAPConnectorConfig{
				ID:         "ldap",
				ServerHost: server.Host,
				ServerPort: server.Port + 1,
			},
			wantErr: true,
		},
		{
			config: connector.LDAPConnectorConfig{
				ID:         "ldap",
				ServerHost: server.Host,
				ServerPort: server.Port,
			},
		},
		{
			config: connector.LDAPConnectorConfig{
				ID:         "ldap",
				ServerHost: server.Host,
				ServerPort: server.Port,
				UseTLS:     true,
				CertFile:   "/tmp/ldap.crt",
				KeyFile:    "/tmp/ldap.key",
				CaFile:     "/tmp/openldap-ca.pem",
			},
		},
		{
			config: connector.LDAPConnectorConfig{
				ID:         "ldap",
				ServerHost: server.Host,
				ServerPort: server.Port + 247, // 636
				UseSSL:     true,
				CertFile:   "/tmp/ldap.crt",
				KeyFile:    "/tmp/ldap.key",
				CaFile:     "/tmp/openldap-ca.pem",
			},
		},
	}
	for i, tt := range tests {
		templates := template.New(connector.LDAPLoginPageTemplateName)
		c, err := tt.config.Connector(url.URL{}, nil, templates)
		if err != nil {
			t.Errorf("case %d: failed to create connector: %v", i, err)
			continue
		}
		if err := c.Healthy(); err != nil {
			if !tt.wantErr {
				t.Errorf("case %d: Healthy() returned error: %v", i, err)
			}
		} else if tt.wantErr {
			t.Errorf("case %d: expected Healthy() to fail", i)
		}
	}
}

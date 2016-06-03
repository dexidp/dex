package functional

import (
	"fmt"
	"html/template"
	"net"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/coreos/dex/connector"
	"golang.org/x/net/context"
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

func TestLDAPPoolHighWatermarkAndLockContention(t *testing.T) {
	server := ldapServer(t)
	ldapPool := &connector.LDAPPool{
		MaxIdleConn: 30,
		ServerHost:  server.Host,
		ServerPort:  server.Port,
		UseTLS:      false,
		UseSSL:      false,
	}

	// Excercise pool operations with MaxIdleConn + 10 concurrent goroutines.
	// We are testing both pool high watermark code and lock contention
	numRoutines := ldapPool.MaxIdleConn + 10
	var wg sync.WaitGroup
	wg.Add(numRoutines)
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	for i := 0; i < numRoutines; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					ldapConn, err := ldapPool.Acquire()
					if err != nil {
						t.Errorf("Unable to acquire LDAP Connection: %v", err)
					}
					s := ldap.NewSearchRequest("", ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false, "(objectClass=*)", []string{}, nil)
					_, err = ldapConn.Search(s)
					if err != nil {
						t.Errorf("Search request failed. Dead/invalid LDAP connection from pool?: %v", err)
						ldapConn.Close()
					} else {
						ldapPool.Put(ldapConn)
					}
					_, _ = ldapPool.CheckConnections()
				}
			}
		}()
	}

	// Wait for all operations to complete and check status.
	// There should be MaxIdleConn connections in the pool. This confirms:
	// 1. The tests was indeed executed concurrently
	// 2. Even though we ran more routines than the configured MaxIdleConn the high
	//    watermark code did its job and closed surplus connections
	wg.Wait()
	alive, killed := ldapPool.CheckConnections()
	if alive < ldapPool.MaxIdleConn {
		t.Errorf("expected %v connections, got alive=%v killed=%v", ldapPool.MaxIdleConn, alive, killed)
	}
}

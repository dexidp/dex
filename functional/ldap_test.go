package functional

import (
	"html/template"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/coreos/dex/connector"
	"golang.org/x/net/context"
	"gopkg.in/ldap.v2"
)

type LDAPServer struct {
	Host      string // Address (host:port) of LDAP service.
	LDAPSHost string // Address (host:port) of LDAPS service (TLS port).
	BindDN    string
	BindPw    string
}

const (
	ldapEnvHost  = "DEX_TEST_LDAP_HOST"
	ldapsEnvHost = "DEX_TEST_LDAPS_HOST"

	ldapEnvBindName = "DEX_TEST_LDAP_BINDNAME"
	ldapEnvBindPass = "DEX_TEST_LDAP_BINDPASS"
)

func ldapServer(t *testing.T) LDAPServer {
	getenv := func(key string) string {
		val := os.Getenv(key)
		if val == "" {
			t.Fatalf("%s not set", key)
		}
		t.Logf("%s=%v", key, val)
		return val
	}
	return LDAPServer{
		Host:      getenv(ldapEnvHost),
		LDAPSHost: getenv(ldapsEnvHost),
		BindDN:    os.Getenv(ldapEnvBindName),
		BindPw:    os.Getenv(ldapEnvBindPass),
	}
}

func TestLDAPConnect(t *testing.T) {
	server := ldapServer(t)
	l, err := ldap.Dial("tcp", server.Host)
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
				ID:   "ldap",
				Host: "localhost:0",
			},
			wantErr: true,
		},
		{
			config: connector.LDAPConnectorConfig{
				ID:   "ldap",
				Host: server.Host,
			},
		},
		{
			config: connector.LDAPConnectorConfig{
				ID:       "ldap",
				Host:     server.Host,
				UseTLS:   true,
				CertFile: "/tmp/ldap.crt",
				KeyFile:  "/tmp/ldap.key",
				CaFile:   "/tmp/openldap-ca.pem",
			},
		},
		{
			config: connector.LDAPConnectorConfig{
				ID:       "ldap",
				Host:     server.LDAPSHost,
				UseSSL:   true,
				CertFile: "/tmp/ldap.crt",
				KeyFile:  "/tmp/ldap.key",
				CaFile:   "/tmp/openldap-ca.pem",
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
		Host:        server.Host,
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
					err := ldapPool.Do(func(conn *ldap.Conn) error {
						s := &ldap.SearchRequest{
							Scope:  ldap.ScopeBaseObject,
							Filter: "(objectClass=*)",
						}
						_, err := conn.Search(s)
						return err
					})
					if err != nil {
						t.Errorf("Search request failed. Dead/invalid LDAP connection from pool?: %v", err)
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

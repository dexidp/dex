// Package ldap implements strategies for authenticating using the LDAP protocol.
package ldap

import (
	"errors"
	"fmt"

	"gopkg.in/ldap.v2"

	"github.com/coreos/dex/connector"
)

// Config holds the configuration parameters for the LDAP connector.
type Config struct {
	Host   string `yaml:"host"`
	BindDN string `yaml:"bindDN"`
}

// Open returns an authentication strategy using LDAP.
func (c *Config) Open() (connector.Connector, error) {
	if c.Host == "" {
		return nil, errors.New("missing host parameter")
	}
	if c.BindDN == "" {
		return nil, errors.New("missing bindDN paramater")
	}
	return &ldapConnector{*c}, nil
}

type ldapConnector struct {
	Config
}

var _ connector.PasswordConnector = (*ldapConnector)(nil)

func (c *ldapConnector) do(f func(c *ldap.Conn) error) error {
	// TODO(ericchiang): Connection pooling.
	conn, err := ldap.Dial("tcp", c.Host)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer conn.Close()

	return f(conn)
}

func (c *ldapConnector) Login(username, password string) (connector.Identity, bool, error) {
	err := c.do(func(conn *ldap.Conn) error {
		return conn.Bind(fmt.Sprintf("uid=%s,%s", username, c.BindDN), password)
	})
	if err != nil {
		// TODO(ericchiang): Determine when the user has entered invalid credentials.
		return connector.Identity{}, false, err
	}

	return connector.Identity{Username: username}, true, nil
}

func (c *ldapConnector) Close() error {
	return nil
}

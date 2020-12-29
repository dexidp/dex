// Package ldapcluster implements strategies for authenticating with a cluster of LDAP servers using the LDAP protocol.
package ldapcluster

import (
	"context"

	"github.com/dexidp/dex/connector"
	conn_ldap "github.com/dexidp/dex/connector/ldap"
	"github.com/dexidp/dex/pkg/log"
)

// Config holds the configuration parameters for the LDAP cluster connector. The LDAP
// connectors require executing two queries, the first to find the user based on
// the username and password given to the connector. The second to use the user
// entry to search for groups.
// The cluster connector takes multiple LDAP connectors.
//
// An example config:
//connectors:
//- type: ldapcluster
//  name: OpenLDAP
//  id: ldapcluster
//  config:
//    clustermembers:
//      - host: localhost:399
//
//        # No TLS for this setup.
//        insecureNoSSL: true
//
//        # This would normally be a read-only user.
//        bindDN: cn=admin,dc=example,dc=org
//        bindPW: admin
//
//        usernamePrompt: Email Address
//
//        userSearch:
//          baseDN: ou=People,dc=example,dc=org
//          filter: "(objectClass=person)"
//          username: mail
//          # "DN" (case sensitive) is a special attribute name. It indicates that
//          # this value should be taken from the entity's DN not an attribute on
//          # the entity.
//          idAttr: DN
//          emailAttr: mail
//          nameAttr: cn
//
//        groupSearch:
//          baseDN: ou=Groups,dc=example,dc=org
//          filter: "(objectClass=groupOfNames)"
//
//          userMatchers:
//            # A user is a member of a group when their DN matches
//            # the value of a "member" attribute on the group entity.
//          - userAttr: DN
//            groupAttr: member
//
//          # The group name should be the "cn" value.
//          nameAttr: cn
//
//      - host: localhost:389
//
//        # No TLS for this setup.
//        insecureNoSSL: true
//
//        # This would normally be a read-only user.
//        bindDN: cn=admin,dc=example,dc=org
//        bindPW: admin
//
//        usernamePrompt: Email Address
//
//        userSearch:
//          baseDN: ou=People,dc=example,dc=org
//          filter: "(objectClass=person)"
//          username: mail
//          # "DN" (case sensitive) is a special attribute name. It indicates that
//          # this value should be taken from the entity's DN not an attribute on
//          # the entity.
//          idAttr: DN
//          emailAttr: mail
//          nameAttr: cn
//
//        groupSearch:
//          baseDN: ou=Groups,dc=example,dc=org
//          filter: "(objectClass=groupOfNames)"
//
//          userMatchers:
//            # A user is a member of a group when their DN matches
//            # the value of a "member" attribute on the group entity.
//          - userAttr: DN
//            groupAttr: member
//
//          # The group name should be the "cn" value.
//          nameAttr: cn
//

type Config struct {
	ClusterMembers []conn_ldap.Config
}

// Open returns an authentication strategy using LDAP.
func (c *Config) Open(id string, logger log.Logger) (connector.Connector, error) {
	conn, err := c.OpenConnector(logger)
	if err != nil {
		return nil, err
	}
	return connector.Connector(conn), nil
}

// OpenConnector is the same as Open but returns a type with all implemented connector interfaces.
func (c *Config) OpenConnector(logger log.Logger) (interface {
	connector.Connector
	connector.PasswordConnector
	connector.RefreshConnector
}, error) {
	return c.openConnector(logger)
}

func (c *Config) openConnector(logger log.Logger) (*ldapClusterConnector, error) {
	var lcc ldapClusterConnector
	// Initialize each of the connector members.
	for _, v := range c.ClusterMembers {
		lc, e := v.OpenConnector(logger)
		if e != nil {
			return nil, e
		}
		lcc.MemberConnectors = append(lcc.MemberConnectors, lc)
	}

	lcc.activeMemberIdx = 0
	lcc.logger = logger

	return &lcc, nil
}

type ConnectorIf interface {
	connector.Connector
	connector.PasswordConnector
	connector.RefreshConnector
}

type ldapClusterConnector struct {
	MemberConnectors [](ConnectorIf)
	activeMemberIdx  int
	logger           log.Logger
}

func (c *ldapClusterConnector) Login(ctx context.Context, s connector.Scopes, username, password string) (ident connector.Identity, validPass bool, err error) {
	// make this check to avoid unauthenticated bind to the LDAP server.
	if password == "" {
		return connector.Identity{}, false, nil
	}

	// Check the active connector first.
	// If the active connector index is -1, we will start
	// with first connector.
	if c.activeMemberIdx == -1 {
		c.activeMemberIdx = 0
	}
	lc := c.MemberConnectors[c.activeMemberIdx]
	i, b, e := lc.Login(ctx, s, username, password)
	if e != nil {
		c.logger.Infof("Failed to connect to server idx: %d", c.activeMemberIdx)
		// Current active server has returned error.
		// Try the other servers in round robin manner.
		// If the error returned by a server is nil,
		// then make that server as
		// the current active server.
		for k, v := range c.MemberConnectors {
			if k == c.activeMemberIdx {
				// we just tried it.
				// hence skip.
				continue
			}
			i, b, e = v.Login(ctx, s, username, password)
			if e == nil {
				c.logger.Infof("setting active index as: %d", k)
				c.activeMemberIdx = k
				return i, b, e
			}
		}
	}
	return i, b, e
}

func (c *ldapClusterConnector) Refresh(ctx context.Context, s connector.Scopes, ident connector.Identity) (connector.Identity, error) {
	lc := c.MemberConnectors[c.activeMemberIdx]
	i, e := lc.Refresh(ctx, s, ident)
	if e != nil {
		c.logger.Infof("Failed to connect to active index: %d", c.activeMemberIdx)
		// current active server has returned error.
		// Try the other servers in round robin manner.
		// If the error returned by a server is nil,
		// then make that server as
		// the current active server.
		for k, v := range c.MemberConnectors {
			if k == c.activeMemberIdx {
				// we just tried it.
				// hence skip.
				continue
			}
			c.logger.Infof("Trying index: %d", k)
			i, e = v.Refresh(ctx, s, ident)
			if e == nil {
				c.logger.Infof("setting active index as: %d", k)
				c.activeMemberIdx = k
				return i, nil
			}
			c.logger.Errorf("Failed to connect to index: %d", k)
		}
	}

	return i, e
}

func (c *ldapClusterConnector) Prompt() string {
	lc := c.MemberConnectors[c.activeMemberIdx]
	return lc.Prompt()
}

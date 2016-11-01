// Package ldap implements strategies for authenticating using the LDAP protocol.
package ldap

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"unicode"

	"gopkg.in/ldap.v2"

	"github.com/coreos/dex/connector"
)

// Config holds the configuration parameters for the LDAP connector. The LDAP
// connectors require executing two queries, the first to find the user based on
// the username and password given to the connector. The second to use the user
// entry to search for groups.
//
// An example config:
//
//     type: ldap
//     config:
//       host: ldap.example.com:636
//       # The following field is required if using port 389.
//       # insecureNoSSL: true
//       rootCA: /etc/dex/ldap.ca
//       bindDN: uid=seviceaccount,cn=users,dc=example,dc=com
//       bindPW: password
//       userSearch:
//         # Would translate to the query "(&(objectClass=person)(uid=<username>))"
//         baseDN: cn=users,dc=example,dc=com
//         filter: "(objectClass=person)"
//         username: uid
//         idAttr: uid
//         emailAttr: mail
//         nameAttr: name
//       groupSearch:
//         # Would translate to the query "(&(objectClass=group)(member=<user uid>))"
//         baseDN: cn=groups,dc=example,dc=com
//         filter: "(objectClass=group)"
//         userAttr: uid
//         groupAttr: member
//         nameAttr: name
//
type Config struct {
	// The host and optional port of the LDAP server. If port isn't supplied, it will be
	// guessed based on the TLS configuration. 389 or 636.
	Host string `yaml:"host"`

	// Required if LDAP host does not use TLS.
	InsecureNoSSL bool `yaml:"insecureNoSSL"`

	// Path to a trusted root certificate file.
	RootCA string `yaml:"rootCA"`

	// BindDN and BindPW for an application service account. The connector uses these
	// credentials to search for users and groups.
	BindDN string `yaml:"bindDN"`
	BindPW string `yaml:"bindPW"`

	// User entry search configuration.
	UserSearch struct {
		// BsaeDN to start the search from. For example "cn=users,dc=example,dc=com"
		BaseDN string `yaml:"baseDN"`

		// Optional filter to apply when searching the directory. For example "(objectClass=person)"
		Filter string `yaml:"filter"`

		// Attribute to match against the inputted username. This will be translated and combined
		// with the other filter as "(<attr>=<username>)".
		Username string `yaml:"username"`

		// Can either be:
		// * "sub" - search the whole sub tree
		// * "one" - only search one level
		Scope string `yaml:"scope"`

		// A mapping of attributes on the user entry to claims.
		IDAttr    string `yaml:"idAttr"`    // Defaults to "uid"
		EmailAttr string `yaml:"emailAttr"` // Defaults to "mail"
		NameAttr  string `yaml:"nameAttr"`  // No default.

	} `yaml:"userSearch"`

	// Group search configuration.
	GroupSearch struct {
		// BsaeDN to start the search from. For example "cn=groups,dc=example,dc=com"
		BaseDN string `yaml:"baseDN"`

		// Optional filter to apply when searching the directory. For example "(objectClass=posixGroup)"
		Filter string `yaml:"filter"`

		Scope string `yaml:"scope"` // Defaults to "sub"

		// These two fields are use to match a user to a group.
		//
		// It adds an additional requirement to the filter that an attribute in the group
		// match the user's attribute value. For example that the "members" attribute of
		// a group matches the "uid" of the user. The exact filter being added is:
		//
		//   (<groupAttr>=<userAttr value>)
		//
		UserAttr  string `yaml:"userAttr"`
		GroupAttr string `yaml:"groupAttr"`

		// The attribute of the group that represents its name.
		NameAttr string `yaml:"nameAttr"`
	} `yaml:"groupSearch"`
}

func parseScope(s string) (int, bool) {
	// NOTE(ericchiang): ScopeBaseObject doesn't really make sense for us because we
	// never know the user's or group's DN.
	switch s {
	case "", "sub":
		return ldap.ScopeWholeSubtree, true
	case "one":
		return ldap.ScopeSingleLevel, true
	}
	return 0, false
}

// escapeRune maps a rune to a hex encoded value. For example 'é' would become '\\c3\\a9'
func escapeRune(buff *bytes.Buffer, r rune) {
	// Really inefficient, but it seems correct.
	for _, b := range []byte(string(r)) {
		buff.WriteString("\\")
		buff.WriteString(hex.EncodeToString([]byte{b}))
	}
}

// NOTE(ericchiang): There are no good documents on how to escape an LDAP string.
// This implementation is inspired by an Oracle document, and is purposefully
// extremely restrictive.
//
// See: https://docs.oracle.com/cd/E19424-01/820-4811/gdxpo/index.html
func escapeFilter(s string) string {
	r := strings.NewReader(s)
	buff := new(bytes.Buffer)
	for {
		ru, _, err := r.ReadRune()
		if err != nil {
			// ignore decoding issues
			return buff.String()
		}

		switch {
		case ru > unicode.MaxASCII: // Not ASCII
			escapeRune(buff, ru)
		case !unicode.IsPrint(ru): // Not printable
			escapeRune(buff, ru)
		case strings.ContainsRune(`*\()`, ru): // Reserved characters
			escapeRune(buff, ru)
		default:
			buff.WriteRune(ru)
		}
	}
}

// Open returns an authentication strategy using LDAP.
func (c *Config) Open() (connector.Connector, error) {
	requiredFields := []struct {
		name string
		val  string
	}{
		{"host", c.Host},
		{"userSearch.baseDN", c.UserSearch.BaseDN},
		{"userSearch.username", c.UserSearch.Username},
	}

	for _, field := range requiredFields {
		if field.val == "" {
			return nil, fmt.Errorf("ldap: missing required field %q", field.name)
		}
	}

	var (
		host string
		err  error
	)
	if host, _, err = net.SplitHostPort(c.Host); err != nil {
		host = c.Host
		if c.InsecureNoSSL {
			c.Host = c.Host + ":389"
		} else {
			c.Host = c.Host + ":636"
		}
	}

	tlsConfig := new(tls.Config)
	if c.RootCA != "" {
		data, err := ioutil.ReadFile(c.RootCA)
		if err != nil {
			return nil, fmt.Errorf("ldap: read ca file: %v", err)
		}
		rootCAs := x509.NewCertPool()
		if !rootCAs.AppendCertsFromPEM(data) {
			return nil, fmt.Errorf("ldap: no certs found in ca file")
		}
		tlsConfig.RootCAs = rootCAs
		// NOTE(ericchiang): This was required for our internal LDAP server
		// but might be because of an issue with our root CA.
		tlsConfig.ServerName = host
	}
	userSearchScope, ok := parseScope(c.UserSearch.Scope)
	if !ok {
		return nil, fmt.Errorf("userSearch.Scope unknown value %q", c.UserSearch.Scope)
	}
	groupSearchScope, ok := parseScope(c.GroupSearch.Scope)
	if !ok {
		return nil, fmt.Errorf("userSearch.Scope unknown value %q", c.GroupSearch.Scope)
	}
	return &ldapConnector{*c, userSearchScope, groupSearchScope, tlsConfig}, nil
}

type ldapConnector struct {
	Config

	userSearchScope  int
	groupSearchScope int

	tlsConfig *tls.Config
}

var _ connector.PasswordConnector = (*ldapConnector)(nil)

// do initializes a connection to the LDAP directory and passes it to the
// provided function. It then performs appropriate teardown or reuse before
// returning.
func (c *ldapConnector) do(f func(c *ldap.Conn) error) error {
	var (
		conn *ldap.Conn
		err  error
	)
	if c.InsecureNoSSL {
		conn, err = ldap.Dial("tcp", c.Host)
	} else {
		conn, err = ldap.DialTLS("tcp", c.Host, c.tlsConfig)
	}
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer conn.Close()

	// If bindDN and bindPW are empty this will default to an anonymous bind.
	if err := conn.Bind(c.BindDN, c.BindPW); err != nil {
		return fmt.Errorf("ldap: initial bind for user %q failed: %v", c.BindDN, err)
	}

	return f(conn)
}

func getAttr(e ldap.Entry, name string) string {
	for _, a := range e.Attributes {
		if a.Name != name {
			continue
		}
		if len(a.Values) == 0 {
			return ""
		}
		return a.Values[0]
	}
	return ""
}

func (c *ldapConnector) Login(username, password string) (ident connector.Identity, validPass bool, err error) {
	var (
		// We want to return a different error if the user's password is incorrect vs
		// if there was an error.
		incorrectPass = false
		user          ldap.Entry
	)

	filter := fmt.Sprintf("(%s=%s)", c.UserSearch.Username, escapeFilter(username))
	if c.UserSearch.Filter != "" {
		filter = fmt.Sprintf("(&%s%s)", c.UserSearch.Filter, filter)
	}

	// Initial search.
	req := &ldap.SearchRequest{
		BaseDN: c.UserSearch.BaseDN,
		Filter: filter,
		Scope:  c.userSearchScope,
		// We only need to search for these specific requests.
		Attributes: []string{
			c.UserSearch.IDAttr,
			c.UserSearch.EmailAttr,
			c.GroupSearch.UserAttr,
			// TODO(ericchiang): what if this contains duplicate values?
		},
	}

	if c.UserSearch.NameAttr != "" {
		req.Attributes = append(req.Attributes, c.UserSearch.NameAttr)
	}

	err = c.do(func(conn *ldap.Conn) error {
		resp, err := conn.Search(req)
		if err != nil {
			return fmt.Errorf("ldap: search with filter %q failed: %v", req.Filter, err)
		}

		switch n := len(resp.Entries); n {
		case 0:
			log.Printf("ldap: no results returned for filter: %q", filter)
			incorrectPass = true
			return nil
		case 1:
		default:
			return fmt.Errorf("ldap: filter returned multiple (%d) results: %q", n, filter)
		}

		user = *resp.Entries[0]

		// Try to authenticate as the distinguished name.
		if err := conn.Bind(user.DN, password); err != nil {
			// Detect a bad password through the LDAP error code.
			if ldapErr, ok := err.(*ldap.Error); ok {
				if ldapErr.ResultCode == ldap.LDAPResultInvalidCredentials {
					log.Printf("ldap: invalid password for user %q", user.DN)
					incorrectPass = true
					return nil
				}
			}
			return fmt.Errorf("ldap: failed to bind as dn %q: %v", user.DN, err)
		}
		return nil
	})
	if err != nil {
		return connector.Identity{}, false, err
	}
	if incorrectPass {
		return connector.Identity{}, false, nil
	}

	// Encode entry for follow up requests such as the groups query and
	// refresh attempts.
	if ident.ConnectorData, err = json.Marshal(user); err != nil {
		return connector.Identity{}, false, fmt.Errorf("ldap: marshal entry: %v", err)
	}

	// If we're missing any attributes, such as email or ID, we want to report
	// an error rather than continuing.
	missing := []string{}

	// Fill the identity struct using the attributes from the user entry.
	if ident.UserID = getAttr(user, c.UserSearch.IDAttr); ident.UserID == "" {
		missing = append(missing, c.UserSearch.IDAttr)
	}
	if ident.Email = getAttr(user, c.UserSearch.EmailAttr); ident.Email == "" {
		missing = append(missing, c.UserSearch.EmailAttr)
	}
	if c.UserSearch.NameAttr != "" {
		if ident.Username = getAttr(user, c.UserSearch.NameAttr); ident.Username == "" {
			missing = append(missing, c.UserSearch.NameAttr)
		}
	}

	if len(missing) != 0 {
		err := fmt.Errorf("ldap: entry %q missing following required attribute(s): %q", user.DN, missing)
		return connector.Identity{}, false, err
	}

	return ident, true, nil
}

func (c *ldapConnector) Groups(ident connector.Identity) ([]string, error) {
	// Decode the user entry from the identity.
	var user ldap.Entry
	if err := json.Unmarshal(ident.ConnectorData, &user); err != nil {
		return nil, fmt.Errorf("ldap: failed to unmarshal connector data: %v", err)
	}

	filter := fmt.Sprintf("(%s=%s)", c.GroupSearch.GroupAttr, escapeFilter(getAttr(user, c.GroupSearch.UserAttr)))
	if c.GroupSearch.Filter != "" {
		filter = fmt.Sprintf("(&%s%s)", c.GroupSearch.Filter, filter)
	}

	req := &ldap.SearchRequest{
		BaseDN:     c.GroupSearch.BaseDN,
		Filter:     filter,
		Scope:      c.groupSearchScope,
		Attributes: []string{c.GroupSearch.NameAttr},
	}

	var groups []*ldap.Entry
	if err := c.do(func(conn *ldap.Conn) error {
		resp, err := conn.Search(req)
		if err != nil {
			return fmt.Errorf("ldap: search failed: %v", err)
		}
		groups = resp.Entries
		return nil
	}); err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		// TODO(ericchiang): Is this going to spam the logs?
		log.Printf("ldap: groups search with filter %q returned no groups", filter)
	}

	var groupNames []string

	for _, group := range groups {
		name := getAttr(*group, c.GroupSearch.NameAttr)
		if name == "" {
			// Be obnoxious about missing missing attributes. If the group entry is
			// missing its name attribute, that indicates a misconfiguration.
			//
			// In the future we can add configuration options to just log these errors.
			return nil, fmt.Errorf("ldap: group entity %q missing required attribute %q",
				group.DN, c.GroupSearch.NameAttr)
		}

		groupNames = append(groupNames, name)
	}
	return groupNames, nil
}

func (c *ldapConnector) Close() error {
	return nil
}

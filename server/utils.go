package server

// import (
// 	"crypto/tls"
// 	"crypto/x509"
// 	"errors"
// 	"fmt"
// 	ldap "gopkg.in/ldap.v2"
// )

// // OpenTLS connection to LDAP server using tls hand-shake
// func LDAPOpenTLS(rootPEM, network, hostAddress string) (*ldap.Conn, error) {
// 	roots := x509.NewCertPool()
// 	ok := roots.AppendCertsFromPEM([]byte(rootPEM))
// 	if !ok {
// 		return nil, errors.New("Failed to append LDAP root ca")
// 	}
// 	tlsConfig := &tls.Config{RootCAs: roots}
// 	return ldap.DialTLS(network, hostAddress, tlsConfig)
// }

// // Open connection to LDAP server
// func LDAPOpen(network, hostAddress string) (*ldap.Conn, error) {
// 	return ldap.Dial(network, hostAddress)
// }

// // SearchUserAttributesForClass if object class is not needed use '*' to escape
// func SearchUserAttributesForClass(l *ldap.Conn, baseDN, objectClass string, ldapUserAttrList []string) (*ldap.SearchResult, error) {
// 	fmt.Printf("SearchUserAttributesForClass(): baseDN= %s\n", baseDN)
// 	searchRequest := ldap.NewSearchRequest(
// 		baseDN, // The base dn to search
// 		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
// 		fmt.Sprintf("(&(objectClass=%s))", objectClass), // The filter to apply
// 		ldapUserAttrList,                                // A list attributes to retrieve
// 		nil,
// 	)
// 	return l.Search(searchRequest)
// }
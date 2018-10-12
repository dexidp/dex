package userinfo

import (
	ldap "gopkg.in/ldap.v2"
)

// Userinfo enables all required LDAP operations for a given user.
type Userinfo interface {

	// Close the LDAP connection
	Close()

	// Authenticate a (techuser) against the password stored in the DRD
	Authenticate() error

	// Retrieves user attributes
	GetUserInformation(userSearchID, userID string) (*ldap.SearchResult, error)
}

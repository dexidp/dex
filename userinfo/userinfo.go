package userinfo

import (
	ldap "gopkg.in/ldap.v2"
)

type Userinfo interface{
	Close()
	
	// Authenticate a (techuser) against the password stored in the DRD
	Authenticate() error

	// Retrieves user attributes
	GetUserInformation(userSearchID, userID string) (*ldap.SearchResult, error)
}

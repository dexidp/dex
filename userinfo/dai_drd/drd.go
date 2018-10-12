package dai_drd

import (	
	"fmt"
	"errors"
	"crypto/tls"
	"crypto/x509"
	"github.com/dexidp/dex/userinfo"
	ldap "gopkg.in/ldap.v2"
	"github.com/sirupsen/logrus"

)

type conn struct {
	ldap	*ldap.Conn
	logger	logrus.FieldLogger
	config	LDAPConfig
	userSearchIndex	map[string]UserSearch
}

func (c *LDAPConfig) Open(logger logrus.FieldLogger) (userinfo.Userinfo, error) {
	logger.Infof("opening DAI DRD userinfo adapter")
	ldap, err := c.open(logger)
	if err != nil {
		return nil, err
	}
	logger.Infof("opened DAI DRD userinfo adapter")
	return ldap, err
}

func (c* LDAPConfig) open(logger logrus.FieldLogger) (*conn, error){
	var lc *ldap.Conn
	var err error
		
	// check to see if we need to create a secure ldap connection, or just straight
	if !c.InsecureNoSSL {
		roots := x509.NewCertPool()
		ok := roots.AppendCertsFromPEM([]byte(c.RootPEM))
		if !ok {
			return nil, errors.New("Failed to append LDAP root ca")
		}
		tlsConfig := &tls.Config{RootCAs: roots}
		lc, err = ldap.DialTLS(c.Network, c.HostAddress, tlsConfig)
	} else {
		lc, err = ldap.Dial(c.Network, c.HostAddress)
	}
	
	if err != nil {
		logger.Errorf("cannot open LDAP connection")
		return nil, err
	}

	// make an indexed map out of the defined user search queries, so we do not have to go this over and over again on any user search query
	userSearchMap := make(map[string]UserSearch)
	for _,entry := range c.UserSearch {
		userSearchMap[entry.Type] = entry
	}

	return &conn{ldap: lc, logger: logger, config: *c,userSearchIndex: userSearchMap}, err
}

type LDAPConfig struct {
	HostAddress 	string			`json:"host"`
	Network 		string			`json:"network"`
	BindDN 			string			`json:"bindDN"`
	BindPWD 		string			`json:"bindPWD"`
	InsecureNoSSL 	bool			`json:"insecureNoSSL"`
	RootPEM			string			`json:"rootPEM"`
	UserSearch		[]UserSearch 	`json:"userSearch"`
}

type UserSearch struct {
	Type 	string `json:"type"`
	BaseDN	string `json:"baseDN"`
	Filter	string `json:"filter"`
	LDAPUserAttrList []string `json:"ldapUserAttrList"`
}


func (c *conn) Close() {
	c.ldap.Close()
}

func (c *conn) Authenticate() error {
	return errors.New("not implemented")
}

func (c *conn) GetUserInformation(userSearchID, userID string) (*ldap.SearchResult, error) {
	c.logger.Debugf("SearchUserAttributesForClass(): userSearchID=%s, userID=%s", userSearchID, userID) 
	
	dn := fmt.Sprintf("uid=%s,%s",userID,c.userSearchIndex[userSearchID].BaseDN)
	c.logger.Debugf(">>> %s", dn)
	searchRequest := ldap.NewSearchRequest(
		dn, // The base dn to search
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		c.userSearchIndex[userSearchID].Filter, // The filter to apply
		c.userSearchIndex[userSearchID].LDAPUserAttrList,                                // A list attributes to retrieve
		nil,
	)
	return c.ldap.Search(searchRequest)
}
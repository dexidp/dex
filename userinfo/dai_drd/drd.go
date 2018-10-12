package dai_drd

import (	
	"errors"
	"github.com/dexidp/dex/userinfo"
	ldap "gopkg.in/ldap.v2"
	"github.com/sirupsen/logrus"

)

type conn struct {
	ldap	*ldap.Conn
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
	ldap, err := ldap.Dial(c.Network, c.HostAddress)
	if err != nil {
		logger.Errorf("cannot open LDAP connection")
		return nil, err
	}

	return &conn{ldap: ldap}, err
}

type LDAPConfig struct {
	HostAddress 	string	`json:"host"`
	Network 		string	`json:"network"`
	BindDN 			string	`json:"bindDN"`
	BindPWD 		string	`json:"bindPWD"`
	InsecureNoSSL 	bool	`json:"insecureNoSSL"`
}


func (c *conn) Close() {
	c.ldap.Close()
}

func (c *conn) Authenticate() error {
	return errors.New("not implemented")
}

func (c *conn) GetUserInformation() error {
	return errors.New("not implemented")
}
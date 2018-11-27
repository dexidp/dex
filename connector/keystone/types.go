package keystone

import (
	"github.com/sirupsen/logrus"
)

type Connector struct {
	Domain 			 string
	KeystoneHost 	 string
	KeystoneUsername string
	KeystonePassword string
	Logger 			 logrus.FieldLogger
}

type ConnectorData struct {
	AccessToken string `json:"accessToken"`
}

type KeystoneUser struct {
	Domain KeystoneDomain `json:"domain"`
	ID 	   string 		  `json:"id"`
	Name   string 		  `json:"name"`
}

type KeystoneDomain struct {
	ID string   `json:"id"`
	Name string `json:"name"`
}

type Config struct {
	Domain 			 string `json:"domain"`
	KeystoneHost 	 string `json:"keystoneHost"`
	KeystoneUsername string `json:"keystoneUsername"`
	KeystonePassword string `json:"keystonePassword"`
}

type LoginRequestData struct {
	Auth `json:"auth"`
}

type Auth struct {
	Identity `json:"identity"`
}

type Identity struct {
	Methods  []string `json:"methods"`
	Password 		  `json:"password"`
}

type Password struct {
	User `json:"user"`
}

type User struct {
	Name   string 	`json:"name"`
	Domain 			`json:"domain"`
	Password string `json:"password"`
}

type Domain struct {
	ID string `json:"id"`
}

type Token struct {
	IssuedAt  string 	   			 `json:"issued_at"`
	Extras 	  map[string]interface{} `json:"extras"`
	Methods   []string 	   			 `json:"methods"`
	ExpiresAt string 	   			 `json:"expires_at"`
	User 	  KeystoneUser 			 `json:"user"`
}

type TokenResponse struct {
	Token Token `json:"token"`
}

type CreateUserRequest struct {
	CreateUser CreateUserForm  `json:"user"`
}

type CreateUserForm struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Enabled  bool   `json:"enabled"`
	Password string `json:"password"`
	Roles  []string `json:"roles"`
}

type UserResponse struct {
	User CreateUserResponse `json:"user"`
}

type CreateUserResponse struct {
	Username string   `json:"username"`
	Name 	 string   `json:"name"`
	Roles 	 []string `json:"roles"`
	Enabled  bool     `json:"enabled"`
	Options  string   `json:"options"`
	ID 		 string   `json:"id"`
	Email 	 string   `json:"email"`
}

type CreateGroup struct {
	Group CreateGroupForm `json:"group"`
}

type CreateGroupForm struct {
	Description string `json:"description"`
	Name 		string `json:"name"`
}

type GroupID struct {
	Group GroupIDForm `json:"group"`
}

type GroupIDForm struct {
	ID string `json:"id"`
}

type Links struct {
	Self string `json:"self"`
	Previous string `json:"previous"`
	Next string `json:"next"`
}

type Group struct {
	DomainID 	string `json:"domain_id`
	Description string `json:"description"`
	ID 			string `json:"id"`
	Links 		Links  `json:"links"`
	Name 		string `json:"name"`
}

type GroupsResponse struct {
	Links  Links   `json:"links"`
	Groups []Group `json:"groups"`
}

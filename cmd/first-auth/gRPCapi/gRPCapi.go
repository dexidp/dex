package gRPCapi

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/dexidp/dex/api/v2"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type GrpcApiDex struct {
	DexClient api.DexClient
}

// TODO VAl: need to protect each global variable access via mutex

// NewGrpcApiDex - will create e new instance of our struct GrpcApiDex
// @params: hostAndPort, caPath, clientCrt, clientKey
// @returns: *GrpcApiDex, error
func NewGrpcApiDex(hostAndPort, caPath, clientCrt, clientKey string) (*GrpcApiDex, error) {

	cPool := x509.NewCertPool()
	caCert, err := ioutil.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("invalid CA crt file: %s", caPath)
	}
	if cPool.AppendCertsFromPEM(caCert) != true {
		return nil, fmt.Errorf("failed to parse CA crt")
	}

	clientCert, err := tls.LoadX509KeyPair(clientCrt, clientKey)
	if err != nil {
		return nil, fmt.Errorf("invalid client crt file: %s", caPath)
	}

	clientTLSConfig := &tls.Config{
		RootCAs:      cPool,
		Certificates: []tls.Certificate{clientCert},
	}
	creds := credentials.NewTLS(clientTLSConfig)

	conn, err := grpc.Dial(hostAndPort, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("dial: %v", err)
	}

	return &GrpcApiDex{
		DexClient: api.NewDexClient(conn),
	}, nil
}

// AddClient - will add a new Client in dex Database [dex.db/client]
// @params: id, name, secret, uris
// @returns: error
func (g *GrpcApiDex) AddClient(id string, name string, secret string, uris []string) error {
	// Create the client in Dex point of view
	client := &api.Client{
		Id:           id,
		Name:         name,
		Secret:       secret,
		RedirectUris: uris,
	}
	// Create the request to be send to Dex
	reqCreateClient := &api.CreateClientReq{
		Client: client,
	}
	// Send this request to Dex to create the Client into the DataBase
	if _, err := g.DexClient.CreateClient(context.TODO(), reqCreateClient); err != nil {
		return err
	}
	log.Debugf("A client (%+v) Has been created !\n", *reqCreateClient.Client)
	return nil
}

// UpdateClient - will change arguments fields of a specific clients given by this ID
// @params: id, name, uris
// @returns: error
func (g *GrpcApiDex) UpdateClient(id string, dataI ...interface{}) error {
	// catch what we need to update
	client := api.Client{}
	for _, data := range dataI {
		switch data.(type) {
		case *api.Client:
			client = data.(api.Client)
		case []string:
			// The field to update is the RedirectUris
			var tempUris []string
			for _, dataStr := range data.([]string) {
				if strings.HasPrefix(dataStr, "http") {
					tempUris = append(tempUris, dataStr)
				}
			}
			client.RedirectUris = tempUris
		case string:
			if strings.HasPrefix(data.(string), "http") {
				// The field to update is uris
				client.RedirectUris = []string{data.(string)}
			} else {
				client.Name = data.(string)
			}
		default:
			return errors.New("Error during uptading client: Invalid parameters")
		}
	}
	// Send request to update to Dex
	reqUpdateClient := &api.UpdateClientReq{
		Id:           id,
		Name:         client.Name,
		RedirectUris: client.RedirectUris,
	}
	if _, err := g.DexClient.UpdateClient(context.TODO(), reqUpdateClient); err != nil {
		return err
	}
	log.Debugf("A client (%+v) Has been updated !\n", *reqUpdateClient)
	return nil
}

// DeleteClient - will delete client from Dex
// @params: id
// @returns: error
func (g *GrpcApiDex) DeleteClient(id string) error {
	reqDeleteClient := &api.DeleteClientReq{
		Id: id,
	}
	if _, err := g.DexClient.DeleteClient(context.TODO(), reqDeleteClient); err != nil {
		log.Fatalf("failed to delete oauth2 client with id %s, %v\n", reqDeleteClient.Id, err)
	}
	log.Debugf("A client (%+v) has been deleted !\n", *reqDeleteClient)
	return nil
}

// AddPassword - will add a new User/Password in dex Database [dex.db/password]
// @params: email, password, username, userId
// @returns: error
func (g *GrpcApiDex) AddPassword(email string, pwd string, userName string, userId string) error {
	// Create the Password in Dex point of view
	hash_str, err := g._hashPassword(pwd)
	if err != nil {
		return err
	}
	password := &api.Password{
		Email:    email,
		Hash:     []byte(hash_str),
		Username: userName,
		UserId:   userId,
	}
	// Create the request to be send to Dex
	reqCreatePwd := api.CreatePasswordReq{
		Password: password,
	}
	// Send this request to Dex to create the Client into the DataBase
	if resp, err := g.DexClient.CreatePassword(context.TODO(), &reqCreatePwd); err != nil || (resp != nil && resp.AlreadyExists) {
		if resp != nil && resp.AlreadyExists {
			return fmt.Errorf("Can't create password because %s already exists\n", password.Email)
		}
		return err
	}
	log.Debugf("The password for %s has been created !\n", password.Email)
	return nil
}

// UpdatePassword - will update information linked to the email users password
// @params: interface{} could be password struct or slice of strings - for now just Hash and userName can be changed /!\ if password is given in entry, be sure that it is bcrypt hashed
// @returns: error
func (g *GrpcApiDex) UpdatePassword(email string, dataI ...interface{}) error {
	resp, err := g.DexClient.ListPasswords(context.TODO(), &api.ListPasswordReq{})
	if err != nil || resp.Passwords == nil {
		if resp.Passwords == nil {
			return errors.New("No passwords into the database")
		}
		return err
	}
	for _, password := range resp.Passwords {
		if password.Email == email {
			// catch what we need to update
			for _, data := range dataI {
				// Catch what we want to modify
				switch data.(type) {
				case *api.Password:
					password = data.(*api.Password)
				case []string:
					for _, dataStr := range data.([]string) {
						if strings.HasPrefix(dataStr, "$") {
							// The field to update is hash
							password.Hash = []byte(dataStr)
						} else {
							password.Username = dataStr
						}
					}
				case string:
					if strings.HasPrefix(data.(string), "$") {
						// The field to update is hash
						password.Hash = []byte(data.(string))
					} else {
						password.Username = data.(string)
					}
				default:
					return errors.New("Error during uptading password: Invalid parameters")
				}
			}
			// sent request to Dex to update
			reqUpdatePwd := api.UpdatePasswordReq{
				Email:       password.Email,
				NewHash:     password.Hash,
				NewUsername: password.Username,
			}
			if resp, err := g.DexClient.UpdatePassword(context.TODO(), &reqUpdatePwd); err != nil || (resp != nil && resp.NotFound) {
				if resp != nil && resp.NotFound {
					return fmt.Errorf("Can't updtate password because %s was not found\n", reqUpdatePwd.Email)
				}
				return err
			}
			log.Debugf("The password for %s has been updated !\n", reqUpdatePwd.Email)
			return nil
		}
	}
	return errors.New("Error during uptading passwords: No user foud with this email")
}

// DeletePassword - will delete paswword/user in Dex
// @params: user's email
// @returns: error
func (g *GrpcApiDex) DeletePassword(email string) error {
	reqDeletePwd := api.DeletePasswordReq{
		Email: email,
	}
	if resp, err := g.DexClient.DeletePassword(context.TODO(), &reqDeletePwd); err != nil || (resp != nil && resp.NotFound) {
		if resp != nil && resp.NotFound {
			return fmt.Errorf("Can't delete password because %s was not found\n", reqDeletePwd.Email)
		}
		return err
	}
	log.Debugf("The password for %s has been deleted !\n", reqDeletePwd.Email)
	return nil
}

// ListPassword - will list paswword/user from Dex
// @params: ,none
// @returns: []*Password, error
func (g *GrpcApiDex) ListPassword() ([]*api.Password, error) {
	resp, err := g.DexClient.ListPasswords(context.TODO(), &api.ListPasswordReq{})
	if err != nil || resp.Passwords == nil {
		if resp.Passwords == nil {
			return nil, nil
		}
		return nil, err
	}
	log.Debugf("Passwords has been found and returned !\n")
	return resp.Passwords, nil
}

// VerifyPassword - will check if paswword is correct for user's email in Dex
// @params: user's email, user's password
// @returns: bool, error
func (g *GrpcApiDex) VerifyPassword(email, password string) (bool, error) {
	reqVerifyPwd := api.VerifyPasswordReq{
		Email:    email,
		Password: password,
	}
	resp, err := g.DexClient.VerifyPassword(context.TODO(), &reqVerifyPwd)
	if err != nil || (resp != nil && resp.NotFound) {
		if resp != nil && resp.NotFound {
			return false, fmt.Errorf("Can't verify the password of %s because it was not found !\n", reqVerifyPwd.Email)
		}
		return false, err
	}
	log.Debugf("The password verifying for %s is %v !\n", reqVerifyPwd.Email, resp.Verified)
	return resp.Verified, nil
}

// _hashPassword - will convert a password in string to a bcrypt hash password in string
// @params: password to be convert
// @returns: password converted, error
func (g *GrpcApiDex) _hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// UserIdp - struct to define UserIdp
type UserIdp struct {
	IdpId    string
	InternId string
}

// AddUserIdp - will add a new User into the UserIdp table of dex.db
// @params: user UserIdp
// @returns: error
func (g *GrpcApiDex) AddUserIdp(user UserIdp) error {
	//Check if the Users already exists
	if exists, err := g.hasUserIdp(user.IdpId); err != nil || exists {
		if exists {
			log.Infof("The user with the id %s already exist in our database", user.IdpId)
			return nil
		}
		return err
	}
	// Create the user in Dex point of view
	u := &api.UserIdp{
		IdpId:    user.IdpId,
		InternId: user.InternId,
	}
	// Create the request to be send to Dex
	reqCreateUserIdp := &api.CreateUserIdpReq{
		UserIdp: u,
	}
	// Send this request to Dex to create the UserIdp into the DataBase
	if _, err := g.DexClient.CreateUserIdp(context.TODO(), reqCreateUserIdp); err != nil {
		return err
	}
	log.Debugf("An user (%+v) Has been created !\n", user)
	return nil
}

// GetUserIdp - will get the User into the UserIdp table of dex.db
// @params: idp_id string
// @returns: *api.UserIdp, error
func (g *GrpcApiDex) GetUserIdp(idp_id string) (*api.UserIdp, error) {
	resp, err := g.DexClient.ListUserIdp(context.TODO(), &api.ListUserIdpReq{})
	if err != nil {
		return nil, err
	}
	for _, user := range resp.UserIdps {
		if user.IdpId == idp_id {
			return user, nil
		}
	}
	return nil, errors.New("No UserIdp into the database")
}

// UpdateUserIdp - will change arguments fields of a specific UserIdp given by this ID
// @params: user UserIdp
// @returns: error
func (g *GrpcApiDex) UpdateUserIdp(user UserIdp) error {
	//Check if the Users exists
	if exists, err := g.hasUserIdp(user.IdpId); err != nil || !exists {
		if !exists {
			log.Infof("Cannot update UserIdp (%s), it doesn't exists", user.IdpId)
			return nil
		}
		return err
	}
	// Send request yo update user_idp
	reqUpdateUserIdp := &api.UpdateUserIdpReq{
		IdpId:    user.IdpId,
		InternId: user.InternId,
	}
	if _, err := g.DexClient.UpdateUserIdp(context.TODO(), reqUpdateUserIdp); err != nil {
		return err
	}
	return nil
}

// hasUserIdp - check if the User is or not into the UserIdp table of dex.db
// @params: idp_id string
// @returns: bool, error
func (g *GrpcApiDex) hasUserIdp(idp_id string) (bool, error) {
	resp, err := g.DexClient.ListUserIdp(context.TODO(), &api.ListUserIdpReq{})
	if err != nil {
		return false, err
	}
	for _, user := range resp.UserIdps {
		if user.IdpId == idp_id {
			return true, nil
		}
	}
	return false, nil
}

// DeleteUserIdp - will delete UserIdp from Dex
// @params: idp_id string
// @returns: error
func (g *GrpcApiDex) DeleteUserIdp(idp_id string) error {
	reqDeleteUserIdp := &api.DeleteUserIdpReq{
		IdpId: idp_id,
	}
	if _, err := g.DexClient.DeleteUserIdp(context.TODO(), reqDeleteUserIdp); err != nil {
		return err
	}
	log.Debugf("An User (%+v) has been deleted !\n", *reqDeleteUserIdp)
	return nil
}

// ListUserIdp - will list UserIdp from Dex
// @params: none
// @returns: []*aoi.UserIdp, error
func (g *GrpcApiDex) ListUserIdp() ([]*api.UserIdp, error) {
	resp, err := g.DexClient.ListUserIdp(context.TODO(), &api.ListUserIdpReq{})
	if err != nil || resp.UserIdps == nil {
		if resp.UserIdps == nil {
			return nil, nil
		}
		return nil, err
	}
	log.Debugf("UserIdp has been found and returned !\n")
	return resp.UserIdps, nil
}

// User - struct to define User
type User struct {
	InternId  string
	Pseudo    string
	Email     string
	AclTokens []string
}

// AddUser - will add a new User into the User table of dex.db
// @params: user User
// @returns: error
func (g *GrpcApiDex) AddUser(user User) error {
	//Check if the Users already exists
	if exists, err := g.hasUser(user.InternId); err != nil || exists {
		if exists {
			log.Infof("The user with the id %s already exist in our database", user.InternId)
			return nil
		}
		return err
	}
	// Create the user in Dex point of view
	u := &api.User{
		InternId:  user.InternId,
		Pseudo:    user.Pseudo,
		Email:     user.Email,
		AclTokens: user.AclTokens,
	}
	// Create the request to be send to Dex
	reqCreateUser := &api.CreateUserReq{
		User: u,
	}
	// Send this request to Dex to create the User into the DataBase
	if _, err := g.DexClient.CreateUser(context.TODO(), reqCreateUser); err != nil {
		return err
	}
	log.Debugf("An user (%+v) Has been created !\n", user)
	return nil
}

// GetUser - will get the User into the User table of dex.db
// @params: intern_id string
// @returns: *api.User, error
func (g *GrpcApiDex) GetUser(intern_id string) (*api.User, error) {
	resp, err := g.DexClient.ListUser(context.TODO(), &api.ListUserReq{})
	if err != nil {
		return nil, err
	}
	for _, user := range resp.Users {
		if user.InternId == intern_id {
			return user, nil
		}
	}
	return nil, errors.New("No User into the database")
}

// UpdateUser - will change arguments fields of a specific User given by this ID
// @params: user User
// @returns: error
func (g *GrpcApiDex) UpdateUser(user User) error {
	//Check if the Users exists
	if exists, err := g.hasUser(user.InternId); err != nil || !exists {
		if !exists {
			log.Infof("Cannot update User (%s), it doesn't exists", user.InternId)
			return nil
		}
		return err
	}
	// Send request yo update user_idp
	reqUpdateUser := &api.UpdateUserReq{
		InternId:  user.InternId,
		Pseudo:    user.Pseudo,
		Email:     user.Email,
		AclTokens: user.AclTokens,
	}
	if _, err := g.DexClient.UpdateUser(context.TODO(), reqUpdateUser); err != nil {
		return err
	}
	return nil
}

// hasUser - check if the User is or not into the User table of dex.db
// @params: intern_id string
// @returns: bool, error
func (g *GrpcApiDex) hasUser(intern_id string) (bool, error) {
	resp, err := g.DexClient.ListUser(context.TODO(), &api.ListUserReq{})
	if err != nil {
		return false, err
	}
	for _, user := range resp.Users {
		if user.InternId == intern_id {
			return true, nil
		}
	}
	return false, nil
}

// DeleteUser - will delete User from Dex
// @params: intern_id string
// @returns: error
func (g *GrpcApiDex) DeleteUser(intern_id string) error {
	reqDeleteUser := &api.DeleteUserReq{
		InternId: intern_id,
	}
	if _, err := g.DexClient.DeleteUser(context.TODO(), reqDeleteUser); err != nil {
		return err
	}
	log.Debugf("An User (%+v) has been deleted !\n", *reqDeleteUser)
	return nil
}

// ListUser - will list User from Dex
// @params: none
// @returns: []*api.User, error
func (g *GrpcApiDex) ListUser() ([]*api.User, error) {
	resp, err := g.DexClient.ListUser(context.TODO(), &api.ListUserReq{})
	if err != nil || resp.Users == nil {
		if resp.Users == nil {
			return nil, nil
		}
		return nil, err
	}
	log.Debugf("User has been found and returned !\n")
	return resp.Users, nil
}

// AclToken - struct to define AclToken
type AclToken struct {
	Id           string
	Desc         string
	MaxUser      string
	ClientTokens []string
}

// AddAclToken - will add a new AclToken into the AclToken table of dex.db
// @params: token AclToken
// @returns: error
func (g *GrpcApiDex) AddAclToken(token AclToken) error {
	//Check if the AclTokens already exists
	if exists, err := g.hasAclToken(token.Id); err != nil || exists {
		if exists {
			log.Infof("The acl token with the id %s already exist in our database", token.Id)
			return nil
		}
		return err
	}
	// Create the user in Dex point of view
	t := &api.AclToken{
		Id:           token.Id,
		Desc:         token.Desc,
		MaxUser:      token.MaxUser,
		ClientTokens: token.ClientTokens,
	}
	// Create the request to be send to Dex
	reqCreateAclToken := &api.CreateAclTokenReq{
		AclToken: t,
	}
	// Send this request to Dex to create the AclToken into the DataBase
	if _, err := g.DexClient.CreateAclToken(context.TODO(), reqCreateAclToken); err != nil {
		return err
	}
	log.Debugf("An user (%+v) Has been created !\n", token)
	return nil
}

// GetAclToken - will get the AclToken into the AclToken table of dex.db
// @params: id string
// @returns: *api.AclToken, error
func (g *GrpcApiDex) GetAclToken(id string) (*api.AclToken, error) {
	resp, err := g.DexClient.ListAclToken(context.TODO(), &api.ListAclTokenReq{})
	if err != nil {
		return nil, err
	}
	for _, token := range resp.AclTokens {
		if token.Id == id {
			return token, nil
		}
	}
	return nil, errors.New("No AclToken into the database")
}

// UpdateAclToken - will change arguments fields of a specific AclToken given by this ID
// @params: token AclToken
// @returns: error
func (g *GrpcApiDex) UpdateAclToken(token AclToken) error {
	//Check if the AclTokens exists
	if exists, err := g.hasAclToken(token.Id); err != nil || !exists {
		if !exists {
			log.Infof("Cannot update AclToken (%s), it doesn't exists", token.Id)
			return nil
		}
		return err
	}
	// Send request yo update user_idp
	reqUpdateAclToken := &api.UpdateAclTokenReq{
		Id:           token.Id,
		Desc:         token.Desc,
		MaxUser:      token.MaxUser,
		ClientTokens: token.ClientTokens,
	}
	if _, err := g.DexClient.UpdateAclToken(context.TODO(), reqUpdateAclToken); err != nil {
		return err
	}
	return nil
}

// hasAclToken - check if the AclToken is or not into the AclToken table of dex.db
// @params: id string
// @returns: bool, error
func (g *GrpcApiDex) hasAclToken(id string) (bool, error) {
	resp, err := g.DexClient.ListAclToken(context.TODO(), &api.ListAclTokenReq{})
	if err != nil {
		return false, err
	}
	for _, token := range resp.AclTokens {
		if token.Id == id {
			return true, nil
		}
	}
	return false, nil
}

// DeleteAclToken - will delete AclToken from Dex
// @params: id string
// @returns: error
func (g *GrpcApiDex) DeleteAclToken(id string) error {
	reqDeleteAclToken := &api.DeleteAclTokenReq{
		Id: id,
	}
	if _, err := g.DexClient.DeleteAclToken(context.TODO(), reqDeleteAclToken); err != nil {
		return err
	}
	log.Debugf("An AclToken (%+v) has been deleted !\n", *reqDeleteAclToken)
	return nil
}

// ListAclToken - will list AclToken from Dex
// @params: none
// @returns: []*AclToken, error
func (g *GrpcApiDex) ListAclToken() ([]*api.AclToken, error) {
	resp, err := g.DexClient.ListAclToken(context.TODO(), &api.ListAclTokenReq{})
	if err != nil || resp.AclTokens == nil {
		if resp.AclTokens == nil {
			return nil, nil
		}
		return nil, err
	}
	log.Debugf("AclToken has been found and returned !\n")
	return resp.AclTokens, nil
}

// ClientToken - struct to define ClientToken
type ClientToken struct {
	Id        string
	ClientId  string
	CreatedAt time.Time
	ExpiredAt time.Time
}

// AddClientToken - will add a new ClientToken into the ClientToken table of dex.db
// @params: token ClientToken
// @returns: error
func (g *GrpcApiDex) AddClientToken(token ClientToken) error {
	//Check if the ClientTokens already exists
	if exists, err := g.hasClientToken(token.Id); err != nil || exists {
		if exists {
			log.Infof("The client token with the id %s already exist in our database", token.Id)
			return nil
		}
		return err
	}
	// Create the user in Dex point of view
	t := &api.ClientToken{
		Id:        token.Id,
		ClientId:  token.ClientId,
		CreatedAt: token.CreatedAt.Unix(),
		ExpiredAt: token.ExpiredAt.Unix(),
	}
	// Create the request to be send to Dex
	reqCreateClientToken := &api.CreateClientTokenReq{
		ClientToken: t,
	}
	// Send this request to Dex to create the ClientToken into the DataBase
	if _, err := g.DexClient.CreateClientToken(context.TODO(), reqCreateClientToken); err != nil {
		return err
	}
	log.Debugf("A client token (%+v) Has been created !\n", token)
	return nil
}

// GetClientToken - will get the ClientToken into the ClientToken table of dex.db
// @params: id string
// @returns: *api.ClientToken, error
func (g *GrpcApiDex) GetClientToken(id string) (*api.ClientToken, error) {
	resp, err := g.DexClient.ListClientToken(context.TODO(), &api.ListClientTokenReq{})
	if err != nil {
		return nil, err
	}
	for _, token := range resp.ClientTokens {
		if token.Id == id {
			return token, nil
		}
	}
	return nil, errors.New("No ClientToken into the database")
}

// UpdateClientToken - will change arguments fields of a specific ClientToken given by this ID
// @params: token ClientToken
// @returns: error
func (g *GrpcApiDex) UpdateClientToken(token ClientToken) error {
	//Check if the ClientTokens exists
	if exists, err := g.hasClientToken(token.Id); err != nil || !exists {
		if !exists {
			log.Infof("Cannot update ClientToken (%s), it doesn't exists", token.Id)
			return nil
		}
		return err
	}
	// Send request yo update user_idp
	reqUpdateClientToken := &api.UpdateClientTokenReq{
		Id:        token.Id,
		ClientId:  token.ClientId,
		CreatedAt: token.CreatedAt.Unix(),
		ExpiredAt: token.ExpiredAt.Unix(),
	}
	if _, err := g.DexClient.UpdateClientToken(context.TODO(), reqUpdateClientToken); err != nil {
		return err
	}
	return nil
}

// hasClientToken - check if the ClientToken is or not into the ClientToken table of dex.db
// @params: id string
// @returns: bool, error
func (g *GrpcApiDex) hasClientToken(id string) (bool, error) {
	resp, err := g.DexClient.ListClientToken(context.TODO(), &api.ListClientTokenReq{})
	if err != nil {
		return false, err
	}
	for _, token := range resp.ClientTokens {
		if token.Id == id {
			return true, nil
		}
	}
	return false, nil
}

// DeleteClientToken - will delete ClientToken from Dex
// @params: id string
// @returns: error
func (g *GrpcApiDex) DeleteClientToken(id string) error {
	reqDeleteClientToken := &api.DeleteClientTokenReq{
		Id: id,
	}
	if _, err := g.DexClient.DeleteClientToken(context.TODO(), reqDeleteClientToken); err != nil {
		return err
	}
	log.Debugf("An ClientToken (%+v) has been deleted !\n", *reqDeleteClientToken)
	return nil
}

// ListClientToken - will list ClientToken from Dex
// @params: none
// @returns: []*ClientToken, error
func (g *GrpcApiDex) ListClientToken() ([]*api.ClientToken, error) {
	resp, err := g.DexClient.ListClientToken(context.TODO(), &api.ListClientTokenReq{})
	if err != nil || resp.ClientTokens == nil {
		if resp.ClientTokens == nil {
			return nil, nil
		}
		return nil, err
	}
	log.Debugf("ClientToken has been found and returned !\n")
	return resp.ClientTokens, nil
}

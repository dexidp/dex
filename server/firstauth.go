package server

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/storage"

	gomail "gopkg.in/mail.v2"
)

var (
	modeAuto   = "auto"
	modeManual = "manual"

	// ErrNotAllow is the error returned by firstAuth if a Token/User was not access to the client
	ErrNotAccess = errors.New("You are Not Allow to access to the client")
	//ErrExpiredToken is the error returneed when a client token has expired
	ErrExpiredToken = errors.New("The access to the client has been expired")
	//ErrMaxUsersReached is the error returneed when a client token has expired
	ErrMaxUsersReached = errors.New("The Token is already used by the maximum number of user")
)

//////////////////////////////////
/// FirstAuth Server functions ///
//////////////////////////////////

// getFirstAuthentificate - Will catch if the User is already into our database, if not, it will run the first Auth algorithm
// @params: r *http.Request, store storage.Storage, authReq storage.AuthRequest, oldUrl string
// @returns: redirectURL string, error
func (s *Server) getFirstAuthentificate(r *http.Request, authReq storage.AuthRequest, identity connector.Identity, oldUrl string) (bool, string, error) {
	redirectURL := oldUrl
	isAuth, err := Authenticate(r, s.storage, authReq, identity, s.firstAuth)
	if err != nil {
		errType := "You have not the access to " + authReq.ClientID
		errMsg := err.Error()
		redirectURL = path.Join(s.issuerURL.Path, "/firstauth/noaccess"+"?errType="+errType+"&errMsg="+errMsg+"&req="+authReq.ID)
	} else if !isAuth {
		redirectURL = path.Join(s.issuerURL.Path, "/firstauth/acltoken"+"?req="+authReq.ID)
	}
	// User Authenticate successfully
	return isAuth, redirectURL, err
}

// Authenticate - run the first Auth algorithm
// @params: r *http.Request, store storage.Storage
// @returns: isAuth bool, redirectPath string, error
func Authenticate(r *http.Request, s storage.Storage, oldAuthReq storage.AuthRequest, identity connector.Identity, configFirstAuth FirstAuth) (bool, error) {

	authReq, err := s.GetAuthRequest(r.FormValue("req"))
	if err != nil {
		if err == storage.ErrNotFound {
			authReq = oldAuthReq
		} else {
			return false, err
		}
	}
	userIdpID := identity.UserID + "_" + authReq.ConnectorID

	if dataFirstUser, err := s.GetUserIdp(userIdpID); err == storage.ErrNotFound {
		// UserIdp not found
		aclTokenID := r.FormValue("aclToken")
		if aclTokenID == "" {
			keys, ok := r.URL.Query()["aclToken"]
			if !ok || len(keys[0]) < 1 {
				return checkMode(s, authReq, identity, configFirstAuth)
			}
			aclTokenID = keys[0]
		}
		if aclToken, err := s.GetAclToken(aclTokenID); err == storage.ErrNotFound {
			// AclToken not found
			return checkMode(s, authReq, identity, configFirstAuth)
		} else if err != nil {
			return false, err
		} else {
			// AclToken found
			// Check if it is a gobal acl token
			if max, _ := strconv.Atoi(aclToken.MaxUser); max != 0 {
				if filled, err := isAclTokenFilled(s, aclToken); err != nil || filled {
					if filled {
						return false, ErrMaxUsersReached
					}
					return false, err
				}
			}
			if err := createUser(s, authReq, identity, []string{aclTokenID}); err != nil {
				return false, err
			}
			// Check if the User has access to the service
			for _, clientTokID := range aclToken.ClientTokens {
				clientTok, err := s.GetClientToken(clientTokID)
				if err != nil {
					return false, err
				}
				// expired ?
				if clientTok.ExpiredAt.Unix() < time.Now().Unix() {
					return false, ErrExpiredToken
				}
				if clientTok.ClientID == authReq.ClientID {
					return true, nil
				}
			}
			return false, ErrNotAccess
		}
	} else if err != nil {
		log.Println("An error occured whan searching an user")
		return false, err
	} else {
		// Check if the User has access to the service
		access, err := hasUserAccess(s, authReq, identity, dataFirstUser.InternID, configFirstAuth)
		if err == nil && access {
			return true, nil
		}
		return false, err
	}
}

// hasUserAccess - check if the user has access to the client thanks to this acl token and if his tokens are not expired
// @params: store storage.Storage, internId string
// @returns: bool(have or not access), error (internal error OR ErrExpiredToken OR ErrNotAccess)
func hasUserAccess(s storage.Storage, authReq storage.AuthRequest, identity connector.Identity, internId string, configFirstAuth FirstAuth) (bool, error) {
	var err_expired error
	// Catch data of the internal user
	user, err := s.GetUser(internId)
	if err != nil {
		return false, err
	}
	for _, aclTokID := range user.AclTokens {
		// Catch the aclToken
		aclToken, err := s.GetAclToken(aclTokID)
		if err != nil {
			return false, err
		}
		// Catch all clientToken link to the aclToken
		for _, clientTokID := range aclToken.ClientTokens {
			clientTok, err := s.GetClientToken(clientTokID)
			if err == nil && clientTok.ClientID == authReq.ClientID {
				// expired ?
				if clientTok.ExpiredAt.Unix() < time.Now().Unix() {
					err_expired = ErrExpiredToken
				} else {
					// Have access
					return true, nil
				}
			}
		}
	}

	hasGToken, err := hasGlobalAclToken(s, authReq, identity, configFirstAuth)
	if err != nil {
		return false, err
	}
	if hasGToken {
		// Have access
		return true, nil
	}

	if err_expired != nil {
		return false, err_expired
	}
	return false, ErrNotAccess
}

// hasTokenExpired - check if a client token has expired
// @params: store storage.Storage, clientTokID string
// @returns: hasAccess bool, err error (ErrExpiredToken OR internal error)
func hasTokenExpired(s storage.Storage, clientTokID string) (bool, error) {
	clientTok, err := s.GetClientToken(clientTokID)
	if err != nil {
		return true, err
	}
	if clientTok.ExpiredAt.Unix() < time.Now().Unix() {
		return true, ErrExpiredToken
	}
	return false, nil
}

// isAclTokenFilled - check if an acl_token is filled
// @params: s storage.Storage, aclToken storage.AclToken
// @returns: isFilled bool, err error (ErrMaxUsersReached OR internal error)
func isAclTokenFilled(s storage.Storage, aclToken storage.AclToken) (bool, error) {
	// Get the list of internal users
	users, err := s.ListUser()
	if err != nil {
		return true, err
	}
	count := 0
	// Check how many user already use the acl token
	for _, user := range users {
		for _, userTok := range user.AclTokens {
			if userTok == aclToken.ID {
				count++
				break
			}
		}
	}
	maxUser, _ := strconv.Atoi(aclToken.MaxUser)
	if count < maxUser {
		return false, nil
	}
	return true, nil
}

// hasGlobalAclToken - check into database if there is a global acl tokens (MaxUSer = 0) with a client token for the specific client request
// @params:  s storage, clientID string
// @returns: bool, error
func hasGlobalAclToken(s storage.Storage, authReq storage.AuthRequest, identity connector.Identity, configFirstAuth FirstAuth) (bool, error) {

	// check if we are in auto mode
	if configFirstAuth.Mode == modeAuto && configFirstAuth.Default != nil {
		// Catch default connectors into app's config
		for _, defaultData := range configFirstAuth.Default {
			if defaultData.Connector == authReq.ConnectorID {
				// Create the user
				if err := createUser(s, authReq, identity, []string{defaultData.AclToken}); err != nil {
					return false, err
				}
				return true, nil
			}
		}
	} else {
		tokenList, err := s.ListAclToken()
		if err != nil {
			return false, err
		}
		// Find all global token
		for _, token := range tokenList {
			if maxUser, _ := strconv.Atoi(token.MaxUser); maxUser == 0 {
				// Check if it has a client_token for this client
				for _, clientTokID := range token.ClientTokens {
					clientTok, err := s.GetClientToken(clientTokID)
					if err != nil {
						return false, err
					}
					if clientTok.ClientID == authReq.ClientID {
						// Create the user
						err := createUser(s, authReq, identity, []string{token.ID})
						if err != nil {
							return false, err
						}
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}

// createUser - will create an userIdp linked to an internal user
// @params: s storage.Storage, authReq storage.AuthRequest, aclTokenID []string
// @returns: error
func createUser(s storage.Storage, authReq storage.AuthRequest, identity connector.Identity, aclTokenID []string) error {
	userIdpID := identity.UserID + "_" + authReq.ConnectorID
	userPseudo := identity.Username
	userEmail := identity.Email
	internID := storage.NewID()
	if err := s.CreateUserIdp(storage.UserIdp{IdpID: userIdpID, InternID: internID}); err != nil {
		if err == storage.ErrAlreadyExists {
			//Update the user instead of created one
			userIdp, err := s.GetUserIdp(userIdpID)
			if err != nil {
				return err
			}
			updater := func(old storage.User) (storage.User, error) {
				old.AclTokens = append(old.AclTokens, aclTokenID...)
				return old, nil
			}
			// Update the user
			if err := s.UpdateUser(userIdp.InternID, updater); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if err := s.CreateUser(storage.User{InternID: internID, Pseudo: userPseudo, Email: userEmail, AclTokens: aclTokenID}); err != nil {
		return err
	}
	return nil
}

// createAclToken - will create an acl token into the dB
// @params:s storage.Storage, desc string, clientTokenID []string
// @returns: actTokenID, err
func createAclToken(s storage.Storage, desc string, maxUser int, clientTokenID []string) (string, error) {
	aclTokenID := storage.NewID()
	max := strconv.Itoa(maxUser)
	if err := s.CreateAclToken(storage.AclToken{ID: aclTokenID, Desc: desc, MaxUser: max, ClientTokens: clientTokenID}); err != nil {
		return "", err
	}
	return aclTokenID, nil
}

// createClientToken - will create a client token into dB
// @params: s storage.Storage, authReq storage.AuthRequest, aclTokenID []string
// @returns: error
func createClientToken(s storage.Storage, idClient string, days int) (string, error) {
	clientTokenID := storage.NewID()
	created := time.Now()
	expired := created.AddDate(0, 0, days)
	if err := s.CreateClientToken(storage.ClientToken{ID: clientTokenID, ClientID: idClient, CreatedAt: created, ExpiredAt: expired}); err != nil {
		return "", err
	}
	return clientTokenID, nil
}

// checkMode - check used and act adequate to the mode
// @params: mode string, returnURL string
// @returns: isAuth bool, returnURL string, error
func checkMode(s storage.Storage, authReq storage.AuthRequest, identity connector.Identity, configFirstAuth FirstAuth) (bool, error) {
	if configFirstAuth.Mode == modeAuto {
		// Check if there are global acl_token
		hasGToken, err := hasGlobalAclToken(s, authReq, identity, configFirstAuth)
		if err != nil {
			return false, err
		}
		if hasGToken {
			return true, nil
		}
	}
	return false, nil
}

///////////////////////////////////
/// FirstAuth Handler functions ///
///////////////////////////////////

func (s *Server) handleFirstAuthToken(w http.ResponseWriter, r *http.Request) {
	authReq, err := s.storage.GetAuthRequest(r.FormValue("req"))
	if err != nil {
		s.logger.Errorf("Failed to get auth request: %v", err)
		s.renderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}

	// Redirect to Join  page if asked
	if r.FormValue("invitation") == "asked" {
		redirectURL := path.Join(s.issuerURL.Path, "/firstauth/join") + "?req=" + authReq.ID
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}

	// Recreate the Identity struct for the authentication from data into authReq
	identity := connector.Identity{
		UserID:            authReq.Claims.UserID,
		Username:          authReq.Claims.Username,
		PreferredUsername: authReq.Claims.PreferredUsername,
		Email:             authReq.Claims.Email,
		EmailVerified:     authReq.Claims.EmailVerified,
		Groups:            authReq.Claims.Groups,
	}

	switch r.Method {
	case http.MethodGet:
		if err := s.templates.firstAuthToken(r, w, authReq.ID, "", "Token"); err != nil {
			s.logger.Errorf("Server template error: %v", err)
		}
	case http.MethodPost:
		redirectURL := path.Join(s.issuerURL.Path, "/approval") + "?req=" + authReq.ID
		_, redirectURL, err = s.getFirstAuthentificate(r, authReq, identity, redirectURL)
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}

func (s *Server) handleFirstAuthNoAccess(w http.ResponseWriter, r *http.Request) {
	errType := r.FormValue("errType")
	errMsg := r.FormValue("errMsg")
	if err := s.templates.firstAuthNoAccess(r, w, errType, errMsg); err != nil {
		s.logger.Errorf("Server template error: %v", err)
	}
}

func (s *Server) handleFirstAuthJoin(w http.ResponseWriter, r *http.Request) {
	authReq, err := s.storage.GetAuthRequest(r.FormValue("req"))
	if err != nil {
		s.logger.Errorf("Failed to get auth request: %v", err)
		s.renderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}
	switch r.Method {
	case http.MethodGet:
		claims := authReq.Claims
		if err := s.templates.firstAuthJoin(r, w, claims.UserID, claims.Username, claims.Email); err != nil {
			s.logger.Errorf("Server template error: %v", err)
		}
	case http.MethodPost:
		info := infoInvitation{
			email:       r.FormValue("email"),
			firstname:   r.FormValue("firstname"),
			lastname:    r.FormValue("lastname"),
			company:     r.FormValue("company"),
			companysize: r.FormValue("companysize"),
			board:       r.FormValue("boards"),
			aglMember:   r.FormValue("aglmember"),
			reason:      r.FormValue("reason"),
		}
		// send a request invitation by mail
		if err := s.sendMail(info); err != nil {
			s.logger.Errorf("SMTP error: %v", err)
		}

		url := strings.SplitN(strings.Split(authReq.RedirectURI, "http://")[1], "/", 2)[0]
		http.Redirect(w, r, "http://"+url, http.StatusSeeOther)
	}
}

///////////////////////
/// Mailer function ///
///////////////////////

type infoInvitation struct {
	email       string
	firstname   string
	lastname    string
	company     string
	companysize string
	board       string
	aglMember   string
	reason      string
}

// sendMail - create a mesage with personal information an send it by mail to the receiver
// @params: i infoInvitation
// @returns: error
func (s *Server) sendMail(i infoInvitation) error {
	m := gomail.NewMessage()
	m.SetHeader("To", s.firstAuth.Mailer.Receiver)
	m.SetHeader("From", s.firstAuth.Mailer.User)
	m.SetHeader("Subject", "Request Invitation")
	body := fmt.Sprintf("The user %s (%s) ask to join the program. Here is this information:\n\tfirstname: %s\n\tlastname: %s\n\tcompany: %s\n\tcompanysize: %s\n\tboard: %s\n\taglMember: %s\n\treason: %s\n\t",
		i.firstname, i.email, i.firstname, i.lastname, i.company, i.companysize, i.board, i.aglMember, i.reason)
	m.SetBody("text/plain", body)
	clPwd, _ := base64.StdEncoding.DecodeString(s.firstAuth.Mailer.Password)
	d := gomail.NewDialer(s.firstAuth.Mailer.Host, s.firstAuth.Mailer.Port, s.firstAuth.Mailer.User, string(clPwd[:len(clPwd)-1]))
	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

/////////////////////////////////////////
/// FirstAuth HTML Template functions ///
/////////////////////////////////////////

func (t *templates) firstAuthToken(r *http.Request, w http.ResponseWriter, authReqID, lastToken, token string) error {
	data := struct {
		AuthReqID   string
		Token       string
		TokenPrompt string
		ReqPath     string
	}{authReqID, lastToken, token, r.URL.Path}
	return renderTemplate(w, t.firstAuthTokenTmpl, data)
}

func (t *templates) firstAuthJoin(r *http.Request, w http.ResponseWriter, userID, username, email string) error {
	data := struct {
		UserID   string
		Username string
		Email    string
		ReqPath  string
	}{userID, username, email, r.URL.Path}
	return renderTemplate(w, t.firstAuthJoinTmpl, data)
}

func (t *templates) firstAuthNoAccess(r *http.Request, w http.ResponseWriter, errType, errMsg string) error {
	data := struct {
		ErrType string
		ErrMsg  string
		ReqPath string
	}{errType, errMsg, r.URL.Path}
	return renderTemplate(w, t.errorTmpl, data)
}

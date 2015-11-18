package api

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/pkg/log"
	schema "github.com/coreos/dex/schema/workerschema"
	"github.com/coreos/dex/user"
)

var (
	errorMap = map[error]Error{
		user.ErrorNotFound:       ErrorResourceNotFound,
		user.ErrorDuplicateEmail: ErrorDuplicateEmail,
		user.ErrorInvalidEmail:   ErrorInvalidEmail,
		client.ErrorNotFound:     ErrorInvalidClient,
	}

	ErrorInvalidEmail = newError("invalid_email", "invalid email.", http.StatusBadRequest)

	ErrorInvalidClient = newError("invalid_client", "invalid email.", http.StatusBadRequest)

	ErrorDuplicateEmail   = newError("duplicate_email", "Email already in use.", http.StatusBadRequest)
	ErrorResourceNotFound = newError("resource_not_found", "Resource could not be found.", http.StatusNotFound)

	ErrorUnauthorized = newError("unauthorized", "The given user and client are not authorized to make this request.", http.StatusUnauthorized)

	ErrorMaxResultsTooHigh = newError("max_results_too_high", fmt.Sprintf("The max number of results per page is %d", maxUsersPerPage), http.StatusBadRequest)

	ErrorInvalidRedirectURL = newError("invalid_redirect_url", "The provided redirect URL is invalid for the given client", http.StatusBadRequest)
)

const (
	maxUsersPerPage = 100
)

func internalError(internal error) Error {
	return Error{
		Code:     http.StatusInternalServerError,
		Type:     "server_error",
		Desc:     "",
		Internal: internal,
	}
}

func newError(typ string, desc string, code int) Error {
	return Error{
		Code: code,
		Type: typ,
		Desc: desc,
	}
}

// Error is the error type returned by AdminAPI methods.
type Error struct {
	Type string

	// The HTTP Code to return for this type of error.
	Code int

	Desc string

	// The underlying error - not to be consumed by external users.
	Internal error
}

func (e Error) Error() string {
	return fmt.Sprintf("%v: Desc: %v Internal: %v", e.Type, e.Desc, e.Internal)
}

// UsersAPI is the user management API for Dex administrators.

// All calls take a Creds object with the ClientID of the calling app and the
// calling User. It is assumed that the clientID has already validated as an
// admin app before calling.
type UsersAPI struct {
	manager            *user.Manager
	localConnectorID   string
	clientIdentityRepo client.ClientIdentityRepo
	emailer            Emailer
}

type Emailer interface {
	SendInviteEmail(string, url.URL, string) (*url.URL, error)
}

type Creds struct {
	ClientID string
	User     user.User
}

func NewUsersAPI(manager *user.Manager, cir client.ClientIdentityRepo, emailer Emailer, localConnectorID string) *UsersAPI {
	return &UsersAPI{
		manager:            manager,
		clientIdentityRepo: cir,
		localConnectorID:   localConnectorID,
		emailer:            emailer,
	}
}

func (u *UsersAPI) GetUser(creds Creds, id string) (schema.User, error) {
	log.Infof("userAPI: GetUser")

	if !u.Authorize(creds) {
		return schema.User{}, ErrorUnauthorized
	}

	usr, err := u.manager.Get(id)

	if err != nil {
		return schema.User{}, mapError(err)
	}

	return userToSchemaUser(usr), nil
}

func (u *UsersAPI) DisableUser(creds Creds, userID string, disable bool) (schema.UserDisableResponse, error) {
	log.Infof("userAPI: DisableUser")
	if !u.Authorize(creds) {
		return schema.UserDisableResponse{}, ErrorUnauthorized
	}

	if err := u.manager.Disable(userID, disable); err != nil {
		return schema.UserDisableResponse{}, mapError(err)
	}

	return schema.UserDisableResponse{
		Ok: true,
	}, nil
}

func (u *UsersAPI) CreateUser(creds Creds, usr schema.User, redirURL url.URL) (schema.UserCreateResponse, error) {
	log.Infof("userAPI: CreateUser")
	if !u.Authorize(creds) {
		return schema.UserCreateResponse{}, ErrorUnauthorized
	}

	hash, err := generateTempHash()
	if err != nil {
		return schema.UserCreateResponse{}, mapError(err)
	}

	metadata, err := u.clientIdentityRepo.Metadata(creds.ClientID)
	if err != nil {
		return schema.UserCreateResponse{}, mapError(err)
	}

	validRedirURL, err := client.ValidRedirectURL(&redirURL, metadata.RedirectURLs)
	if err != nil {
		return schema.UserCreateResponse{}, ErrorInvalidRedirectURL
	}

	id, err := u.manager.CreateUser(schemaUserToUser(usr), user.Password(hash), u.localConnectorID)
	if err != nil {
		return schema.UserCreateResponse{}, mapError(err)
	}

	userUser, err := u.manager.Get(id)
	if err != nil {
		return schema.UserCreateResponse{}, mapError(err)
	}

	usr = userToSchemaUser(userUser)

	url, err := u.emailer.SendInviteEmail(usr.Email, validRedirURL, creds.ClientID)

	// An email is sent only if we don't get a link and there's no error.
	emailSent := err == nil && url == nil

	var resetLink string
	if url != nil {
		resetLink = url.String()
	}

	return schema.UserCreateResponse{
		User:              &usr,
		EmailSent:         emailSent,
		ResetPasswordLink: resetLink,
	}, nil
}

func (u *UsersAPI) ListUsers(creds Creds, maxResults int, nextPageToken string) ([]*schema.User, string, error) {
	log.Infof("userAPI: ListUsers")

	if !u.Authorize(creds) {
		return nil, "", ErrorUnauthorized
	}

	if maxResults > maxUsersPerPage {
		return nil, "", ErrorMaxResultsTooHigh
	}

	users, tok, err := u.manager.List(user.UserFilter{}, maxResults, nextPageToken)
	if err != nil {
		return nil, "", mapError(err)
	}

	list := []*schema.User{}
	for _, usr := range users {
		schemaUsr := userToSchemaUser(usr)
		list = append(list, &schemaUsr)
	}

	return list, tok, nil
}

func (u *UsersAPI) Authorize(creds Creds) bool {
	return creds.User.Admin && !creds.User.Disabled
}

func userToSchemaUser(usr user.User) schema.User {
	return schema.User{
		Id:            usr.ID,
		Email:         usr.Email,
		EmailVerified: usr.EmailVerified,
		DisplayName:   usr.DisplayName,
		Admin:         usr.Admin,
		Disabled:      usr.Disabled,
		CreatedAt:     usr.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func schemaUserToUser(usr schema.User) user.User {
	return user.User{
		ID:            usr.Id,
		Email:         usr.Email,
		EmailVerified: usr.EmailVerified,
		DisplayName:   usr.DisplayName,
		Admin:         usr.Admin,
		Disabled:      usr.Disabled,
	}
}

func mapError(e error) error {
	if mapped, ok := errorMap[e]; ok {
		return mapped
	}
	return internalError(e)
}

func generateTempHash() (string, error) {
	b := make([]byte, 32)
	n, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	if n != 32 {
		return "", errors.New("unable to read enough random bytes")
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

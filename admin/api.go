// package admin provides an implementation of the API described in auth/schema/adminschema.
package admin

import (
	"net/http"

	"github.com/coreos/go-oidc/oidc"
	"github.com/go-gorp/gorp"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/db"
	"github.com/coreos/dex/schema/adminschema"
	"github.com/coreos/dex/user"
	"github.com/coreos/dex/user/manager"
)

// AdminAPI provides the logic necessary to implement the Admin API.
type AdminAPI struct {
	userManager        *manager.UserManager
	userRepo           user.UserRepo
	passwordInfoRepo   user.PasswordInfoRepo
	clientIdentityRepo client.ClientIdentityRepo
	localConnectorID   string
}

// TODO(ericchiang): Swap the DbMap for a storage interface. See #278

func NewAdminAPI(dbMap *gorp.DbMap, userManager *manager.UserManager, localConnectorID string) *AdminAPI {
	if localConnectorID == "" {
		panic("must specify non-blank localConnectorID")
	}
	return &AdminAPI{
		userManager:        userManager,
		userRepo:           db.NewUserRepo(dbMap),
		passwordInfoRepo:   db.NewPasswordInfoRepo(dbMap),
		clientIdentityRepo: db.NewClientIdentityRepo(dbMap),
		localConnectorID:   localConnectorID,
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
	return e.Type
}

func errorMaker(typ string, desc string, code int) func(internal error) Error {
	return func(internal error) Error {
		return Error{
			Type:     typ,
			Code:     code,
			Desc:     desc,
			Internal: internal,
		}
	}
}

var (
	errorMap = map[error]func(error) Error{
		user.ErrorNotFound:       errorMaker("resource_not_found", "Resource could not be found.", http.StatusNotFound),
		user.ErrorDuplicateEmail: errorMaker("bad_request", "Email already in use.", http.StatusBadRequest),
		user.ErrorInvalidEmail:   errorMaker("bad_request", "invalid email.", http.StatusBadRequest),
	}
)

func (a *AdminAPI) GetAdmin(id string) (adminschema.Admin, error) {
	usr, err := a.userRepo.Get(nil, id)

	if err != nil {
		return adminschema.Admin{}, mapError(err)
	}

	pwi, err := a.passwordInfoRepo.Get(nil, id)
	if err != nil {
		return adminschema.Admin{}, mapError(err)
	}

	return adminschema.Admin{
		Id:       id,
		Email:    usr.Email,
		Password: string(pwi.Password),
	}, nil
}

func (a *AdminAPI) CreateAdmin(admn adminschema.Admin) (string, error) {
	userID, err := a.userManager.CreateUser(user.User{
		Email: admn.Email,
		Admin: true}, user.Password(admn.Password), a.localConnectorID)
	if err != nil {
		return "", mapError(err)
	}
	return userID, nil
}

func (a *AdminAPI) GetState() (adminschema.State, error) {
	state := adminschema.State{}

	admins, err := a.userRepo.GetAdminCount(nil)
	if err != nil {
		return adminschema.State{}, err
	}

	state.AdminUserCreated = admins > 0

	return state, nil
}

func (a *AdminAPI) CreateClient(req adminschema.ClientCreateRequest) (adminschema.ClientCreateResponse, error) {
	cli, err := adminschema.MapSchemaClientToClient(*req.Client)
	if err != nil {
		// TODO  should be 400s
		return adminschema.ClientCreateResponse{}, mapError(err)
	}

	if err := cli.Metadata.Valid(); err != nil {
		// TODO make sure this is not 500
		return adminschema.ClientCreateResponse{}, mapError(err)
	}

	// metadata is guarenteed to have at least one redirect_uri by earlier validation.
	id, err := oidc.GenClientID(cli.Metadata.RedirectURIs[0].Host)
	if err != nil {
		return adminschema.ClientCreateResponse{}, mapError(err)
	}

	cli.Credentials.ID = id

	creds, err := a.clientIdentityRepo.New(cli)
	if err != nil {
		return adminschema.ClientCreateResponse{}, mapError(err)
	}

	req.Client.Id = creds.ID
	req.Client.Secret = creds.Secret
	return adminschema.ClientCreateResponse{
		Client: req.Client,
	}, nil

	//	github.com/coreos/dex/integrationoidc.ClientRegistrationResponse{ClientID: c.ID, ClientSecret: c.Secret, ClientMetadata: req.Client.Metadata}, nil
}

func mapError(e error) error {
	if mapped, ok := errorMap[e]; ok {
		return mapped(e)
	}
	return Error{
		Code:     http.StatusInternalServerError,
		Type:     "server_error",
		Desc:     "",
		Internal: e,
	}
}

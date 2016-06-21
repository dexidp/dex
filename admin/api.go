// Package admin provides an implementation of the API described in auth/schema/adminschema.
package admin

import (
	"net/http"

	"github.com/coreos/dex/client"
	clientmanager "github.com/coreos/dex/client/manager"
	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/schema/adminschema"
	"github.com/coreos/dex/user"
	usermanager "github.com/coreos/dex/user/manager"
)

// AdminAPI provides the logic necessary to implement the Admin API.
type AdminAPI struct {
	userManager         *usermanager.UserManager
	userRepo            user.UserRepo
	passwordInfoRepo    user.PasswordInfoRepo
	connectorConfigRepo connector.ConnectorConfigRepo
	clientRepo          client.ClientRepo
	clientManager       *clientmanager.ClientManager
	localConnectorID    string
}

func NewAdminAPI(userRepo user.UserRepo, pwiRepo user.PasswordInfoRepo, clientRepo client.ClientRepo, connectorConfigRepo connector.ConnectorConfigRepo, userManager *usermanager.UserManager, clientManager *clientmanager.ClientManager, localConnectorID string) *AdminAPI {
	if localConnectorID == "" {
		panic("must specify non-blank localConnectorID")
	}
	return &AdminAPI{
		userManager:         userManager,
		userRepo:            userRepo,
		passwordInfoRepo:    pwiRepo,
		clientRepo:          clientRepo,
		clientManager:       clientManager,
		connectorConfigRepo: connectorConfigRepo,
		localConnectorID:    localConnectorID,
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
	ErrorMissingClient = errorMaker("bad_request", "The 'client' cannot be empty", http.StatusBadRequest)(nil)

	ErrorInvalidClientFunc = errorMaker("bad_request", "Your client could not be validated.", http.StatusBadRequest)

	errorMap = map[error]func(error) Error{
		client.ErrorMissingRedirectURI: errorMaker("bad_request", "Non-public clients must have at least one redirect URI", http.StatusBadRequest),

		client.ErrorPublicClientRedirectURIs: errorMaker("bad_request", "Public clients cannot specify redirect URIs", http.StatusBadRequest),

		client.ErrorPublicClientMissingName: errorMaker("bad_request", "Public clients require a ClientName", http.StatusBadRequest),

		user.ErrorNotFound:       errorMaker("resource_not_found", "Resource could not be found.", http.StatusNotFound),
		user.ErrorDuplicateEmail: errorMaker("bad_request", "Email already in use.", http.StatusBadRequest),
		user.ErrorInvalidEmail:   errorMaker("bad_request", "invalid email.", http.StatusBadRequest),

		adminschema.ErrorInvalidRedirectURI: errorMaker("bad_request", "invalid redirectURI.", http.StatusBadRequest),
		adminschema.ErrorInvalidLogoURI:     errorMaker("bad_request", "invalid logoURI.", http.StatusBadRequest),
		adminschema.ErrorInvalidClientURI:   errorMaker("bad_request", "invalid clientURI.", http.StatusBadRequest),
		adminschema.ErrorNoRedirectURI:      errorMaker("bad_request", "invalid redirectURI.", http.StatusBadRequest),
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
	if req.Client == nil {
		return adminschema.ClientCreateResponse{}, ErrorMissingClient
	}

	cli, err := adminschema.MapSchemaClientToClient(*req.Client)
	if err != nil {
		return adminschema.ClientCreateResponse{}, mapError(err)
	}

	creds, err := a.clientManager.New(cli, &clientmanager.ClientOptions{
		TrustedPeers: req.Client.TrustedPeers,
	})
	if err != nil {
		return adminschema.ClientCreateResponse{}, mapError(err)
	}

	req.Client.Id = creds.ID
	req.Client.Secret = creds.Secret
	return adminschema.ClientCreateResponse{
		Client: req.Client,
	}, nil
}

func (a *AdminAPI) SetConnectors(connectorConfigs []connector.ConnectorConfig) error {
	return a.connectorConfigRepo.Set(connectorConfigs)
}

func (a *AdminAPI) GetConnectors() ([]connector.ConnectorConfig, error) {
	return a.connectorConfigRepo.All()
}

func mapError(e error) error {
	switch t := e.(type) {
	case client.ValidationError:
		return ErrorInvalidClientFunc(t)
	default:
	}

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

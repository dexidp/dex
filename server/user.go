package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/oidc"
	"github.com/julienschmidt/httprouter"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/pkg/log"
	schema "github.com/coreos/dex/schema/workerschema"
	"github.com/coreos/dex/user"
	"github.com/coreos/dex/user/api"
)

const (
	defaultMaxResults = 20
)

var (
	UsersSubTree        = "/users"
	UsersListEndpoint   = addBasePath(UsersSubTree) + "/"
	UsersCreateEndooint = addBasePath(UsersSubTree)
	UsersGetEndpoint    = addBasePath(UsersSubTree + "/:id")
)

type UserMgmtServer struct {
	api         *api.UsersAPI
	jwtvFactory JWTVerifierFactory
	um          *user.Manager
	cir         client.ClientIdentityRepo
}

func NewUserMgmtServer(userMgmtAPI *api.UsersAPI, jwtvFactory JWTVerifierFactory, um *user.Manager, cir client.ClientIdentityRepo) *UserMgmtServer {
	return &UserMgmtServer{
		api:         userMgmtAPI,
		jwtvFactory: jwtvFactory,
		um:          um,
		cir:         cir,
	}
}

func (s *UserMgmtServer) HTTPHandler() http.Handler {
	r := httprouter.New()
	r.GET(UsersListEndpoint, s.listUsers)
	r.POST(UsersCreateEndooint, s.createUser)
	r.GET(UsersGetEndpoint, s.getUser)
	return r
}

func (s *UserMgmtServer) listUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	creds, err := s.getCreds(r)
	if err != nil {
		s.writeError(w, err)
		return
	}

	nextPageToken := r.URL.Query().Get("nextPageToken")

	maxResults, err := intFromQuery(r.URL.Query(), "maxResults", defaultMaxResults)

	if err != nil {
		writeAPIError(w, http.StatusBadRequest,
			newAPIError(errorInvalidRequest, "maxResults must be an integer"))
		return
	}

	users, nextPageToken, err := s.api.ListUsers(creds, maxResults, nextPageToken)
	if err != nil {
		s.writeError(w, err)
		return
	}

	usersResponse := schema.UsersResponse{
		Users:         users,
		NextPageToken: nextPageToken,
	}
	writeResponseWithBody(w, http.StatusOK, usersResponse)
}

func (s *UserMgmtServer) getUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	creds, err := s.getCreds(r)
	if err != nil {
		s.writeError(w, err)
		return
	}

	id := ps.ByName("id")
	if id == "" {
		writeAPIError(w, http.StatusBadRequest,
			newAPIError(errorInvalidRequest, "id is required"))
		return
	}

	usr, err := s.api.GetUser(creds, id)
	if err != nil {
		s.writeError(w, err)
		return
	}
	userResponse := schema.UserResponse{
		User: &usr,
	}
	writeResponseWithBody(w, http.StatusOK, userResponse)
}

func (s *UserMgmtServer) createUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	creds, err := s.getCreds(r)
	if err != nil {
		s.writeError(w, err)
		return
	}

	createReq := schema.UserCreateRequest{}
	err = json.NewDecoder(r.Body).Decode(&createReq)
	if err != nil {
		writeInvalidRequest(w, "cannot parse JSON body")
		return
	}

	redirURL, err := url.Parse(createReq.RedirectURL)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest,
			newAPIError(errorInvalidRequest, "redirectURL must be a valid URL"))
		return
	}

	createdResponse, err := s.api.CreateUser(creds, *createReq.User, *redirURL)
	if err != nil {
		s.writeError(w, err)
		return
	}

	writeResponseWithBody(w, http.StatusOK, createdResponse)
}

func (s *UserMgmtServer) writeError(w http.ResponseWriter, err error) {
	log.Errorf("Error calling user management API: %v: ", err)
	if apiErr, ok := err.(api.Error); ok {
		writeAPIError(w, apiErr.Code, newAPIError(apiErr.Type, apiErr.Desc))
		return
	}

	writeAPIError(w, http.StatusInternalServerError, newAPIError(errorServerError, err.Error()))
}

func (s *UserMgmtServer) getCreds(r *http.Request) (api.Creds, error) {
	token, err := oidc.ExtractBearerToken(r)
	if err != nil {
		log.Errorf("userMgmtServer: GetCreds err: %q", err)
		return api.Creds{}, api.ErrorUnauthorized
	}

	jwt, err := jose.ParseJWT(token)
	if err != nil {
		log.Errorf("userMgmtServer: GetCreds err: %q", err)
		return api.Creds{}, api.ErrorUnauthorized
	}

	claims, err := jwt.Claims()
	if err != nil {
		log.Errorf("userMgmtServer: GetCreds err: %q", err)
		return api.Creds{}, api.ErrorUnauthorized
	}

	clientID, ok, err := claims.StringClaim("aud")
	if err != nil {
		log.Errorf("userMgmtServer: GetCreds err: %q", err)
		return api.Creds{}, err
	}
	if !ok || clientID == "" {
		return api.Creds{}, errors.New("no aud(client ID) claim")
	}

	verifier := s.jwtvFactory(clientID)
	if err := verifier.Verify(jwt); err != nil {
		log.Errorf("userMgmtServer: GetCreds err: %q", err)
		return api.Creds{}, api.ErrorUnauthorized
	}

	sub, ok, err := claims.StringClaim("sub")
	if err != nil {
		log.Errorf("userMgmtServer: GetCreds err: %q", err)
		return api.Creds{}, err
	}
	if !ok || sub == "" {
		return api.Creds{}, api.ErrorUnauthorized
	}

	usr, err := s.um.Get(sub)
	if err != nil {
		if err == user.ErrorNotFound {
			return api.Creds{}, api.ErrorUnauthorized
		}
		log.Errorf("userMgmtServer: GetCreds err: %q", err)
		return api.Creds{}, err
	}

	if usr.Disabled {
		return api.Creds{}, api.ErrorUnauthorized
	}

	isAdmin, err := s.cir.IsDexAdmin(clientID)
	if err != nil {
		log.Errorf("userMgmtServer: GetCreds err: %q", err)
		return api.Creds{}, err
	}
	if !isAdmin {
		return api.Creds{}, api.ErrorUnauthorized
	}

	return api.Creds{
		ClientID: clientID,
		User:     usr,
	}, nil
}

func intFromQuery(ps url.Values, name string, defaultVal int) (int, error) {
	s := ps.Get(name)
	if s == "" {
		return defaultVal, nil
	}
	return strconv.Atoi(s)
}

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

	clientmanager "github.com/coreos/dex/client/manager"
	"github.com/coreos/dex/pkg/log"
	schema "github.com/coreos/dex/schema/workerschema"
	"github.com/coreos/dex/user"
	"github.com/coreos/dex/user/api"
	usermanager "github.com/coreos/dex/user/manager"
)

const (
	defaultMaxResults = 20
)

var (
	UsersSubTree                  = "/users"
	UsersListEndpoint             = addBasePath(UsersSubTree)
	UsersCreateEndpoint           = addBasePath(UsersSubTree)
	UsersGetEndpoint              = addBasePath(UsersSubTree + "/:id")
	UsersDisableEndpoint          = addBasePath(UsersSubTree + "/:id/disable")
	UsersResendInvitationEndpoint = addBasePath(UsersSubTree + "/:id/resend-invitation")
	AccountSubTree                = "/account"
	AccountListRefreshTokens      = addBasePath(AccountSubTree + "/:userid/refresh")
	AccountRevokeRefreshToken     = addBasePath(AccountSubTree + "/:userid/refresh/:clientid")
)

type UserMgmtServer struct {
	api         *api.UsersAPI
	jwtvFactory JWTVerifierFactory
	um          *usermanager.UserManager
	cm          *clientmanager.ClientManager
}

func NewUserMgmtServer(userMgmtAPI *api.UsersAPI, jwtvFactory JWTVerifierFactory, um *usermanager.UserManager, cm *clientmanager.ClientManager) *UserMgmtServer {
	return &UserMgmtServer{
		api:         userMgmtAPI,
		jwtvFactory: jwtvFactory,
		um:          um,
		cm:          cm,
	}
}

func (s *UserMgmtServer) HTTPHandler() http.Handler {
	r := httprouter.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	r.GET(UsersListEndpoint, s.authAdminUser(s.listUsers))
	r.POST(UsersCreateEndpoint, s.authAdminUser(s.createUser))
	r.POST(UsersDisableEndpoint, s.authAdminUser(s.disableUser))
	r.GET(UsersGetEndpoint, s.authAdminUser(s.getUser))
	r.POST(UsersResendInvitationEndpoint, s.authAdminUser(s.resendInvitationEmail))

	r.GET(AccountListRefreshTokens, s.authAccount(s.listClientsWithRefreshTokens))
	r.DELETE(AccountRevokeRefreshToken, s.authAccount(s.revokeRefreshTokensForClient))
	return r
}

func (s *UserMgmtServer) authAdminUser(handle authedHandle) httprouter.Handle {
	return s.authAPIHandle(handle, true)
}

func (s *UserMgmtServer) authAccount(handle authedHandle) httprouter.Handle {
	return s.authAPIHandle(handle, false)
}

// authedHandle is an HTTP handle which requires requests to be authenticated as an admin user.
type authedHandle func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, creds api.Creds)

// authAPIHandle is a middleware function with authenticates an HTTP request before passing
// it along to the authedHandle.
//
// The authorization checks for an ID token bearer token in the request header, requiring the
// audience (aud claim) be a client ID of an admin client.
//
// If requiresAdmin is true, the subject identifier (sub claim) of the ID token provided must be
// that of an admin user.
func (s *UserMgmtServer) authAPIHandle(handle authedHandle, requiresAdmin bool) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		creds, err := s.getCreds(r, requiresAdmin)
		if err != nil {
			s.writeError(w, err)
			return
		}
		if creds.User.Disabled || (requiresAdmin && !creds.User.Admin) {
			s.writeError(w, api.ErrorUnauthorized)
			return
		}
		handle(w, r, ps, creds)
	}
}

func (s *UserMgmtServer) listUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params, creds api.Creds) {
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

func (s *UserMgmtServer) getUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params, creds api.Creds) {
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

func (s *UserMgmtServer) createUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params, creds api.Creds) {
	createReq := schema.UserCreateRequest{}
	if err := json.NewDecoder(r.Body).Decode(&createReq); err != nil {
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

func (s *UserMgmtServer) disableUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params, creds api.Creds) {
	id := ps.ByName("id")
	if id == "" {
		writeAPIError(w, http.StatusBadRequest,
			newAPIError(errorInvalidRequest, "id is required"))
		return
	}

	disableReq := schema.UserDisableRequest{}
	if err := json.NewDecoder(r.Body).Decode(&disableReq); err != nil {
		writeInvalidRequest(w, "cannot parse JSON body")
	}

	resp, err := s.api.DisableUser(creds, id, disableReq.Disable)
	if err != nil {
		s.writeError(w, err)
		return
	}

	writeResponseWithBody(w, http.StatusOK, resp)
}

func (s *UserMgmtServer) resendInvitationEmail(w http.ResponseWriter, r *http.Request, ps httprouter.Params, creds api.Creds) {
	id := ps.ByName("id")
	if id == "" {
		writeAPIError(w, http.StatusBadRequest, newAPIError(errorInvalidRequest, "id is required"))
		return
	}
	resendEmailInvitationReq := schema.ResendEmailInvitationRequest{}
	if err := json.NewDecoder(r.Body).Decode(&resendEmailInvitationReq); err != nil {
		writeInvalidRequest(w, "cannot parse JSON body")
		return
	}

	redirURL, err := url.Parse(resendEmailInvitationReq.RedirectURL)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest,
			newAPIError(errorInvalidRequest, "redirectURL must be a valid URL"))
		return
	}

	resendEmailInvitationResponse, err := s.api.ResendEmailInvitation(creds, id, *redirURL)
	if err != nil {
		s.writeError(w, err)
		return
	}

	writeResponseWithBody(w, http.StatusOK, resendEmailInvitationResponse)
}

func (s *UserMgmtServer) listClientsWithRefreshTokens(w http.ResponseWriter, r *http.Request, ps httprouter.Params, creds api.Creds) {
	clients, err := s.api.ListClientsWithRefreshTokens(creds, ps.ByName("userid"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeResponseWithBody(w, http.StatusOK, schema.RefreshClientList{Clients: clients})
}

func (s *UserMgmtServer) revokeRefreshTokensForClient(w http.ResponseWriter, r *http.Request, ps httprouter.Params, creds api.Creds) {
	if err := s.api.RevokeRefreshTokensForClient(creds, ps.ByName("userid"), ps.ByName("clientid")); err != nil {
		s.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK) // NOTE (ericchiang): http.StatusNoContent or return an empty JSON object?
}

func (s *UserMgmtServer) writeError(w http.ResponseWriter, err error) {
	log.Errorf("Error calling user management API: %v: ", err)
	if apiErr, ok := err.(api.Error); ok {
		writeAPIError(w, apiErr.Code, newAPIError(apiErr.Type, apiErr.Desc))
		return
	}

	writeAPIError(w, http.StatusInternalServerError, newAPIError(errorServerError, err.Error()))
}

func (s *UserMgmtServer) getCreds(r *http.Request, requiresAdmin bool) (api.Creds, error) {
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

	isAdmin, err := s.cm.IsDexAdmin(clientID)
	if err != nil {
		log.Errorf("userMgmtServer: GetCreds err: %q", err)
		return api.Creds{}, err
	}
	if requiresAdmin && !isAdmin {
		return api.Creds{}, api.ErrorForbidden
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

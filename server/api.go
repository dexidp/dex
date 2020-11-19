package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/dexidp/dex/api/v2"
	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/version"
)

// apiVersion increases every time a new call is added to the API. Clients should use this info
// to determine if the server supports specific features.
const apiVersion = 2

const (
	// recCost is the recommended bcrypt cost, which balances hash strength and
	// efficiency.
	recCost = 12

	// upBoundCost is a sane upper bound on bcrypt cost determined by benchmarking:
	// high enough to ensure secure encryption, low enough to not put unnecessary
	// load on a dex server.
	upBoundCost = 16
)

// NewAPI returns a server which implements the gRPC API interface.
func NewAPI(s storage.Storage, logger log.Logger) api.DexServer {
	return dexAPI{
		s:      s,
		logger: logger,
	}
}

type dexAPI struct {
	s      storage.Storage
	logger log.Logger
}

func (d dexAPI) CreateClient(ctx context.Context, req *api.CreateClientReq) (*api.CreateClientResp, error) {
	if req.Client == nil {
		return nil, errors.New("no client supplied")
	}

	if req.Client.Id == "" {
		req.Client.Id = storage.NewID()
	}
	if req.Client.Secret == "" {
		req.Client.Secret = storage.NewID() + storage.NewID()
	}

	c := storage.Client{
		ID:           req.Client.Id,
		Secret:       req.Client.Secret,
		RedirectURIs: req.Client.RedirectUris,
		TrustedPeers: req.Client.TrustedPeers,
		Public:       req.Client.Public,
		Name:         req.Client.Name,
		LogoURL:      req.Client.LogoUrl,
	}
	if err := d.s.CreateClient(c); err != nil {
		if err == storage.ErrAlreadyExists {
			return &api.CreateClientResp{AlreadyExists: true}, nil
		}
		d.logger.Errorf("api: failed to create client: %v", err)
		return nil, fmt.Errorf("create client: %v", err)
	}

	return &api.CreateClientResp{
		Client: req.Client,
	}, nil
}

func (d dexAPI) UpdateClient(ctx context.Context, req *api.UpdateClientReq) (*api.UpdateClientResp, error) {
	if req.Id == "" {
		return nil, errors.New("update client: no client ID supplied")
	}

	err := d.s.UpdateClient(req.Id, func(old storage.Client) (storage.Client, error) {
		if req.RedirectUris != nil {
			old.RedirectURIs = req.RedirectUris
		}
		if req.TrustedPeers != nil {
			old.TrustedPeers = req.TrustedPeers
		}
		if req.Name != "" {
			old.Name = req.Name
		}
		if req.LogoUrl != "" {
			old.LogoURL = req.LogoUrl
		}
		return old, nil
	})
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.UpdateClientResp{NotFound: true}, nil
		}
		d.logger.Errorf("api: failed to update the client: %v", err)
		return nil, fmt.Errorf("update client: %v", err)
	}
	return &api.UpdateClientResp{}, nil
}

func (d dexAPI) DeleteClient(ctx context.Context, req *api.DeleteClientReq) (*api.DeleteClientResp, error) {
	err := d.s.DeleteClient(req.Id)
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.DeleteClientResp{NotFound: true}, nil
		}
		d.logger.Errorf("api: failed to delete client: %v", err)
		return nil, fmt.Errorf("delete client: %v", err)
	}
	return &api.DeleteClientResp{}, nil
}

// checkCost returns an error if the hash provided does not meet lower or upper
// bound cost requirements.
func checkCost(hash []byte) error {
	actual, err := bcrypt.Cost(hash)
	if err != nil {
		return fmt.Errorf("parsing bcrypt hash: %v", err)
	}
	if actual < bcrypt.DefaultCost {
		return fmt.Errorf("given hash cost = %d does not meet minimum cost requirement = %d", actual, bcrypt.DefaultCost)
	}
	if actual > upBoundCost {
		return fmt.Errorf("given hash cost = %d is above upper bound cost = %d, recommended cost = %d", actual, upBoundCost, recCost)
	}
	return nil
}

func (d dexAPI) CreatePassword(ctx context.Context, req *api.CreatePasswordReq) (*api.CreatePasswordResp, error) {
	if req.Password == nil {
		return nil, errors.New("no password supplied")
	}
	if req.Password.UserId == "" {
		return nil, errors.New("no user ID supplied")
	}
	if req.Password.Hash != nil {
		if err := checkCost(req.Password.Hash); err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("no hash of password supplied")
	}

	p := storage.Password{
		Email:    req.Password.Email,
		Hash:     req.Password.Hash,
		Username: req.Password.Username,
		UserID:   req.Password.UserId,
	}
	if err := d.s.CreatePassword(p); err != nil {
		if err == storage.ErrAlreadyExists {
			return &api.CreatePasswordResp{AlreadyExists: true}, nil
		}
		d.logger.Errorf("api: failed to create password: %v", err)
		return nil, fmt.Errorf("create password: %v", err)
	}

	return &api.CreatePasswordResp{}, nil
}

func (d dexAPI) UpdatePassword(ctx context.Context, req *api.UpdatePasswordReq) (*api.UpdatePasswordResp, error) {
	if req.Email == "" {
		return nil, errors.New("no email supplied")
	}
	if req.NewHash == nil && req.NewUsername == "" {
		return nil, errors.New("nothing to update")
	}

	if req.NewHash != nil {
		if err := checkCost(req.NewHash); err != nil {
			return nil, err
		}
	}

	updater := func(old storage.Password) (storage.Password, error) {
		if req.NewHash != nil {
			old.Hash = req.NewHash
		}

		if req.NewUsername != "" {
			old.Username = req.NewUsername
		}

		return old, nil
	}

	if err := d.s.UpdatePassword(req.Email, updater); err != nil {
		if err == storage.ErrNotFound {
			return &api.UpdatePasswordResp{NotFound: true}, nil
		}
		d.logger.Errorf("api: failed to update password: %v", err)
		return nil, fmt.Errorf("update password: %v", err)
	}

	return &api.UpdatePasswordResp{}, nil
}

func (d dexAPI) DeletePassword(ctx context.Context, req *api.DeletePasswordReq) (*api.DeletePasswordResp, error) {
	if req.Email == "" {
		return nil, errors.New("no email supplied")
	}

	err := d.s.DeletePassword(req.Email)
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.DeletePasswordResp{NotFound: true}, nil
		}
		d.logger.Errorf("api: failed to delete password: %v", err)
		return nil, fmt.Errorf("delete password: %v", err)
	}
	return &api.DeletePasswordResp{}, nil
}

func (d dexAPI) GetVersion(ctx context.Context, req *api.VersionReq) (*api.VersionResp, error) {
	return &api.VersionResp{
		Server: version.Version,
		Api:    apiVersion,
	}, nil
}

func (d dexAPI) ListPasswords(ctx context.Context, req *api.ListPasswordReq) (*api.ListPasswordResp, error) {
	passwordList, err := d.s.ListPasswords()
	if err != nil {
		d.logger.Errorf("api: failed to list passwords: %v", err)
		return nil, fmt.Errorf("list passwords: %v", err)
	}

	var passwords []*api.Password
	for _, password := range passwordList {
		p := api.Password{
			Email:    password.Email,
			Username: password.Username,
			UserId:   password.UserID,
		}
		passwords = append(passwords, &p)
	}

	return &api.ListPasswordResp{
		Passwords: passwords,
	}, nil
}

func (d dexAPI) VerifyPassword(ctx context.Context, req *api.VerifyPasswordReq) (*api.VerifyPasswordResp, error) {
	if req.Email == "" {
		return nil, errors.New("no email supplied")
	}

	if req.Password == "" {
		return nil, errors.New("no password to verify supplied")
	}

	password, err := d.s.GetPassword(req.Email)
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.VerifyPasswordResp{
				NotFound: true,
			}, nil
		}
		d.logger.Errorf("api: there was an error retrieving the password: %v", err)
		return nil, fmt.Errorf("verify password: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword(password.Hash, []byte(req.Password)); err != nil {
		d.logger.Infof("api: password check failed: %v", err)
		return &api.VerifyPasswordResp{
			Verified: false,
		}, nil
	}
	return &api.VerifyPasswordResp{
		Verified: true,
	}, nil
}

func (d dexAPI) ListRefresh(ctx context.Context, req *api.ListRefreshReq) (*api.ListRefreshResp, error) {
	id := new(internal.IDTokenSubject)
	if err := internal.Unmarshal(req.UserId, id); err != nil {
		d.logger.Errorf("api: failed to unmarshal ID Token subject: %v", err)
		return nil, err
	}

	var refreshTokenRefs []*api.RefreshTokenRef
	offlineSessions, err := d.s.GetOfflineSessions(id.UserId, id.ConnId)
	if err != nil {
		if err == storage.ErrNotFound {
			// This means that this user-client pair does not have a refresh token yet.
			// An empty list should be returned instead of an error.
			return &api.ListRefreshResp{
				RefreshTokens: refreshTokenRefs,
			}, nil
		}
		d.logger.Errorf("api: failed to list refresh tokens %t here : %v", err == storage.ErrNotFound, err)
		return nil, err
	}

	for _, session := range offlineSessions.Refresh {
		r := api.RefreshTokenRef{
			Id:        session.ID,
			ClientId:  session.ClientID,
			CreatedAt: session.CreatedAt.Unix(),
			LastUsed:  session.LastUsed.Unix(),
		}
		refreshTokenRefs = append(refreshTokenRefs, &r)
	}

	return &api.ListRefreshResp{
		RefreshTokens: refreshTokenRefs,
	}, nil
}

func (d dexAPI) RevokeRefresh(ctx context.Context, req *api.RevokeRefreshReq) (*api.RevokeRefreshResp, error) {
	id := new(internal.IDTokenSubject)
	if err := internal.Unmarshal(req.UserId, id); err != nil {
		d.logger.Errorf("api: failed to unmarshal ID Token subject: %v", err)
		return nil, err
	}

	var (
		refreshID string
		notFound  bool
	)
	updater := func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
		refreshRef := old.Refresh[req.ClientId]
		if refreshRef == nil || refreshRef.ID == "" {
			d.logger.Errorf("api: refresh token issued to client %q for user %q not found for deletion", req.ClientId, id.UserId)
			notFound = true
			return old, storage.ErrNotFound
		}

		refreshID = refreshRef.ID

		// Remove entry from Refresh list of the OfflineSession object.
		delete(old.Refresh, req.ClientId)

		return old, nil
	}

	if err := d.s.UpdateOfflineSessions(id.UserId, id.ConnId, updater); err != nil {
		if err == storage.ErrNotFound {
			return &api.RevokeRefreshResp{NotFound: true}, nil
		}
		d.logger.Errorf("api: failed to update offline session object: %v", err)
		return nil, err
	}

	if notFound {
		return &api.RevokeRefreshResp{NotFound: true}, nil
	}

	// Delete the refresh token from the storage
	//
	// TODO(ericchiang): we don't have any good recourse if this call fails.
	// Consider garbage collection of refresh tokens with no associated ref.
	if err := d.s.DeleteRefresh(refreshID); err != nil {
		d.logger.Errorf("failed to delete refresh token: %v", err)
		return nil, err
	}

	return &api.RevokeRefreshResp{}, nil
}

// API functions to manage user_idp table

func (d dexAPI) CreateUserIdp(ctx context.Context, req *api.CreateUserIdpReq) (*api.CreateUserIdpResp, error) {
	if req.UserIdp == nil {
		return nil, errors.New("no idpUser supplied")
	}

	if req.UserIdp.IdpId == "" {
		return nil, errors.New("no idp_id supplied")
	}
	if req.UserIdp.InternId == "" {
		return nil, errors.New("no intern_id supplied")
	}

	u := storage.UserIdp{
		IdpID:    req.UserIdp.IdpId,
		InternID: req.UserIdp.InternId,
	}
	if err := d.s.CreateUserIdp(u); err != nil {
		if err == storage.ErrAlreadyExists {
			return &api.CreateUserIdpResp{AlreadyExists: true}, nil
		}
		d.logger.Errorf("api: failed to create UserIdp: %v", err)
		return nil, fmt.Errorf("create UserIdp: %v", err)
	}

	return &api.CreateUserIdpResp{
		UserIdp: req.UserIdp,
	}, nil
}

func (d dexAPI) UpdateUserIdp(ctx context.Context, req *api.UpdateUserIdpReq) (*api.UpdateUserIdpResp, error) {
	if req.IdpId == "" {
		return nil, errors.New("update UserIdp: no IdpID supplied")
	}

	err := d.s.UpdateUserIdp(req.IdpId, func(old storage.UserIdp) (storage.UserIdp, error) {
		if req.InternId != "" {
			old.InternID = req.InternId
		}
		return old, nil
	})

	if err != nil {
		if err == storage.ErrNotFound {
			return &api.UpdateUserIdpResp{NotFound: true}, nil
		}
		d.logger.Errorf("api: failed to update the UserIdp: %v", err)
		return nil, fmt.Errorf("update UserIdp: %v", err)
	}
	return &api.UpdateUserIdpResp{}, nil
}

func (d dexAPI) ListUserIdp(ctx context.Context, req *api.ListUserIdpReq) (*api.ListUserIdpResp, error) {
	userList, err := d.s.ListUserIdp()
	if err != nil {
		d.logger.Errorf("api: failed to list UserIdp: %v", err)
		return nil, fmt.Errorf("list UserIdp: %v", err)
	}

	var users []*api.UserIdp
	for _, user := range userList {
		u := api.UserIdp{
			IdpId:    user.IdpID,
			InternId: user.InternID,
		}
		users = append(users, &u)
	}

	return &api.ListUserIdpResp{
		UserIdps: users,
	}, nil
}

func (d dexAPI) DeleteUserIdp(ctx context.Context, req *api.DeleteUserIdpReq) (*api.DeleteUserIdpResp, error) {
	err := d.s.DeleteUserIdp(req.IdpId)
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.DeleteUserIdpResp{NotFound: true}, nil
		}
		d.logger.Errorf("api: failed to delete UserIdp: %v", err)
		return nil, fmt.Errorf("delete UserIdp: %v", err)
	}
	return &api.DeleteUserIdpResp{}, nil
}

// API functions to manage user table

func (d dexAPI) CreateUser(ctx context.Context, req *api.CreateUserReq) (*api.CreateUserResp, error) {
	if req.User == nil {
		return nil, errors.New("no User supplied")
	}

	if req.User.InternId == "" {
		return nil, errors.New("no InternId supplied")
	}

	u := storage.User{
		InternID:  req.User.InternId,
		Pseudo:    req.User.Pseudo,
		Email:     req.User.Email,
		AclTokens: req.User.AclTokens,
	}
	if err := d.s.CreateUser(u); err != nil {
		if err == storage.ErrAlreadyExists {
			return &api.CreateUserResp{AlreadyExists: true}, nil
		}
		d.logger.Errorf("api: failed to create User: %v", err)
		return nil, fmt.Errorf("create User: %v", err)
	}

	return &api.CreateUserResp{
		User: req.User,
	}, nil
}

func (d dexAPI) UpdateUser(ctx context.Context, req *api.UpdateUserReq) (*api.UpdateUserResp, error) {
	if req.InternId == "" {
		return nil, errors.New("update User: no InternId supplied")
	}

	err := d.s.UpdateUser(req.InternId, func(old storage.User) (storage.User, error) {
		if req.Pseudo != "" {
			old.Pseudo = req.Pseudo
		}
		if req.Email != "" {
			old.Email = req.Email
		}
		if req.AclTokens != nil {
			old.AclTokens = req.AclTokens
		}
		return old, nil
	})

	if err != nil {
		if err == storage.ErrNotFound {
			return &api.UpdateUserResp{NotFound: true}, nil
		}
		d.logger.Errorf("api: failed to update the User: %v", err)
		return nil, fmt.Errorf("update User: %v", err)
	}
	return &api.UpdateUserResp{}, nil
}

func (d dexAPI) ListUser(ctx context.Context, req *api.ListUserReq) (*api.ListUserResp, error) {
	userList, err := d.s.ListUser()
	if err != nil {
		d.logger.Errorf("api: failed to list User: %v", err)
		return nil, fmt.Errorf("list User: %v", err)
	}

	var users []*api.User
	for _, user := range userList {
		u := api.User{
			InternId:  user.InternID,
			Pseudo:    user.Pseudo,
			Email:     user.Email,
			AclTokens: user.AclTokens,
		}
		users = append(users, &u)
	}

	return &api.ListUserResp{
		Users: users,
	}, nil
}

func (d dexAPI) DeleteUser(ctx context.Context, req *api.DeleteUserReq) (*api.DeleteUserResp, error) {
	err := d.s.DeleteUser(req.InternId)
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.DeleteUserResp{NotFound: true}, nil
		}
		d.logger.Errorf("api: failed to delete User: %v", err)
		return nil, fmt.Errorf("delete User: %v", err)
	}
	return &api.DeleteUserResp{}, nil
}

// API functions to manage acl_token table

func (d dexAPI) CreateAclToken(ctx context.Context, req *api.CreateAclTokenReq) (*api.CreateAclTokenResp, error) {
	if req.AclToken == nil {
		return nil, errors.New("no AclToken supplied")
	}

	if req.AclToken.Id == "" {
		return nil, errors.New("no Id supplied")
	}

	u := storage.AclToken{
		ID:           req.AclToken.Id,
		Desc:         req.AclToken.Desc,
		MaxUser:      req.AclToken.MaxUser,
		ClientTokens: req.AclToken.ClientTokens,
	}
	if err := d.s.CreateAclToken(u); err != nil {
		if err == storage.ErrAlreadyExists {
			return &api.CreateAclTokenResp{AlreadyExists: true}, nil
		}
		d.logger.Errorf("api: failed to create AclToken: %v", err)
		return nil, fmt.Errorf("create AclToken: %v", err)
	}

	return &api.CreateAclTokenResp{
		AclToken: req.AclToken,
	}, nil
}

func (d dexAPI) UpdateAclToken(ctx context.Context, req *api.UpdateAclTokenReq) (*api.UpdateAclTokenResp, error) {
	if req.Id == "" {
		return nil, errors.New("update AclToken: no Id supplied")
	}

	err := d.s.UpdateAclToken(req.Id, func(old storage.AclToken) (storage.AclToken, error) {
		if req.Desc != "" {
			old.Desc = req.Desc
		}
		if req.MaxUser != "" {
			old.MaxUser = req.MaxUser
		}
		if req.ClientTokens != nil {
			old.ClientTokens = req.ClientTokens
		}
		return old, nil
	})

	if err != nil {
		if err == storage.ErrNotFound {
			return &api.UpdateAclTokenResp{NotFound: true}, nil
		}
		d.logger.Errorf("api: failed to update the AclToken: %v", err)
		return nil, fmt.Errorf("update AclToken: %v", err)
	}
	return &api.UpdateAclTokenResp{}, nil
}

func (d dexAPI) ListAclToken(ctx context.Context, req *api.ListAclTokenReq) (*api.ListAclTokenResp, error) {
	tokenList, err := d.s.ListAclToken()
	if err != nil {
		d.logger.Errorf("api: failed to list AclToken: %v", err)
		return nil, fmt.Errorf("list AclToken: %v", err)
	}

	var tokens []*api.AclToken
	for _, token := range tokenList {
		t := api.AclToken{
			Id:           token.ID,
			Desc:         token.Desc,
			MaxUser:      token.MaxUser,
			ClientTokens: token.ClientTokens,
		}
		tokens = append(tokens, &t)
	}

	return &api.ListAclTokenResp{
		AclTokens: tokens,
	}, nil
}

func (d dexAPI) DeleteAclToken(ctx context.Context, req *api.DeleteAclTokenReq) (*api.DeleteAclTokenResp, error) {
	err := d.s.DeleteAclToken(req.Id)
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.DeleteAclTokenResp{NotFound: true}, nil
		}
		d.logger.Errorf("api: failed to delete AclToken: %v", err)
		return nil, fmt.Errorf("delete AclToken: %v", err)
	}
	return &api.DeleteAclTokenResp{}, nil
}

// API functions to manage client_token table

func (d dexAPI) CreateClientToken(ctx context.Context, req *api.CreateClientTokenReq) (*api.CreateClientTokenResp, error) {
	if req.ClientToken == nil {
		return nil, errors.New("no ClientToken supplied")
	}

	if req.ClientToken.Id == "" {
		return nil, errors.New("no Id supplied")
	}

	u := storage.ClientToken{
		ID:        req.ClientToken.Id,
		ClientID:  req.ClientToken.ClientId,
		CreatedAt: time.Unix(req.ClientToken.CreatedAt, 0),
		ExpiredAt: time.Unix(req.ClientToken.ExpiredAt, 0),
	}
	if err := d.s.CreateClientToken(u); err != nil {
		if err == storage.ErrAlreadyExists {
			return &api.CreateClientTokenResp{AlreadyExists: true}, nil
		}
		d.logger.Errorf("api: failed to create ClientToken: %v", err)
		return nil, fmt.Errorf("create ClientToken: %v", err)
	}

	return &api.CreateClientTokenResp{
		ClientToken: req.ClientToken,
	}, nil
}

func (d dexAPI) UpdateClientToken(ctx context.Context, req *api.UpdateClientTokenReq) (*api.UpdateClientTokenResp, error) {
	if req.Id == "" {
		return nil, errors.New("update ClientToken: no Id supplied")
	}

	err := d.s.UpdateClientToken(req.Id, func(old storage.ClientToken) (storage.ClientToken, error) {
		if req.ClientId != "" {
			old.ClientID = req.ClientId
		}
		if req.CreatedAt != 0 {
			old.CreatedAt = time.Unix(req.CreatedAt, 0)
		}
		if req.ExpiredAt != 0 {
			old.ExpiredAt = time.Unix(req.ExpiredAt, 0)
		}
		return old, nil
	})

	if err != nil {
		if err == storage.ErrNotFound {
			return &api.UpdateClientTokenResp{NotFound: true}, nil
		}
		d.logger.Errorf("api: failed to update the ClientToken: %v", err)
		return nil, fmt.Errorf("update ClientToken: %v", err)
	}
	return &api.UpdateClientTokenResp{}, nil
}

func (d dexAPI) ListClientToken(ctx context.Context, req *api.ListClientTokenReq) (*api.ListClientTokenResp, error) {
	tokenList, err := d.s.ListClientToken()
	if err != nil {
		d.logger.Errorf("api: failed to list ClientToken: %v", err)
		return nil, fmt.Errorf("list ClientToken: %v", err)
	}

	var tokens []*api.ClientToken
	for _, token := range tokenList {
		t := api.ClientToken{
			Id:        token.ID,
			ClientId:  token.ClientID,
			CreatedAt: token.CreatedAt.Unix(),
			ExpiredAt: token.ExpiredAt.Unix(),
		}
		tokens = append(tokens, &t)
	}

	return &api.ListClientTokenResp{
		ClientTokens: tokens,
	}, nil
}

func (d dexAPI) DeleteClientToken(ctx context.Context, req *api.DeleteClientTokenReq) (*api.DeleteClientTokenResp, error) {
	err := d.s.DeleteClientToken(req.Id)
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.DeleteClientTokenResp{NotFound: true}, nil
		}
		d.logger.Errorf("api: failed to delete ClientToken: %v", err)
		return nil, fmt.Errorf("delete ClientToken: %v", err)
	}
	return &api.DeleteClientTokenResp{}, nil
}

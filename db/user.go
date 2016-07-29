package db

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/repo"
	"github.com/coreos/dex/user"
)

const (
	// This table is named authd_user for historical reasons; namely, that the
	// original name of the project was authd, and there are existing tables out
	// there that we don't want to have to rename in production.
	userTableName                  = "authd_user"
	remoteIdentityMappingTableName = "remote_identity_mapping"
)

func init() {
	register(table{
		name:    userTableName,
		model:   userModel{},
		autoinc: false,
		pkey:    []string{"id"},
		unique:  []string{"email"},
	})

	register(table{
		name:    remoteIdentityMappingTableName,
		model:   remoteIdentityMappingModel{},
		autoinc: false,
		pkey:    []string{"connector_id", "remote_id"},
	})
}

func NewUserRepo(dbm *gorp.DbMap) user.UserRepo {
	return &userRepo{
		db: &db{dbm},
	}
}

func NewUserRepoFromUsers(dbm *gorp.DbMap, us []user.UserWithRemoteIdentities) (user.UserRepo, error) {
	repo := NewUserRepo(dbm).(*userRepo)
	for _, u := range us {
		um, err := newUserModel(&u.User)
		if err != nil {
			return nil, err
		}
		err = repo.executor(nil).Insert(um)
		for _, ri := range u.RemoteIdentities {
			err = repo.AddRemoteIdentity(nil, u.User.ID, ri)
			if err != nil {
				return nil, err
			}
		}
	}
	return repo, nil
}

type userRepo struct {
	*db
}

func (r *userRepo) Get(tx repo.Transaction, userID string) (user.User, error) {
	return r.get(tx, userID)
}

func (r *userRepo) Create(tx repo.Transaction, usr user.User) (err error) {
	if usr.ID == "" {
		return user.ErrorInvalidID
	}

	_, err = r.get(tx, usr.ID)
	if err == nil {
		return user.ErrorDuplicateID
	}
	if err != user.ErrorNotFound {
		return err
	}

	if !user.ValidEmail(usr.Email) {
		return user.ErrorInvalidEmail
	}

	// make sure there's no other user with the same Email
	_, err = r.getByEmail(tx, usr.Email)
	if err == nil {
		return user.ErrorDuplicateEmail
	}
	if err != user.ErrorNotFound {
		return err
	}

	err = r.insert(tx, usr)
	return err
}

func (r *userRepo) Disable(tx repo.Transaction, userID string, disable bool) error {
	if userID == "" {
		return user.ErrorInvalidID
	}

	qt := r.quote(userTableName)
	ex := r.executor(tx)
	result, err := ex.Exec(fmt.Sprintf("UPDATE %s SET disabled = $1 WHERE id = $2;", qt), disable, userID)
	if err != nil {
		return err
	}

	ct, err := result.RowsAffected()
	switch {
	case err != nil:
		return err
	case ct == 0:
		return user.ErrorNotFound
	}

	return nil
}

func (r *userRepo) GetByEmail(tx repo.Transaction, email string) (user.User, error) {
	return r.getByEmail(tx, email)
}

func (r *userRepo) Update(tx repo.Transaction, usr user.User) error {
	if usr.ID == "" {
		return user.ErrorInvalidID
	}

	if !user.ValidEmail(usr.Email) {
		return user.ErrorInvalidEmail
	}

	// make sure this user exists already
	_, err := r.get(tx, usr.ID)
	if err != nil {
		return err
	}

	// make sure there's no other user with the same Email
	otherUser, err := r.getByEmail(tx, usr.Email)
	if err != user.ErrorNotFound {
		if err != nil {
			return err
		}
		if otherUser.ID != usr.ID {
			return user.ErrorDuplicateEmail
		}
	}

	err = r.update(tx, usr)
	if err != nil {
		return err
	}

	return nil
}

func (r *userRepo) GetByRemoteIdentity(tx repo.Transaction, ri user.RemoteIdentity) (user.User, error) {
	userID, err := r.getUserIDForRemoteIdentity(tx, ri)
	if err != nil {
		return user.User{}, err
	}

	usr, err := r.get(tx, userID)
	if err != nil {
		return user.User{}, err
	}

	if err != nil {
		return user.User{}, err
	}

	return usr, nil
}

func (r *userRepo) AddRemoteIdentity(tx repo.Transaction, userID string, ri user.RemoteIdentity) error {
	_, err := r.get(tx, userID)
	if err != nil {
		return err
	}

	otherUserID, err := r.getUserIDForRemoteIdentity(tx, ri)
	if err != user.ErrorNotFound {
		if err == nil && otherUserID != "" {
			return user.ErrorDuplicateRemoteIdentity
		}
		return err
	}

	err = r.insertRemoteIdentity(tx, userID, ri)
	if err != nil {
		return err
	}

	return nil
}

func (r *userRepo) RemoveRemoteIdentity(tx repo.Transaction, userID string, rid user.RemoteIdentity) error {
	if userID == "" || rid.ID == "" || rid.ConnectorID == "" {
		return user.ErrorInvalidID
	}

	otherUserID, err := r.getUserIDForRemoteIdentity(tx, rid)
	if err != nil {
		return err
	}
	if otherUserID != userID {
		return user.ErrorNotFound
	}

	rim, err := newRemoteIdentityMappingModel(userID, rid)
	if err != nil {
		return err
	}

	ex := r.executor(tx)
	deleted, err := ex.Delete(rim)

	if err != nil {
		return err
	}

	if deleted == 0 {
		return user.ErrorNotFound
	}

	return nil
}

func (r *userRepo) GetRemoteIdentities(tx repo.Transaction, userID string) ([]user.RemoteIdentity, error) {
	ex := r.executor(tx)
	if userID == "" {
		return nil, user.ErrorInvalidID
	}

	qt := r.quote(remoteIdentityMappingTableName)
	rims, err := ex.Select(&remoteIdentityMappingModel{}, fmt.Sprintf("SELECT * FROM %s WHERE user_id = $1", qt), userID)

	if err != nil {
		if err != sql.ErrNoRows {
			return nil, err
		}
		return nil, nil
	}
	if len(rims) == 0 {
		return nil, nil
	}

	var ris []user.RemoteIdentity
	for _, m := range rims {
		rim, ok := m.(*remoteIdentityMappingModel)
		if !ok {
			log.Errorf("expected remoteIdentityMappingModel but found %v", reflect.TypeOf(m))
			return nil, errors.New("unrecognized model")
		}

		ris = append(ris, user.RemoteIdentity{
			ID:          rim.RemoteID,
			ConnectorID: rim.ConnectorID,
		})
	}

	return ris, nil
}

func (r *userRepo) GetAdminCount(tx repo.Transaction) (int, error) {
	qt := r.quote(userTableName)
	ex := r.executor(tx)
	i, err := ex.SelectInt(fmt.Sprintf("SELECT count(*) FROM %s WHERE admin=true;", qt))
	return int(i), err
}

func (r *userRepo) List(tx repo.Transaction, filter user.UserFilter, maxResults int, nextPageToken string) ([]user.User, string, error) {
	var offset int
	var err error
	if nextPageToken != "" {
		filter, maxResults, offset, err = user.DecodeNextPageToken(nextPageToken)
	}
	if err != nil {
		return nil, "", err
	}
	ex := r.executor(tx)

	qt := r.quote(userTableName)

	// Ask for one more than needed so we know if there's more results, and
	// hence, whether a nextPageToken is necessary.
	ums, err := ex.Select(&userModel{}, fmt.Sprintf("SELECT * FROM %s ORDER BY email LIMIT $1 OFFSET $2", qt), maxResults+1, offset)
	if err != nil {
		return nil, "", err
	}
	if len(ums) == 0 {
		return nil, "", user.ErrorNotFound
	}

	var more bool
	var numUsers int
	if len(ums) <= maxResults {
		numUsers = len(ums)
	} else {
		numUsers = maxResults
		more = true
	}

	users := make([]user.User, numUsers)
	for i := 0; i < numUsers; i++ {
		um, ok := ums[i].(*userModel)
		if !ok {
			log.Errorf("expected userModel but found %v", reflect.TypeOf(ums[i]))
			return nil, "", errors.New("unrecognized model")
		}
		usr, err := um.user()
		if err != nil {
			return nil, "", err
		}
		users[i] = usr
	}

	var tok string
	if more {
		tok, err = user.EncodeNextPageToken(filter, maxResults, offset+maxResults)
		if err != nil {
			return nil, "", err
		}
	}

	return users, tok, nil

}

func (r *userRepo) insert(tx repo.Transaction, usr user.User) error {
	ex := r.executor(tx)
	um, err := newUserModel(&usr)
	if err != nil {
		return err
	}
	return ex.Insert(um)
}

func (r *userRepo) update(tx repo.Transaction, usr user.User) error {
	ex := r.executor(tx)
	um, err := newUserModel(&usr)
	if err != nil {
		return err
	}
	_, err = ex.Update(um)
	return err
}

func (r *userRepo) get(tx repo.Transaction, userID string) (user.User, error) {
	ex := r.executor(tx)

	m, err := ex.Get(userModel{}, userID)
	if err != nil {
		return user.User{}, err
	}

	if m == nil {
		return user.User{}, user.ErrorNotFound
	}

	um, ok := m.(*userModel)
	if !ok {
		log.Errorf("expected userModel but found %v", reflect.TypeOf(m))
		return user.User{}, errors.New("unrecognized model")
	}

	return um.user()
}

func (r *userRepo) getUserIDForRemoteIdentity(tx repo.Transaction, ri user.RemoteIdentity) (string, error) {
	ex := r.executor(tx)

	m, err := ex.Get(remoteIdentityMappingModel{}, ri.ConnectorID, ri.ID)
	if err != nil {
		return "", err
	}

	if m == nil {
		return "", user.ErrorNotFound
	}

	rim, ok := m.(*remoteIdentityMappingModel)
	if !ok {
		log.Errorf("expected remoteIdentityMappingModel but found %v", reflect.TypeOf(m))
		return "", errors.New("unrecognized model")
	}

	return rim.UserID, nil
}

func (r *userRepo) getByEmail(tx repo.Transaction, email string) (user.User, error) {
	qt := r.quote(userTableName)
	ex := r.executor(tx)
	var um userModel
	err := ex.SelectOne(&um, fmt.Sprintf("select * from %s where email = $1", qt), strings.ToLower(email))

	if err != nil {
		if err == sql.ErrNoRows {
			return user.User{}, user.ErrorNotFound
		}
		return user.User{}, err
	}
	return um.user()
}

func (r *userRepo) insertRemoteIdentity(tx repo.Transaction, userID string, ri user.RemoteIdentity) error {
	ex := r.executor(tx)
	rim, err := newRemoteIdentityMappingModel(userID, ri)
	if err != nil {

		return err
	}
	err = ex.Insert(rim)
	return err
}

type userModel struct {
	ID            string `db:"id"`
	Email         string `db:"email"`
	EmailVerified bool   `db:"email_verified"`
	DisplayName   string `db:"display_name"`
	Disabled      bool   `db:"disabled"`
	Admin         bool   `db:"admin"`
	CreatedAt     int64  `db:"created_at"`
}

func (u *userModel) user() (user.User, error) {
	usr := user.User{
		ID:            u.ID,
		DisplayName:   u.DisplayName,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		Admin:         u.Admin,
		Disabled:      u.Disabled,
	}

	if u.CreatedAt != 0 {
		usr.CreatedAt = time.Unix(u.CreatedAt, 0).UTC()
	}

	return usr, nil
}

func newUserModel(u *user.User) (*userModel, error) {
	if u.ID == "" {
		return nil, fmt.Errorf("user is missing ID field")
	}
	if u.Email == "" {
		return nil, fmt.Errorf("user %s is missing email field", u.ID)
	}
	um := userModel{
		ID:            u.ID,
		DisplayName:   u.DisplayName,
		Email:         strings.ToLower(u.Email),
		EmailVerified: u.EmailVerified,
		Admin:         u.Admin,
		Disabled:      u.Disabled,
	}

	if !u.CreatedAt.IsZero() {
		um.CreatedAt = u.CreatedAt.Unix()
	}

	return &um, nil
}

func newRemoteIdentityMappingModel(userID string, ri user.RemoteIdentity) (*remoteIdentityMappingModel, error) {
	return &remoteIdentityMappingModel{
		ConnectorID: ri.ConnectorID,
		UserID:      userID,
		RemoteID:    ri.ID,
	}, nil
}

type remoteIdentityMappingModel struct {
	ConnectorID string `db:"connector_id"`
	UserID      string `db:"user_id"`
	RemoteID    string `db:"remote_id"`
}

package manager

import (
	"errors"
	"net/url"

	"github.com/jonboulle/clockwork"

	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/repo"
	"github.com/coreos/dex/user"
)

var (
	ErrorEmailAlreadyVerified   = errors.New("email already verified")
	ErrorPasswordAlreadyChanged = errors.New("password has already been changed")
)

// Manager performs user-related "business-logic" functions on user and related objects.
// This is in contrast to the Repos which perform little more than CRUD operations.
type UserManager struct {
	Clock clockwork.Clock

	userRepo        user.UserRepo
	pwRepo          user.PasswordInfoRepo
	connCfgRepo     connector.ConnectorConfigRepo
	begin           repo.TransactionFactory
	userIDGenerator user.UserIDGenerator
}

type ManagerOptions struct {
	// This is empty right now but will soon contain configuration information
	// such as passowrd length, name length, password expiration time and other
	// variable policies
}

func NewUserManager(userRepo user.UserRepo, pwRepo user.PasswordInfoRepo, connCfgRepo connector.ConnectorConfigRepo, txnFactory repo.TransactionFactory, options ManagerOptions) *UserManager {
	return &UserManager{
		Clock: clockwork.NewRealClock(),

		userRepo:        userRepo,
		pwRepo:          pwRepo,
		connCfgRepo:     connCfgRepo,
		begin:           txnFactory,
		userIDGenerator: user.DefaultUserIDGenerator,
	}
}

func (m *UserManager) Get(id string) (user.User, error) {
	return m.userRepo.Get(nil, id)
}

func (m *UserManager) List(filter user.UserFilter, maxResults int, nextPageToken string) ([]user.User, string, error) {
	return m.userRepo.List(nil, filter, maxResults, nextPageToken)
}

// CreateUser creates a new user with the given hashedPassword; the connID should be the ID of the local connector.
// The userID of the created user is returned as the first argument.
func (m *UserManager) CreateUser(usr user.User, hashedPassword user.Password, connID string) (string, error) {
	tx, err := m.begin()
	if err != nil {
		return "", err
	}

	insertedUser, err := m.insertNewUser(tx, usr.Email, usr.EmailVerified)
	if err != nil {
		rollback(tx)
		return "", err
	}

	usr.ID = insertedUser.ID
	usr.CreatedAt = insertedUser.CreatedAt
	err = m.userRepo.Update(tx, usr)
	if err != nil {
		rollback(tx)
		return "", err
	}

	rid := user.RemoteIdentity{
		ConnectorID: connID,
		ID:          usr.ID,
	}
	if err := m.addRemoteIdentity(tx, usr.ID, rid); err != nil {
		rollback(tx)
		return "", err
	}

	pwi := user.PasswordInfo{
		UserID:   usr.ID,
		Password: hashedPassword,
	}
	err = m.pwRepo.Create(tx, pwi)
	if err != nil {
		rollback(tx)
		return "", err
	}

	err = tx.Commit()
	if err != nil {
		rollback(tx)
		return "", err
	}
	return usr.ID, nil
}

func (m *UserManager) Disable(userID string, disabled bool) error {
	tx, err := m.begin()

	if err = m.userRepo.Disable(tx, userID, disabled); err != nil {
		rollback(tx)
		return err
	}

	if err = tx.Commit(); err != nil {
		rollback(tx)
		return err
	}

	return nil
}

// RegisterWithRemoteIdentity creates new user and attaches the given remote identity.
func (m *UserManager) RegisterWithRemoteIdentity(email string, emailVerified bool, rid user.RemoteIdentity) (string, error) {
	tx, err := m.begin()
	if err != nil {
		return "", err
	}

	if _, err = m.userRepo.GetByRemoteIdentity(tx, rid); err == nil {
		rollback(tx)
		return "", user.ErrorDuplicateRemoteIdentity
	}
	if err != user.ErrorNotFound {
		rollback(tx)
		return "", err
	}

	usr, err := m.insertNewUser(tx, email, emailVerified)
	if err != nil {
		rollback(tx)
		return "", err
	}

	if err := m.addRemoteIdentity(tx, usr.ID, rid); err != nil {
		rollback(tx)
		return "", err
	}

	err = tx.Commit()
	if err != nil {
		rollback(tx)
		return "", err
	}
	return usr.ID, nil
}

// RegisterWithPassword creates a new user with the given name and password.
// connID is the connector ID of the ConnectorLocal connector.
func (m *UserManager) RegisterWithPassword(email, plaintext, connID string) (string, error) {
	tx, err := m.begin()
	if err != nil {
		return "", err
	}

	if !user.ValidPassword(plaintext) {
		rollback(tx)
		return "", user.ErrorInvalidPassword
	}

	usr, err := m.insertNewUser(tx, email, false)
	if err != nil {
		rollback(tx)
		return "", err
	}

	rid := user.RemoteIdentity{
		ConnectorID: connID,
		ID:          usr.ID,
	}
	if err := m.addRemoteIdentity(tx, usr.ID, rid); err != nil {
		rollback(tx)
		return "", err
	}

	password, err := user.NewPasswordFromPlaintext(plaintext)
	if err != nil {
		rollback(tx)
		return "", err
	}
	pwi := user.PasswordInfo{
		UserID:   usr.ID,
		Password: password,
	}

	err = m.pwRepo.Create(tx, pwi)
	if err != nil {
		rollback(tx)
		return "", err
	}

	err = tx.Commit()
	if err != nil {
		rollback(tx)
		return "", err
	}
	return usr.ID, nil
}

type EmailVerifiable interface {
	UserID() string
	Email() string
	Callback() *url.URL
}

// VerifyEmail sets EmailVerified to true for the user for the given EmailVerification.
// The email in the EmailVerification must match the User's email in the
// repository, and it must not already be verified.
// This function expects that ParseAndVerifyEmailVerificationToken was used to
// create it, ensuring that the token was signed and that the JWT was not
// expired.
// The callback url (i.e. where to send the user after the verification) is returned.
func (m *UserManager) VerifyEmail(ev EmailVerifiable) (*url.URL, error) {
	tx, err := m.begin()
	if err != nil {
		return nil, err
	}

	usr, err := m.userRepo.GetByEmail(tx, ev.Email())
	if err != nil {
		rollback(tx)
		return nil, err
	}

	if usr.ID != ev.UserID() {
		rollback(tx)
		return nil, user.ErrorNotFound
	}

	if usr.EmailVerified {
		rollback(tx)
		return nil, ErrorEmailAlreadyVerified
	}

	usr.EmailVerified = true

	err = m.userRepo.Update(tx, usr)
	if err != nil {
		rollback(tx)
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		rollback(tx)
		return nil, err
	}
	return ev.Callback(), nil
}

type PasswordChangeable interface {
	UserID() string
	Password() user.Password
	Callback() *url.URL
}

func (m *UserManager) ChangePassword(pwr PasswordChangeable, plaintext string) (*url.URL, error) {
	tx, err := m.begin()
	if err != nil {
		return nil, err
	}

	if !user.ValidPassword(plaintext) {
		rollback(tx)
		return nil, user.ErrorInvalidPassword
	}

	pwi, err := m.pwRepo.Get(tx, pwr.UserID())
	if err != nil {
		rollback(tx)
		return nil, err
	}

	if string(pwi.Password) != string(pwr.Password()) {
		rollback(tx)
		return nil, ErrorPasswordAlreadyChanged
	}

	newPass, err := user.NewPasswordFromPlaintext(plaintext)
	if err != nil {
		rollback(tx)
		return nil, err
	}

	pwi.Password = newPass
	err = m.pwRepo.Update(tx, pwi)
	if err != nil {
		rollback(tx)
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		rollback(tx)
		return nil, err
	}
	return pwr.Callback(), nil
}

func (m *UserManager) insertNewUser(tx repo.Transaction, email string, emailVerified bool) (user.User, error) {
	if !user.ValidEmail(email) {
		return user.User{}, user.ErrorInvalidEmail
	}

	var err error
	if _, err = m.userRepo.GetByEmail(tx, email); err == nil {
		return user.User{}, user.ErrorDuplicateEmail
	}
	if err != user.ErrorNotFound {
		return user.User{}, err
	}

	userID, err := m.userIDGenerator()
	if err != nil {
		return user.User{}, err
	}

	usr := user.User{
		ID:            userID,
		Email:         email,
		EmailVerified: emailVerified,
		CreatedAt:     m.Clock.Now(),
	}

	err = m.userRepo.Create(tx, usr)
	if err != nil {
		return user.User{}, err
	}
	return usr, nil
}

func (m *UserManager) addRemoteIdentity(tx repo.Transaction, userID string, rid user.RemoteIdentity) error {
	if _, err := m.connCfgRepo.GetConnectorByID(tx, rid.ConnectorID); err != nil {
		return err
	}
	if err := m.userRepo.AddRemoteIdentity(tx, userID, rid); err != nil {
		return err
	}
	return nil
}

func rollback(tx repo.Transaction) {
	err := tx.Rollback()
	if err != nil {
		log.Errorf("unable to rollback: %v", err)
	}
}

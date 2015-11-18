package user

import (
	"errors"
	"net/url"

	"github.com/jonboulle/clockwork"

	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/repo"
)

var (
	ErrorEVEmailDoesntMatch   = errors.New("email in EV doesn't match user email")
	ErrorEmailAlreadyVerified = errors.New("email already verified")

	ErrorPasswordAlreadyChanged = errors.New("password has already been changed")
)

// Manager performs user-related "business-logic" functions on user and related objects.
// This is in contrast to the Repos which perform little more than CRUD operations.
type Manager struct {
	Clock clockwork.Clock

	userRepo        UserRepo
	pwRepo          PasswordInfoRepo
	begin           repo.TransactionFactory
	userIDGenerator UserIDGenerator
}

type ManagerOptions struct {
	// This is empty right now but will soon contain configuration information
	// such as passowrd length, name length, password expiration time and other
	// variable policies
}

func NewManager(userRepo UserRepo, pwRepo PasswordInfoRepo, txnFactory repo.TransactionFactory, options ManagerOptions) *Manager {
	return &Manager{
		Clock: clockwork.NewRealClock(),

		userRepo:        userRepo,
		pwRepo:          pwRepo,
		begin:           txnFactory,
		userIDGenerator: DefaultUserIDGenerator,
	}
}

func (m *Manager) Get(id string) (User, error) {
	return m.userRepo.Get(nil, id)
}

func (m *Manager) List(filter UserFilter, maxResults int, nextPageToken string) ([]User, string, error) {
	return m.userRepo.List(nil, filter, maxResults, nextPageToken)
}

// CreateUser creates a new user with the given hashedPassword; the connID should be the ID of the local connector.
// The userID of the created user is returned as the first argument.
func (m *Manager) CreateUser(user User, hashedPassword Password, connID string) (string, error) {
	tx, err := m.begin()
	if err != nil {
		return "", err
	}

	insertedUser, err := m.insertNewUser(tx, user.Email, user.EmailVerified)
	if err != nil {
		rollback(tx)
		return "", err
	}

	user.ID = insertedUser.ID
	user.CreatedAt = insertedUser.CreatedAt
	err = m.userRepo.Update(tx, user)
	if err != nil {
		rollback(tx)
		return "", err
	}

	rid := RemoteIdentity{
		ConnectorID: connID,
		ID:          user.ID,
	}
	if err := m.userRepo.AddRemoteIdentity(tx, user.ID, rid); err != nil {
		rollback(tx)
		return "", err
	}

	pwi := PasswordInfo{
		UserID:   user.ID,
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
	return user.ID, nil
}

func (m *Manager) Disable(userID string, disabled bool) error {
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
func (m *Manager) RegisterWithRemoteIdentity(email string, emailVerified bool, rid RemoteIdentity) (string, error) {
	tx, err := m.begin()
	if err != nil {
		return "", err
	}

	if _, err = m.userRepo.GetByRemoteIdentity(tx, rid); err == nil {
		rollback(tx)
		return "", ErrorDuplicateRemoteIdentity
	}
	if err != ErrorNotFound {
		rollback(tx)
		return "", err
	}

	user, err := m.insertNewUser(tx, email, emailVerified)
	if err != nil {
		rollback(tx)
		return "", err
	}

	if err := m.userRepo.AddRemoteIdentity(tx, user.ID, rid); err != nil {
		rollback(tx)
		return "", err
	}

	err = tx.Commit()
	if err != nil {
		rollback(tx)
		return "", err
	}
	return user.ID, nil
}

// RegisterWithPassword creates a new user with the given name and password.
// connID is the connector ID of the ConnectorLocal connector.
func (m *Manager) RegisterWithPassword(email, plaintext, connID string) (string, error) {
	tx, err := m.begin()
	if err != nil {
		return "", err
	}

	if !ValidPassword(plaintext) {
		rollback(tx)
		return "", ErrorInvalidPassword
	}

	user, err := m.insertNewUser(tx, email, false)
	if err != nil {
		rollback(tx)
		return "", err
	}

	rid := RemoteIdentity{
		ConnectorID: connID,
		ID:          user.ID,
	}
	if err := m.userRepo.AddRemoteIdentity(tx, user.ID, rid); err != nil {
		rollback(tx)
		return "", err
	}

	password, err := NewPasswordFromPlaintext(plaintext)
	if err != nil {
		rollback(tx)
		return "", err
	}
	pwi := PasswordInfo{
		UserID:   user.ID,
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
	return user.ID, nil
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
func (m *Manager) VerifyEmail(ev EmailVerifiable) (*url.URL, error) {
	tx, err := m.begin()
	if err != nil {
		return nil, err
	}

	user, err := m.userRepo.Get(tx, ev.UserID())
	if err != nil {
		rollback(tx)
		return nil, err
	}

	if user.Email != ev.Email() {
		rollback(tx)
		return nil, ErrorEVEmailDoesntMatch
	}

	if user.EmailVerified {
		rollback(tx)
		return nil, ErrorEmailAlreadyVerified
	}

	user.EmailVerified = true

	err = m.userRepo.Update(tx, user)
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
	Password() Password
	Callback() *url.URL
}

func (m *Manager) ChangePassword(pwr PasswordChangeable, plaintext string) (*url.URL, error) {
	tx, err := m.begin()
	if err != nil {
		return nil, err
	}

	if !ValidPassword(plaintext) {
		rollback(tx)
		return nil, ErrorInvalidPassword
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

	newPass, err := NewPasswordFromPlaintext(plaintext)
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

func (m *Manager) insertNewUser(tx repo.Transaction, email string, emailVerified bool) (User, error) {
	if !ValidEmail(email) {
		return User{}, ErrorInvalidEmail
	}

	var err error
	if _, err = m.userRepo.GetByEmail(tx, email); err == nil {
		return User{}, ErrorDuplicateEmail
	}
	if err != ErrorNotFound {
		return User{}, err
	}

	userID, err := m.userIDGenerator()
	if err != nil {
		return User{}, err
	}

	user := User{
		ID:            userID,
		Email:         email,
		EmailVerified: emailVerified,
		CreatedAt:     m.Clock.Now(),
	}

	err = m.userRepo.Create(tx, user)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func rollback(tx repo.Transaction) {
	err := tx.Rollback()
	if err != nil {
		log.Errorf("unable to rollback: %v", err)
	}
}

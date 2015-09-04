package db

import (
	"errors"
	"time"

	"github.com/coreos/dex/repo"
	"github.com/coreos/dex/user"
	"github.com/go-gorp/gorp"
)

const (
	passwordInfoTableName = "password_info"
)

func init() {
	register(table{
		name:    passwordInfoTableName,
		model:   passwordInfoModel{},
		autoinc: false,
		pkey:    []string{"user_id"},
	})

}

type passwordInfoModel struct {
	UserID          string `db:"user_id"`
	Password        string `db:"password"`
	PasswordExpires int64  `db:"password_expires"`
}

func NewPasswordInfoRepo(dbm *gorp.DbMap) user.PasswordInfoRepo {
	return &passwordInfoRepo{
		dbMap: dbm,
	}
}

type passwordInfoRepo struct {
	dbMap *gorp.DbMap
}

func (r *passwordInfoRepo) Get(tx repo.Transaction, userID string) (user.PasswordInfo, error) {
	return r.get(tx, userID)
}

func (r *passwordInfoRepo) Create(tx repo.Transaction, pw user.PasswordInfo) (err error) {
	if pw.UserID == "" {
		return user.ErrorInvalidID
	}

	_, err = r.get(tx, pw.UserID)
	if err == nil {
		return user.ErrorDuplicateID
	}
	if err != user.ErrorNotFound {
		return err
	}

	err = r.insert(tx, pw)
	if err != nil {
		return err
	}

	return nil
}

func (r *passwordInfoRepo) Update(tx repo.Transaction, pw user.PasswordInfo) error {
	if pw.UserID == "" {
		return user.ErrorInvalidID
	}

	if len(pw.Password) == 0 {
		return user.ErrorInvalidPassword
	}

	// make sure this user exists already
	_, err := r.get(tx, pw.UserID)
	if err != nil {
		return err
	}

	err = r.update(tx, pw)
	if err != nil {
		return err
	}

	return nil
}

func (r *passwordInfoRepo) executor(tx repo.Transaction) gorp.SqlExecutor {
	if tx == nil {
		return r.dbMap
	}

	gorpTx, ok := tx.(*gorp.Transaction)
	if !ok {
		panic("wrong kind of transaction passed to a DB repo")
	}
	return gorpTx
}

func (r *passwordInfoRepo) get(tx repo.Transaction, id string) (user.PasswordInfo, error) {
	ex := r.executor(tx)

	m, err := ex.Get(passwordInfoModel{}, id)
	if err != nil {
		return user.PasswordInfo{}, nil
	}

	if m == nil {
		return user.PasswordInfo{}, user.ErrorNotFound
	}

	pwm, ok := m.(*passwordInfoModel)
	if !ok {
		return user.PasswordInfo{}, errors.New("unrecognized model")
	}

	return pwm.passwordInfo()
}

func (r *passwordInfoRepo) insert(tx repo.Transaction, pw user.PasswordInfo) error {
	ex := r.executor(tx)
	pm, err := newPasswordInfoModel(&pw)
	if err != nil {
		return err
	}
	return ex.Insert(pm)
}

func (r *passwordInfoRepo) update(tx repo.Transaction, pw user.PasswordInfo) error {
	ex := r.executor(tx)
	pm, err := newPasswordInfoModel(&pw)
	if err != nil {
		return err
	}
	_, err = ex.Update(pm)
	return err
}

func (p *passwordInfoModel) passwordInfo() (user.PasswordInfo, error) {
	pw := user.PasswordInfo{
		UserID:   p.UserID,
		Password: user.Password(p.Password),
	}

	if p.PasswordExpires != 0 {
		pw.PasswordExpires = time.Unix(p.PasswordExpires, 0).UTC()
	}

	return pw, nil
}

func newPasswordInfoModel(p *user.PasswordInfo) (*passwordInfoModel, error) {
	pw := passwordInfoModel{
		UserID:   p.UserID,
		Password: string(p.Password),
	}

	if !p.PasswordExpires.IsZero() {
		pw.PasswordExpires = p.PasswordExpires.Unix()
	}

	return &pw, nil
}

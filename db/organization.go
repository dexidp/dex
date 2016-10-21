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
	organizationTableName = "organization"
)

func init() {
	register(table{
		name:    organizationTableName,
		model:   organizationModel{},
		autoinc: false,
		pkey:    []string{"organization_id"},
		unique:  []string{"name"},
	})
}

type organizationModel struct {
	OrganizationID string `db:"organization_id"`
	Name           string `db:"name"`
	OwnerID        string `db:"owner_id"`
	CreatedAt      int64  `db:"created_at"`
	Disabled       bool   `db:"disabled"`
}

func NewOrganizationRepo(dbm *gorp.DbMap) user.OrganizationRepo {
	return &organizationRepo{
		db: &db{dbm},
	}
}

func NewOrganizationRepoFromOrganizations(dbm *gorp.DbMap, infos []user.Organization) (user.OrganizationRepo, error) {
	repo := NewOrganizationRepo(dbm)
	for _, info := range infos {
		if err := repo.Create(nil, info); err != nil {
			return nil, err
		}
	}
	return repo, nil
}

type organizationRepo struct {
	*db
}

func (r *organizationRepo) Get(tx repo.Transaction, orgID string) (user.Organization, error) {
	return r.get(tx, orgID)
}

func (r *organizationRepo) Create(tx repo.Transaction, org user.Organization) (err error) {
	if org.OrganizationID == "" {
		return user.ErrorInvalidID
	}

	_, err = r.get(tx, org.OrganizationID)
	if err == nil {
		return user.ErrorDuplicateID
	}
	if err != user.ErrorNotFound {
		return err
	}

	// make sure there's no other organization with the same Name
	_, err = r.getByName(tx, org.Name)
	if err == nil {
		return user.ErrorDuplicateOrganizationName
	}
	if err != user.ErrorNotFound {
		return err
	}

	err = r.insert(tx, org)
	return err
}

func (r *organizationRepo) Disable(tx repo.Transaction, orgID string, disable bool) error {
	if orgID == "" {
		return user.ErrorInvalidID
	}

	qt := r.quote(organizationTableName)
	ex := r.executor(tx)
	result, err := ex.Exec(fmt.Sprintf("UPDATE %s SET disabled = $1 WHERE id = $2;", qt), disable, orgID)
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

func (r *organizationRepo) GetByName(tx repo.Transaction, name string) (user.Organization, error) {
	return r.getByName(tx, name)
}

func (r *organizationRepo) Update(tx repo.Transaction, org user.Organization) error {
	if org.OrganizationID == "" {
		return user.ErrorInvalidID
	}

	// make sure this organization exists already
	_, err := r.get(tx, org.OrganizationID)
	if err != nil {
		return err
	}

	// make sure there's no other organization with the same Name
	otherOrg, err := r.getByName(tx, org.Name)
	if err != user.ErrorNotFound {
		if err != nil {
			return err
		}
		if otherOrg.OrganizationID != org.OrganizationID {
			return user.ErrorDuplicateOrganizationName
		}
	}

	err = r.update(tx, org)
	if err != nil {
		return err
	}

	return nil
}

func (r *organizationRepo) insert(tx repo.Transaction, org user.Organization) error {
	ex := r.executor(tx)
	om, err := newOrganizationModel(&org)
	if err != nil {
		return err
	}
	return ex.Insert(om)
}

func (r *organizationRepo) update(tx repo.Transaction, org user.Organization) error {
	ex := r.executor(tx)
	om, err := newOrganizationModel(&org)
	if err != nil {
		return err
	}
	_, err = ex.Update(om)
	return err
}

func (r *organizationRepo) get(tx repo.Transaction, orgID string) (user.Organization, error) {
	ex := r.executor(tx)

	m, err := ex.Get(organizationModel{}, orgID)
	if err != nil {
		return user.Organization{}, err
	}

	if m == nil {
		return user.Organization{}, user.ErrorNotFound
	}

	om, ok := m.(*organizationModel)
	if !ok {
		log.Errorf("expected organizationModel but found %v", reflect.TypeOf(m))
		return user.Organization{}, errors.New("unrecognized model")
	}

	return om.organization()
}

func (r *organizationRepo) getByName(tx repo.Transaction, name string) (user.Organization, error) {
	qt := r.quote(organizationTableName)
	ex := r.executor(tx)
	var om organizationModel
	err := ex.SelectOne(&om, fmt.Sprintf("select * from %s where name = $1", qt), strings.ToLower(name))

	if err != nil {
		if err == sql.ErrNoRows {
			return user.Organization{}, user.ErrorNotFound
		}
		return user.Organization{}, err
	}
	return om.organization()
}

func (om *organizationModel) organization() (user.Organization, error) {
	org := user.Organization{
		OrganizationID: om.OrganizationID,
		Name:           om.Name,
		OwnerID:        om.OwnerID,
		Disabled:       om.Disabled,
	}

	if om.CreatedAt != 0 {
		org.CreatedAt = time.Unix(om.CreatedAt, 0).UTC()
	}

	return org, nil
}

func newOrganizationModel(org *user.Organization) (*organizationModel, error) {
	if org.OrganizationID == "" {
		return nil, fmt.Errorf("organization is missing OrganizationID field")
	}
	if org.Name == "" {
		return nil, fmt.Errorf("organization %s is missing name field", org.OrganizationID)
	}
	if org.OwnerID == "" {
		return nil, fmt.Errorf("organization %s is missing OwnerID field", org.OrganizationID)
	}
	om := organizationModel{
		OrganizationID: org.OrganizationID,
		Name:           strings.ToLower(org.Name),
		OwnerID:        org.OwnerID,
		Disabled:       org.Disabled,
	}

	if !org.CreatedAt.IsZero() {
		om.CreatedAt = org.CreatedAt.Unix()
	}

	return &om, nil
}

package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/coreos/dex/repo"
)

var (
	ErrorDuplicateOrganizationName = errors.New("duplicate organization name")
)

type Organization struct {
	OrganizationID string
	Name           string
	OwnerID        string
	Disabled       bool
	CreatedAt      time.Time
}

type OrganizationRepo interface {
	Get(tx repo.Transaction, id string) (Organization, error)
	GetByName(tx repo.Transaction, name string) (Organization, error)
	Update(repo.Transaction, Organization) error
	Create(repo.Transaction, Organization) error
	Disable(tx repo.Transaction, id string, disabled bool) error
}

func (org *Organization) UnmarshalJSON(data []byte) error {
	var dec struct {
		OrganizationID string `json:"organizationId"`
		Name           string `json:"name"`
		OwnerID        string `json:"ownerId"`
	}

	err := json.Unmarshal(data, &dec)
	if err != nil {
		return fmt.Errorf("invalid Organization entry: %v", err)
	}

	org.OrganizationID = dec.OrganizationID
	org.Name = dec.Name
	org.OwnerID = dec.OwnerID

	return nil
}

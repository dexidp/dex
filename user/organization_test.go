package user

import (
	"encoding/json"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestOrganizationMarshaling(t *testing.T) {
	tests := []Organization{
		{
			OrganizationID: "OrgID-1",
			OwnerID:        "UserID-1",
		},
		{
			OrganizationID: "OrgID-2",
			Name:           "OrgName-2",
			OwnerID:        "UserID-2",
		},
	}
	for i, tt := range tests {
		data, err := json.Marshal(tt)
		if err != nil {
			t.Errorf("case %d: failed to marshal organization: %v", i, err)
			continue
		}
		var p Organization
		if err := json.Unmarshal(data, &p); err != nil {
			t.Errorf("case %d: failed to unmarshal organization: %v", i, err)
			continue
		}
		if diff := pretty.Compare(tt, p); diff != "" {
			t.Errorf("case %d: organization did not survive JSON marshal round trip: %s", i, diff)
		}
	}
}

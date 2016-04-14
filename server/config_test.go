package server

import (
	"strings"
	"testing"

	"github.com/coreos/dex/user"
	"github.com/kylelemons/godebug/pretty"
)

func TestLoadUsers(t *testing.T) {
	tests := []struct {
		// The raw JSON file
		raw      string
		expUsers []user.UserWithRemoteIdentities
		// userid -> plaintext password
		expPasswds map[string]string

		wantErr bool
	}{
		{
			raw: `[
			    {
			        "id": "elroy-id",
			        "email": "elroy77@example.com",
			        "displayName": "Elroy Jonez",
			        "password": "bones",
			        "remoteIdentities": [
			            {
			                "connectorId": "local",
			                "id": "elroy-id"
			            }
			        ]
			    }
			]`,
			expUsers: []user.UserWithRemoteIdentities{
				{
					User: user.User{
						ID:          "elroy-id",
						Email:       "elroy77@example.com",
						DisplayName: "Elroy Jonez",
					},
					RemoteIdentities: []user.RemoteIdentity{
						{
							ConnectorID: "local",
							ID:          "elroy-id",
						},
					},
				},
			},
			expPasswds: map[string]string{
				"elroy-id": "bones",
			},
		},
		{
			// using old format.
			raw: `[
			    {
					"user": {
			        	"id": "elroy-id",
			        	"email": "elroy77@example.com",
			        	"displayName": "Elroy Jonez",
			        	"password": "bones"
					},
			        "remoteIdentities": [
			            {
			                "connectorId": "local",
			                "id": "elroy-id"
			            }
			        ]
			    }
			]`,
			wantErr: true,
		},
	}

	for i, tt := range tests {
		users, pwInfos, err := loadUsersFromReader(strings.NewReader(tt.raw))
		if err != nil {
			if !tt.wantErr {
				t.Errorf("case %d: failed to load user: %v", i, err)
			}
			continue
		}
		if tt.wantErr {
			t.Errorf("case %d: wanted parsing error, didn't get one", i)
			continue
		}

		if diff := pretty.Compare(tt.expUsers, users); diff != "" {
			t.Errorf("case: %d: wantUsers!=gotUsers: %s", i, diff)
		}

		// For each password info loaded, verify the password.
		for _, pwInfo := range pwInfos {
			expPW, ok := tt.expPasswds[pwInfo.UserID]
			if !ok {
				t.Errorf("no password entry for %s", pwInfo.UserID)
				continue
			}
			if _, err := pwInfo.Authenticate(expPW); err != nil {
				t.Errorf("case %d: user %s's password did not match", i, pwInfo.UserID)
			}
		}
	}
}

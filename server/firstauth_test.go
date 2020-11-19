package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"testing"
	"time"

	"github.com/dexidp/dex/connector"

	"github.com/dexidp/dex/storage"
)

func TestAuthenticate(t *testing.T) {
	fmt.Printf("hello")
	t0 := time.Now()
	now := func() time.Time { return t0 }
	type dataUser struct {
		ID            string
		Username      string
		PreferredName string
		Email         string
		Groups        []string
		AclTokens     []string
	}

	// Base "Control" test values
	baseClientTokens := []storage.ClientToken{
		{
			ID:        "clientTest0_id",
			ClientID:  "clientTest0",
			CreatedAt: now(),
			ExpiredAt: now().AddDate(0, 0, 10),
		},
		{
			ID:        "clientTest1_id",
			ClientID:  "clientTest1",
			CreatedAt: now(),
			ExpiredAt: now().AddDate(0, 0, 10),
		},
		{
			ID:        "clientTest2_id",
			ClientID:  "clientTest2",
			CreatedAt: now(),
			ExpiredAt: now().Add(-2 * time.Minute),
		},
		{
			ID:        "clientTest3_id",
			ClientID:  "clientTest3",
			CreatedAt: now(),
			ExpiredAt: now().Add(-2 * time.Minute),
		},
	}
	baseAclTokens := []storage.AclToken{
		{
			ID:           "aclTokenTest0_ID",
			Desc:         "aclTokenTest0",
			MaxUser:      "0",
			ClientTokens: []string{baseClientTokens[0].ID, baseClientTokens[1].ID},
		},
		{
			ID:           "aclTokenTest1_ID",
			Desc:         "aclTokenTest1",
			MaxUser:      "1",
			ClientTokens: []string{baseClientTokens[1].ID},
		},
		{
			ID:           "aclTokenTest2_ID",
			Desc:         "aclTokenTest2",
			MaxUser:      "1",
			ClientTokens: []string{baseClientTokens[2].ID},
		},
	}
	baseDefaultConnector := []DefaultConn{
		{
			Connector: "connector0",
			Clients:   []string{baseClientTokens[0].ClientID, baseClientTokens[1].ClientID},
		}, {
			Connector: "connector1",
			Clients:   []string{baseClientTokens[1].ClientID},
		},
	}

	// List of different tests
	tests := []struct {
		testName             string
		clientID             string
		connectorID          string
		userData             dataUser
		firstauthConf        FirstAuth
		scopes               []string
		createUser           bool
		expectedAuthenticate bool
		expectedError        error
	}{
		{
			testName:    "Succesfull Auto Mode",
			clientID:    "clientTest0",
			connectorID: "connector1",
			userData: dataUser{
				ID:            "JohnTestID",
				Username:      "John",
				PreferredName: "Jo",
				Email:         "john.Doe@iot.bzh",
				Groups:        []string{"groupA", "groupB"},
			},
			firstauthConf: FirstAuth{
				Enable:  true,
				Mode:    "auto",
				Default: baseDefaultConnector,
			},
			scopes:               []string{"openid", "email", "name", "groups"},
			createUser:           false,
			expectedAuthenticate: true,
			expectedError:        nil,
		},
		{
			testName:    "Not authenticate Auto Mode",
			clientID:    "clientTest0",
			connectorID: "connector2",
			userData: dataUser{
				ID:            "TotoTestID",
				Username:      "toto",
				PreferredName: "to",
				Email:         "toto@iot.bzh",
				Groups:        []string{"groupA", "groupB"},
			},
			firstauthConf: FirstAuth{
				Enable:  true,
				Mode:    "auto",
				Default: baseDefaultConnector,
			},
			scopes:               []string{"openid", "email", "name", "groups"},
			createUser:           false,
			expectedAuthenticate: false,
			expectedError:        nil,
		},
		{
			testName:    "Successfull Manual Mode",
			clientID:    "clientTest1",
			connectorID: "connector2",
			userData: dataUser{
				ID:            "TataTestID",
				Username:      "tata",
				PreferredName: "ta",
				Email:         "tata@iot.bzh",
				Groups:        []string{"groupA", "groupB"},
				AclTokens:     []string{baseAclTokens[1].ID},
			},
			firstauthConf: FirstAuth{
				Enable:  true,
				Mode:    "manual",
				Default: baseDefaultConnector,
			},
			scopes:               []string{"openid", "email", "name", "groups"},
			createUser:           true,
			expectedAuthenticate: true,
			expectedError:        nil,
		},
		{
			testName:    "Wrong Token Manual Mode",
			clientID:    "clientTest3",
			connectorID: "connector2",
			userData: dataUser{
				ID:            "TitiTestID",
				Username:      "titi",
				PreferredName: "ti",
				Email:         "titi@iot.bzh",
				Groups:        []string{"groupA", "groupB"},
				AclTokens:     []string{baseAclTokens[2].ID},
			},
			firstauthConf: FirstAuth{
				Enable:  true,
				Mode:    "manual",
				Default: baseDefaultConnector,
			},
			scopes:               []string{"openid", "email", "name", "groups"},
			createUser:           true,
			expectedAuthenticate: false,
			expectedError:        ErrNotAccess,
		},
		{
			testName:    "Expired Token Manual Mode",
			clientID:    "clientTest2",
			connectorID: "connector2",
			userData: dataUser{
				ID:            "TutuTestID",
				Username:      "tutu",
				PreferredName: "tu",
				Email:         "tutu@iot.bzh",
				Groups:        []string{"groupA", "groupB"},
				AclTokens:     []string{baseAclTokens[2].ID},
			},
			firstauthConf: FirstAuth{
				Enable:  true,
				Mode:    "manual",
				Default: baseDefaultConnector,
			},
			scopes:               []string{"openid", "email", "name", "groups"},
			createUser:           true,
			expectedAuthenticate: false,
			expectedError:        ErrExpiredToken,
		},
	}
	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Setup a dex server.
			httpServer, s := newTestServer(ctx, t, func(c *Config) {
				c.Issuer = c.Issuer + "/non-root-path"
				c.Now = now
				c.FirstAuth = tc.firstauthConf
			})
			defer httpServer.Close()

			u, err := url.Parse(s.issuerURL.String())
			if err != nil {
				t.Fatalf("Could not parse issuer URL %v", err)
			}
			u.Path = path.Join(u.Path, "firstauth/acltoken")
			q := u.Query()
			//q.Set("state", tc.values.state)
			u.RawQuery = q.Encode()
			req, _ := http.NewRequest("GET", u.String(), nil)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

			// Add data into database
			for _, clientToken := range baseClientTokens {
				if err := s.storage.CreateClientToken(clientToken); err != nil {
					t.Fatalf("failed to create client Token: %v", err)
				}
				defer s.storage.DeleteClientToken(clientToken.ID)
			}
			for _, aclToken := range baseAclTokens {
				if err := s.storage.CreateAclToken(aclToken); err != nil {
					t.Fatalf("failed to create Acl Token: %v", err)
				}
				defer s.storage.DeleteAclToken(aclToken.ID)
			}
			if tc.createUser {
				if err := s.storage.CreateUserIdp(storage.UserIdp{IdpID: tc.userData.ID + "_" + tc.connectorID, InternID: tc.userData.Username + "_internID"}); err != nil {
					t.Fatalf("failed to create user IDP: %v", err)
				}
				defer s.storage.DeleteUserIdp(tc.userData.ID + "_" + tc.connectorID)
				if err := s.storage.CreateUser(storage.User{InternID: tc.userData.Username + "_internID", Pseudo: tc.userData.PreferredName, Email: tc.userData.Email, Username: tc.userData.Username, AclTokens: tc.userData.AclTokens}); err != nil {
					t.Fatalf("failed to create user IDP: %v", err)
				}
				defer s.storage.DeleteUserIdp(tc.userData.Username + "_internID")
			}

			// create data for authenticate
			authReq := storage.AuthRequest{
				ID:                  "xxxxx-xxxxx",
				ClientID:            tc.clientID,
				ResponseTypes:       []string{"code"},
				Scopes:              tc.scopes,
				RedirectURI:         "http://test.firstauth",
				Nonce:               "yyy-yyy",
				State:               "zzz-zzz",
				ForceApprovalPrompt: false,
				Expiry:              now().Add(10 * time.Second),
				LoggedIn:            true,
				Claims: storage.Claims{
					UserID:            tc.userData.ID,
					Username:          tc.userData.Username,
					PreferredUsername: tc.userData.PreferredName,
					Email:             tc.userData.Email,
					EmailVerified:     true,
					Groups:            tc.userData.Groups,
				},
				ConnectorID:   tc.connectorID,
				ConnectorData: []byte{},
			}
			identity := connector.Identity{
				UserID:            tc.userData.ID,
				Username:          tc.userData.Username,
				PreferredUsername: tc.userData.PreferredName,
				Email:             tc.userData.Email,
				EmailVerified:     true,
				Groups:            tc.userData.Groups,
				ConnectorData:     []byte{},
			}

			// Test user authentication
			isAuth, err := Authenticate(req, s.storage, authReq, identity, tc.firstauthConf)
			if tc.expectedError == nil {
				if isAuth != tc.expectedAuthenticate {
					t.Errorf("User should be authenticate")
				}
			} else if tc.expectedError != err {
				t.Errorf("Unexpected Error Type.  Expected %v got %v", tc.expectedError, err)
			}
		})
	}

}

package repo

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"testing"
	"time"

	"github.com/coreos/go-oidc/oidc"
	"github.com/kylelemons/godebug/pretty"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/db"
	"github.com/coreos/dex/refresh"
	"github.com/coreos/dex/user"
)

var (
	testRefreshClientID  = "client1"
	testRefreshClientID2 = "client2"
	testRefreshClients   = []client.LoadableClient{
		{
			Client: client.Client{
				Credentials: oidc.ClientCredentials{
					ID:     testRefreshClientID,
					Secret: base64.URLEncoding.EncodeToString([]byte("secret-2")),
				},
				Metadata: oidc.ClientMetadata{
					RedirectURIs: []url.URL{
						url.URL{Scheme: "https", Host: "client1.example.com", Path: "/callback"},
					},
				},
			},
		},
		{
			Client: client.Client{
				Credentials: oidc.ClientCredentials{
					ID:     testRefreshClientID2,
					Secret: base64.URLEncoding.EncodeToString([]byte("secret-2")),
				},
				Metadata: oidc.ClientMetadata{
					RedirectURIs: []url.URL{
						url.URL{Scheme: "https", Host: "client2.example.com", Path: "/callback"},
					},
				},
			},
		},
	}

	testRefreshUserID = "user1"
	testRefreshUsers  = []user.UserWithRemoteIdentities{
		{
			User: user.User{
				ID:        testRefreshUserID,
				Email:     "Email-1@example.com",
				CreatedAt: time.Now().Truncate(time.Second),
			},
			RemoteIdentities: []user.RemoteIdentity{
				{
					ConnectorID: "IDPC-1",
					ID:          "RID-1",
				},
			},
		},
	}
)

func newRefreshRepo(t *testing.T, users []user.UserWithRemoteIdentities, clients []client.LoadableClient) refresh.RefreshTokenRepo {
	dbMap := connect(t)
	if _, err := db.NewUserRepoFromUsers(dbMap, users); err != nil {
		t.Fatalf("Unable to add users: %v", err)
	}

	if _, err := db.NewClientRepoFromClients(dbMap, clients); err != nil {
		t.Fatalf("Unable to add clients: %v", err)
	}

	return db.NewRefreshTokenRepo(dbMap)
}

func TestRefreshTokenRepoCreateVerify(t *testing.T) {
	tests := []struct {
		createScopes   []string
		verifyClientID string
		wantVerifyErr  bool
	}{
		{
			createScopes:   []string{"openid", "profile"},
			verifyClientID: testRefreshClientID,
		},
		{
			createScopes:   []string{},
			verifyClientID: testRefreshClientID,
		},
		{
			createScopes:   []string{"openid", "profile"},
			verifyClientID: "not-a-client",
			wantVerifyErr:  true,
		},
	}

	for i, tt := range tests {
		repo := newRefreshRepo(t, testRefreshUsers, testRefreshClients)
		tok, err := repo.Create(testRefreshUserID, testRefreshClientID, tt.createScopes)
		if err != nil {
			t.Fatalf("case %d: failed to create refresh token: %v", i, err)
		}

		tokUserID, gotScopes, err := repo.Verify(tt.verifyClientID, tok)
		if tt.wantVerifyErr {
			if err == nil {
				t.Errorf("case %d: want non-nil error.", i)
			}
			continue
		}

		if diff := pretty.Compare(tt.createScopes, gotScopes); diff != "" {
			t.Errorf("case %d: Compare(want, got): %v", i, diff)
		}

		if err != nil {
			t.Errorf("case %d: Could not verify token: %v", i, err)
		} else if tokUserID != testRefreshUserID {
			t.Errorf("case %d: Verified token returned wrong user id, want=%s, got=%s", i,
				testRefreshUserID, tokUserID)
		}
	}
}

// buildRefreshToken combines the token ID and token payload to create a new token.
// used in the tests to created a refresh token.
func buildRefreshToken(tokenID int64, tokenPayload []byte) string {
	return fmt.Sprintf("%d%s%s", tokenID, refresh.TokenDelimer, base64.URLEncoding.EncodeToString(tokenPayload))
}

func TestRefreshRepoVerifyInvalidTokens(t *testing.T) {
	r := db.NewRefreshTokenRepo(connect(t))

	token, err := r.Create("user-foo", "client-foo", oidc.DefaultScope)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	badTokenPayload, err := refresh.DefaultRefreshTokenGenerator()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	tokenWithBadID := "404" + token[1:]
	tokenWithBadPayload := buildRefreshToken(1, badTokenPayload)

	tests := []struct {
		token    string
		creds    oidc.ClientCredentials
		err      error
		expected string
	}{
		{
			"invalid-token-format",
			oidc.ClientCredentials{ID: "client-foo", Secret: "secret-foo"},
			refresh.ErrorInvalidToken,
			"",
		},
		{
			"b/invalid-base64-encoded-format",
			oidc.ClientCredentials{ID: "client-foo", Secret: "secret-foo"},
			refresh.ErrorInvalidToken,
			"",
		},
		{
			"1/invalid-base64-encoded-format",
			oidc.ClientCredentials{ID: "client-foo", Secret: "secret-foo"},
			refresh.ErrorInvalidToken,
			"",
		},
		{
			token + "corrupted-token-payload",
			oidc.ClientCredentials{ID: "client-foo", Secret: "secret-foo"},
			refresh.ErrorInvalidToken,
			"",
		},
		{
			// The token's ID content is invalid.
			tokenWithBadID,
			oidc.ClientCredentials{ID: "client-foo", Secret: "secret-foo"},
			refresh.ErrorInvalidToken,
			"",
		},
		{
			// The token's payload content is invalid.
			tokenWithBadPayload,
			oidc.ClientCredentials{ID: "client-foo", Secret: "secret-foo"},
			refresh.ErrorInvalidToken,
			"",
		},
		{
			token,
			oidc.ClientCredentials{ID: "invalid-client", Secret: "secret-foo"},
			refresh.ErrorInvalidClientID,
			"",
		},
		{
			token,
			oidc.ClientCredentials{ID: "client-foo", Secret: "secret-foo"},
			nil,
			"user-foo",
		},
	}

	for i, tt := range tests {
		result, _, err := r.Verify(tt.creds.ID, tt.token)
		if err != tt.err {
			t.Errorf("Case #%d: expected: %v, got: %v", i, tt.err, err)
		}
		if result != tt.expected {
			t.Errorf("Case #%d: expected: %v, got: %v", i, tt.expected, result)
		}
	}
}

func TestRefreshTokenRepoClientsWithRefreshTokens(t *testing.T) {
	tests := []struct {
		clientIDs []string
	}{
		{clientIDs: []string{"client1", "client2"}},
		{clientIDs: []string{"client1"}},
		{clientIDs: []string{}},
	}

	for i, tt := range tests {
		repo := newRefreshRepo(t, testRefreshUsers, testRefreshClients)

		for _, clientID := range tt.clientIDs {
			_, err := repo.Create(testRefreshUserID, clientID, []string{"openid"})
			if err != nil {
				t.Fatalf("case %d: client_id: %s couldn't create refresh token: %v", i, clientID, err)
			}
		}

		clients, err := repo.ClientsWithRefreshTokens(testRefreshUserID)
		if err != nil {
			t.Fatalf("case %d: unexpected error fetching clients %q", i, err)
		}
		var clientIDs []string
		for _, client := range clients {
			clientIDs = append(clientIDs, client.Credentials.ID)
		}
		sort.Strings(clientIDs)

		if diff := pretty.Compare(clientIDs, tt.clientIDs); diff != "" {
			t.Errorf("case %d: Compare(want, got): %v", i, diff)
		}
	}
}

func TestRefreshTokenRepoRevokeForClient(t *testing.T) {
	tests := []struct {
		createIDs []string
		revokeID  string
	}{
		{
			createIDs: []string{"client1", "client2"},
			revokeID:  "client1",
		},
		{
			createIDs: []string{"client2"},
			revokeID:  "client1",
		},
		{
			createIDs: []string{"client1"},
			revokeID:  "client1",
		},
		{
			createIDs: []string{},
			revokeID:  "oops",
		},
	}

	for i, tt := range tests {
		repo := newRefreshRepo(t, testRefreshUsers, testRefreshClients)

		for _, clientID := range tt.createIDs {
			_, err := repo.Create(testRefreshUserID, clientID, []string{"openid"})
			if err != nil {
				t.Fatalf("case %d: client_id: %s couldn't create refresh token: %v", i, clientID, err)
			}

			if err := repo.RevokeTokensForClient(testRefreshUserID, tt.revokeID); err != nil {
				t.Fatalf("case %d: couldn't revoke refresh token(s): %v", i, err)
			}
		}

		var wantIDs []string
		for _, id := range tt.createIDs {
			if id != tt.revokeID {
				wantIDs = append(wantIDs, id)
			}
		}

		clients, err := repo.ClientsWithRefreshTokens(testRefreshUserID)
		if err != nil {
			t.Fatalf("case %d: unexpected error fetching clients %q", i, err)
		}

		var gotIDs []string
		for _, client := range clients {
			gotIDs = append(gotIDs, client.Credentials.ID)
		}
		sort.Strings(gotIDs)

		if diff := pretty.Compare(wantIDs, gotIDs); diff != "" {
			t.Errorf("case %d: Compare(wantIDs, gotIDs): %v", i, diff)
		}
	}
}

func TestRefreshRepoRevoke(t *testing.T) {
	r := db.NewRefreshTokenRepo(connect(t))

	token, err := r.Create("user-foo", "client-foo", oidc.DefaultScope)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	badTokenPayload, err := refresh.DefaultRefreshTokenGenerator()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	tokenWithBadID := "404" + token[1:]
	tokenWithBadPayload := buildRefreshToken(1, badTokenPayload)

	tests := []struct {
		token  string
		userID string
		err    error
	}{
		{
			"invalid-token-format",
			"user-foo",
			refresh.ErrorInvalidToken,
		},
		{
			"1/invalid-base64-encoded-format",
			"user-foo",
			refresh.ErrorInvalidToken,
		},
		{
			token + "corrupted-token-payload",
			"user-foo",
			refresh.ErrorInvalidToken,
		},
		{
			// The token's ID is invalid.
			tokenWithBadID,
			"user-foo",
			refresh.ErrorInvalidToken,
		},
		{
			// The token's payload is invalid.
			tokenWithBadPayload,
			"user-foo",
			refresh.ErrorInvalidToken,
		},
		{
			token,
			"invalid-user",
			refresh.ErrorInvalidUserID,
		},
		{
			token,
			"user-foo",
			nil,
		},
	}

	for i, tt := range tests {
		if err := r.Revoke(tt.userID, tt.token); err != tt.err {
			t.Errorf("Case #%d: expected: %v, got: %v", i, tt.err, err)
		}
	}
}

package user

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"

	"github.com/coreos/go-oidc/jose"
)

func TestNewUsersFromReader(t *testing.T) {
	tests := []struct {
		json string
		want []UserWithRemoteIdentities
	}{
		{
			json: `[{"user":{"id":"12345", "displayName": "Elroy Canis", "email":"elroy23@example.com"}, "remoteIdentities":[{"connectorID":"google", "id":"elroy@example.com"}] }]`,
			want: []UserWithRemoteIdentities{
				{
					User: User{
						ID:          "12345",
						DisplayName: "Elroy Canis",
						Email:       "elroy23@example.com",
					},
					RemoteIdentities: []RemoteIdentity{
						{
							ConnectorID: "google",
							ID:          "elroy@example.com",
						},
					},
				},
			},
		},
	}

	for i, tt := range tests {
		r := strings.NewReader(tt.json)
		us, err := newUsersFromReader(r)
		if err != nil {
			t.Errorf("case %d: want nil err: %v", i, err)
			continue
		}
		if diff := pretty.Compare(tt.want, us); diff != "" {
			t.Errorf("case %d: Compare(want, got): %v", i, diff)
		}
	}
}

func TestAddToClaims(t *testing.T) {
	tests := []struct {
		user         User
		wantedClaims jose.Claims
	}{
		{
			user: User{
				DisplayName: "Test User Name",
			},
			wantedClaims: jose.Claims{
				"name": "Test User Name",
			},
		},
		{
			user: User{
				DisplayName: "Test User Name",
				Email:       "unverified@example.com",
			},
			wantedClaims: jose.Claims{
				"name":  "Test User Name",
				"email": "unverified@example.com",
			},
		},
		{
			user: User{
				DisplayName:   "Test User Name",
				Email:         "verified@example.com",
				EmailVerified: true,
			},
			wantedClaims: jose.Claims{
				"name":           "Test User Name",
				"email":          "verified@example.com",
				"email_verified": true,
			},
		},
	}

	for i, tt := range tests {
		claims := jose.Claims{}
		tt.user.AddToClaims(claims)
		if !reflect.DeepEqual(claims, tt.wantedClaims) {
			t.Errorf("case %d: want=%#v, got=%#v", i, tt.wantedClaims, claims)
		}
	}
}

func TestValidEmail(t *testing.T) {
	tests := []struct {
		email string
		want  bool
	}{
		{"example@example.com", true},
		{"r@r.com", true},
		{"Barry Gibbs <bg@example.com>", false},
		{"", false},
		{"invalidemail", false},
		{"example@example.com example@example.com", false},
		{"example@example.com Hello, 世界", false},
	}

	for i, tt := range tests {
		if ValidEmail(tt.email) != tt.want {
			t.Errorf("case %d: want=%v, got=%v", i, tt.want, !tt.want)
		}
	}
}

func TestEncodeDecodeNextPageToken(t *testing.T) {
	tests := []nextPageToken{
		{},
		{MaxResults: 100},
		{Offset: 200},
		{MaxResults: 20, Offset: 30},
	}

	for i, tt := range tests {
		enc, err := EncodeNextPageToken(tt.Filter, tt.MaxResults, tt.Offset)
		if err != nil {
			t.Errorf("case %d: unexpected err encoding: %q", i, err)
		}

		dec := nextPageToken{}
		dec.Filter, dec.MaxResults, dec.Offset, err = DecodeNextPageToken(enc)
		if err != nil {
			t.Errorf("case %d: unexpected err decoding: %q", i, err)
		}

		if diff := pretty.Compare(tt, dec); diff != "" {
			t.Errorf("case %d: Compare(want, got): %v", i, diff)
		}
	}
}

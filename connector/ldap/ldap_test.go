package ldap

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/sirupsen/logrus"

	"github.com/dexidp/dex/connector"
)

// connectionMethod indicates how the test should connect to the LDAP server.
type connectionMethod int32

const (
	connectStartTLS connectionMethod = iota
	connectLDAPS
	connectLDAP
	connectInsecureSkipVerify
)

// subtest is a login test against a given schema.
type subtest struct {
	// Name of the sub-test.
	name string

	// Password credentials, and if the connector should request
	// groups as well.
	username string
	password string
	groups   bool

	// Expected result of the login.
	wantErr   bool
	wantBadPW bool
	want      connector.Identity
}

func TestQuery(t *testing.T) {
	c := &Config{}
	c.UserSearch.BaseDN = "ou=People,ou=TestQuery,dc=example,dc=org"
	c.UserSearch.NameAttr = "cn"
	c.UserSearch.EmailAttr = "mail"
	c.UserSearch.IDAttr = "DN"
	c.UserSearch.Username = "cn"

	tests := []subtest{
		{
			name:     "validpassword",
			username: "jane",
			password: "foo",
			want: connector.Identity{
				UserID:        "cn=jane,ou=People,ou=TestQuery,dc=example,dc=org",
				Username:      "jane",
				Email:         "janedoe@example.com",
				EmailVerified: true,
			},
		},
		{
			name:     "validpassword2",
			username: "john",
			password: "bar",
			want: connector.Identity{
				UserID:        "cn=john,ou=People,ou=TestQuery,dc=example,dc=org",
				Username:      "john",
				Email:         "johndoe@example.com",
				EmailVerified: true,
			},
		},
		{
			name:      "invalidpassword",
			username:  "jane",
			password:  "badpassword",
			wantBadPW: true,
		},
		{
			name:      "invaliduser",
			username:  "idontexist",
			password:  "foo",
			wantBadPW: true, // Want invalid password, not a query error.
		},
		{
			name:      "invalid wildcard username",
			username:  "a*", // wildcard query is not allowed
			password:  "foo",
			wantBadPW: true, // Want invalid password, not a query error.
		},
		{
			name:      "invalid wildcard password",
			username:  "john",
			password:  "*",  // wildcard password is not allowed
			wantBadPW: true, // Want invalid password, not a query error.
		},
	}

	runTests(t, connectLDAP, c, tests)
}

func TestQueryWithEmailSuffix(t *testing.T) {
	c := &Config{}
	c.UserSearch.BaseDN = "ou=People,ou=TestQueryWithEmailSuffix,dc=example,dc=org"
	c.UserSearch.NameAttr = "cn"
	c.UserSearch.EmailSuffix = "test.example.com"
	c.UserSearch.IDAttr = "DN"
	c.UserSearch.Username = "cn"

	tests := []subtest{
		{
			name:     "ignoremailattr",
			username: "jane",
			password: "foo",
			want: connector.Identity{
				UserID:        "cn=jane,ou=People,ou=TestQueryWithEmailSuffix,dc=example,dc=org",
				Username:      "jane",
				Email:         "jane@test.example.com",
				EmailVerified: true,
			},
		},
		{
			name:     "nomailattr",
			username: "john",
			password: "bar",
			want: connector.Identity{
				UserID:        "cn=john,ou=People,ou=TestQueryWithEmailSuffix,dc=example,dc=org",
				Username:      "john",
				Email:         "john@test.example.com",
				EmailVerified: true,
			},
		},
	}

	runTests(t, connectLDAP, c, tests)
}

func TestUserFilter(t *testing.T) {
	c := &Config{}
	c.UserSearch.BaseDN = "ou=TestUserFilter,dc=example,dc=org"
	c.UserSearch.NameAttr = "cn"
	c.UserSearch.EmailAttr = "mail"
	c.UserSearch.IDAttr = "DN"
	c.UserSearch.Username = "cn"
	c.UserSearch.Filter = "(ou:dn:=Seattle)"

	tests := []subtest{
		{
			name:     "validpassword",
			username: "jane",
			password: "foo",
			want: connector.Identity{
				UserID:        "cn=jane,ou=People,ou=Seattle,ou=TestUserFilter,dc=example,dc=org",
				Username:      "jane",
				Email:         "janedoe@example.com",
				EmailVerified: true,
			},
		},
		{
			name:     "validpassword2",
			username: "john",
			password: "bar",
			want: connector.Identity{
				UserID:        "cn=john,ou=People,ou=Seattle,ou=TestUserFilter,dc=example,dc=org",
				Username:      "john",
				Email:         "johndoe@example.com",
				EmailVerified: true,
			},
		},
		{
			name:      "invalidpassword",
			username:  "jane",
			password:  "badpassword",
			wantBadPW: true,
		},
		{
			name:      "invaliduser",
			username:  "idontexist",
			password:  "foo",
			wantBadPW: true, // Want invalid password, not a query error.
		},
	}

	runTests(t, connectLDAP, c, tests)
}

func TestGroupQuery(t *testing.T) {
	c := &Config{}
	c.UserSearch.BaseDN = "ou=People,ou=TestGroupQuery,dc=example,dc=org"
	c.UserSearch.NameAttr = "cn"
	c.UserSearch.EmailAttr = "mail"
	c.UserSearch.IDAttr = "DN"
	c.UserSearch.Username = "cn"
	c.GroupSearch.BaseDN = "ou=Groups,ou=TestGroupQuery,dc=example,dc=org"
	c.GroupSearch.UserMatchers = []UserMatcher{
		{
			UserAttr:  "DN",
			GroupAttr: "member",
		},
	}
	c.GroupSearch.NameAttr = "cn"

	tests := []subtest{
		{
			name:     "validpassword",
			username: "jane",
			password: "foo",
			groups:   true,
			want: connector.Identity{
				UserID:        "cn=jane,ou=People,ou=TestGroupQuery,dc=example,dc=org",
				Username:      "jane",
				Email:         "janedoe@example.com",
				EmailVerified: true,
				Groups:        []string{"admins", "developers"},
			},
		},
		{
			name:     "validpassword2",
			username: "john",
			password: "bar",
			groups:   true,
			want: connector.Identity{
				UserID:        "cn=john,ou=People,ou=TestGroupQuery,dc=example,dc=org",
				Username:      "john",
				Email:         "johndoe@example.com",
				EmailVerified: true,
				Groups:        []string{"admins"},
			},
		},
	}

	runTests(t, connectLDAP, c, tests)
}

func TestGroupsOnUserEntity(t *testing.T) {
	c := &Config{}
	c.UserSearch.BaseDN = "ou=People,ou=TestGroupsOnUserEntity,dc=example,dc=org"
	c.UserSearch.NameAttr = "cn"
	c.UserSearch.EmailAttr = "mail"
	c.UserSearch.IDAttr = "DN"
	c.UserSearch.Username = "cn"
	c.GroupSearch.BaseDN = "ou=Groups,ou=TestGroupsOnUserEntity,dc=example,dc=org"
	c.GroupSearch.UserMatchers = []UserMatcher{
		{
			UserAttr:  "departmentNumber",
			GroupAttr: "gidNumber",
		},
	}
	c.GroupSearch.NameAttr = "cn"
	tests := []subtest{
		{
			name:     "validpassword",
			username: "jane",
			password: "foo",
			groups:   true,
			want: connector.Identity{
				UserID:        "cn=jane,ou=People,ou=TestGroupsOnUserEntity,dc=example,dc=org",
				Username:      "jane",
				Email:         "janedoe@example.com",
				EmailVerified: true,
				Groups:        []string{"admins", "developers"},
			},
		},
		{
			name:     "validpassword2",
			username: "john",
			password: "bar",
			groups:   true,
			want: connector.Identity{
				UserID:        "cn=john,ou=People,ou=TestGroupsOnUserEntity,dc=example,dc=org",
				Username:      "john",
				Email:         "johndoe@example.com",
				EmailVerified: true,
				Groups:        []string{"admins", "designers"},
			},
		},
	}
	runTests(t, connectLDAP, c, tests)
}

func TestGroupFilter(t *testing.T) {
	c := &Config{}
	c.UserSearch.BaseDN = "ou=People,ou=TestGroupFilter,dc=example,dc=org"
	c.UserSearch.NameAttr = "cn"
	c.UserSearch.EmailAttr = "mail"
	c.UserSearch.IDAttr = "DN"
	c.UserSearch.Username = "cn"
	c.GroupSearch.BaseDN = "ou=TestGroupFilter,dc=example,dc=org"
	c.GroupSearch.UserMatchers = []UserMatcher{
		{
			UserAttr:  "dn",
			GroupAttr: "member",
		},
	}
	c.GroupSearch.NameAttr = "cn"
	c.GroupSearch.Filter = "(ou:dn:=Seattle)" // ignore other groups

	tests := []subtest{
		{
			name:     "validpassword",
			username: "jane",
			password: "foo",
			groups:   true,
			want: connector.Identity{
				UserID:        "cn=jane,ou=People,ou=TestGroupFilter,dc=example,dc=org",
				Username:      "jane",
				Email:         "janedoe@example.com",
				EmailVerified: true,
				Groups:        []string{"admins", "developers"},
			},
		},
		{
			name:     "validpassword2",
			username: "john",
			password: "bar",
			groups:   true,
			want: connector.Identity{
				UserID:        "cn=john,ou=People,ou=TestGroupFilter,dc=example,dc=org",
				Username:      "john",
				Email:         "johndoe@example.com",
				EmailVerified: true,
				Groups:        []string{"admins"},
			},
		},
	}

	runTests(t, connectLDAP, c, tests)
}

func TestGroupToUserMatchers(t *testing.T) {
	c := &Config{}
	c.UserSearch.BaseDN = "ou=People,ou=TestGroupToUserMatchers,dc=example,dc=org"
	c.UserSearch.NameAttr = "cn"
	c.UserSearch.EmailAttr = "mail"
	c.UserSearch.IDAttr = "DN"
	c.UserSearch.Username = "cn"
	c.GroupSearch.BaseDN = "ou=TestGroupToUserMatchers,dc=example,dc=org"
	c.GroupSearch.UserMatchers = []UserMatcher{
		{
			UserAttr:  "DN",
			GroupAttr: "member",
		},
		{
			UserAttr:  "uid",
			GroupAttr: "memberUid",
		},
	}
	c.GroupSearch.NameAttr = "cn"
	c.GroupSearch.Filter = "(|(objectClass=posixGroup)(objectClass=groupOfNames))" // search all group types

	tests := []subtest{
		{
			name:     "validpassword",
			username: "jane",
			password: "foo",
			groups:   true,
			want: connector.Identity{
				UserID:        "cn=jane,ou=People,ou=TestGroupToUserMatchers,dc=example,dc=org",
				Username:      "jane",
				Email:         "janedoe@example.com",
				EmailVerified: true,
				Groups:        []string{"admins", "developers", "frontend"},
			},
		},
		{
			name:     "validpassword2",
			username: "john",
			password: "bar",
			groups:   true,
			want: connector.Identity{
				UserID:        "cn=john,ou=People,ou=TestGroupToUserMatchers,dc=example,dc=org",
				Username:      "john",
				Email:         "johndoe@example.com",
				EmailVerified: true,
				Groups:        []string{"admins", "qa", "logger"},
			},
		},
	}

	runTests(t, connectLDAP, c, tests)
}

// Test deprecated group to user matching implementation
// which was left for backward compatibility.
// See "Config.GroupSearch.UserMatchers" comments for the details
func TestDeprecatedGroupToUserMatcher(t *testing.T) {
	c := &Config{}
	c.UserSearch.BaseDN = "ou=People,ou=TestDeprecatedGroupToUserMatcher,dc=example,dc=org"
	c.UserSearch.NameAttr = "cn"
	c.UserSearch.EmailAttr = "mail"
	c.UserSearch.IDAttr = "DN"
	c.UserSearch.Username = "cn"
	c.GroupSearch.BaseDN = "ou=TestDeprecatedGroupToUserMatcher,dc=example,dc=org"
	c.GroupSearch.UserAttr = "DN"
	c.GroupSearch.GroupAttr = "member"
	c.GroupSearch.NameAttr = "cn"
	c.GroupSearch.Filter = "(ou:dn:=Seattle)" // ignore other groups

	tests := []subtest{
		{
			name:     "validpassword",
			username: "jane",
			password: "foo",
			groups:   true,
			want: connector.Identity{
				UserID:        "cn=jane,ou=People,ou=TestDeprecatedGroupToUserMatcher,dc=example,dc=org",
				Username:      "jane",
				Email:         "janedoe@example.com",
				EmailVerified: true,
				Groups:        []string{"admins", "developers"},
			},
		},
		{
			name:     "validpassword2",
			username: "john",
			password: "bar",
			groups:   true,
			want: connector.Identity{
				UserID:        "cn=john,ou=People,ou=TestDeprecatedGroupToUserMatcher,dc=example,dc=org",
				Username:      "john",
				Email:         "johndoe@example.com",
				EmailVerified: true,
				Groups:        []string{"admins"},
			},
		},
	}

	runTests(t, connectLDAP, c, tests)
}

func TestStartTLS(t *testing.T) {
	c := &Config{}
	c.UserSearch.BaseDN = "ou=People,ou=TestStartTLS,dc=example,dc=org"
	c.UserSearch.NameAttr = "cn"
	c.UserSearch.EmailAttr = "mail"
	c.UserSearch.IDAttr = "DN"
	c.UserSearch.Username = "cn"

	tests := []subtest{
		{
			name:     "validpassword",
			username: "jane",
			password: "foo",
			want: connector.Identity{
				UserID:        "cn=jane,ou=People,ou=TestStartTLS,dc=example,dc=org",
				Username:      "jane",
				Email:         "janedoe@example.com",
				EmailVerified: true,
			},
		},
	}
	runTests(t, connectStartTLS, c, tests)
}

func TestInsecureSkipVerify(t *testing.T) {
	c := &Config{}
	c.UserSearch.BaseDN = "ou=People,ou=TestInsecureSkipVerify,dc=example,dc=org"
	c.UserSearch.NameAttr = "cn"
	c.UserSearch.EmailAttr = "mail"
	c.UserSearch.IDAttr = "DN"
	c.UserSearch.Username = "cn"

	tests := []subtest{
		{
			name:     "validpassword",
			username: "jane",
			password: "foo",
			want: connector.Identity{
				UserID:        "cn=jane,ou=People,ou=TestInsecureSkipVerify,dc=example,dc=org",
				Username:      "jane",
				Email:         "janedoe@example.com",
				EmailVerified: true,
			},
		},
	}
	runTests(t, connectInsecureSkipVerify, c, tests)
}

func TestLDAPS(t *testing.T) {
	c := &Config{}
	c.UserSearch.BaseDN = "ou=People,ou=TestLDAPS,dc=example,dc=org"
	c.UserSearch.NameAttr = "cn"
	c.UserSearch.EmailAttr = "mail"
	c.UserSearch.IDAttr = "DN"
	c.UserSearch.Username = "cn"

	tests := []subtest{
		{
			name:     "validpassword",
			username: "jane",
			password: "foo",
			want: connector.Identity{
				UserID:        "cn=jane,ou=People,ou=TestLDAPS,dc=example,dc=org",
				Username:      "jane",
				Email:         "janedoe@example.com",
				EmailVerified: true,
			},
		},
	}
	runTests(t, connectLDAPS, c, tests)
}

func TestUsernamePrompt(t *testing.T) {
	tests := map[string]struct {
		config   Config
		expected string
	}{
		"with usernamePrompt unset it returns \"\"": {
			config:   Config{},
			expected: "",
		},
		"with usernamePrompt set it returns that": {
			config:   Config{UsernamePrompt: "Email address"},
			expected: "Email address",
		},
	}

	for n, d := range tests {
		t.Run(n, func(t *testing.T) {
			conn := &ldapConnector{Config: d.config}
			if actual := conn.Prompt(); actual != d.expected {
				t.Errorf("expected %v, got %v", d.expected, actual)
			}
		})
	}
}

func getenv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// runTests runs a set of tests against an LDAP schema.
//
// The tests require LDAP to be running.
// You can use the provided docker-compose file to setup an LDAP server.
func runTests(t *testing.T, connMethod connectionMethod, config *Config, tests []subtest) {
	ldapHost := os.Getenv("DEX_LDAP_HOST")
	if ldapHost == "" {
		t.Skipf(`test environment variable "DEX_LDAP_HOST" not set, skipping`)
	}

	// Shallow copy.
	c := *config

	// We need to configure host parameters but don't want to overwrite user or
	// group search configuration.
	switch connMethod {
	case connectStartTLS:
		c.Host = fmt.Sprintf("%s:%s", ldapHost, getenv("DEX_LDAP_PORT", "389"))
		c.RootCA = "testdata/certs/ca.crt"
		c.StartTLS = true
	case connectLDAPS:
		c.Host = fmt.Sprintf("%s:%s", ldapHost, getenv("DEX_LDAP_TLS_PORT", "636"))
		c.RootCA = "testdata/certs/ca.crt"
	case connectInsecureSkipVerify:
		c.Host = fmt.Sprintf("%s:%s", ldapHost, getenv("DEX_LDAP_TLS_PORT", "636"))
		c.InsecureSkipVerify = true
	case connectLDAP:
		c.Host = fmt.Sprintf("%s:%s", ldapHost, getenv("DEX_LDAP_PORT", "389"))
		c.InsecureNoSSL = true
	}

	c.BindDN = "cn=admin,dc=example,dc=org"
	c.BindPW = "admin"

	l := &logrus.Logger{Out: io.Discard, Formatter: &logrus.TextFormatter{}}

	conn, err := c.openConnector(l)
	if err != nil {
		t.Errorf("open connector: %v", err)
	}

	for _, test := range tests {
		if test.name == "" {
			t.Fatal("go a subtest with no name")
		}

		// Run the subtest.
		t.Run(test.name, func(t *testing.T) {
			s := connector.Scopes{OfflineAccess: true, Groups: test.groups}
			ident, validPW, err := conn.Login(context.Background(), s, test.username, test.password)
			if err != nil {
				if !test.wantErr {
					t.Fatalf("query failed: %v", err)
				}
				return
			}
			if test.wantErr {
				t.Fatalf("wanted query to fail")
			}

			if !validPW {
				if !test.wantBadPW {
					t.Fatalf("invalid password: %v", err)
				}
				return
			}

			if test.wantBadPW {
				t.Fatalf("wanted invalid password")
			}
			got := ident
			got.ConnectorData = nil

			if diff := pretty.Compare(test.want, got); diff != "" {
				t.Error(diff)
				return
			}

			// Verify that refresh tokens work.
			ident, err = conn.Refresh(context.Background(), s, ident)
			if err != nil {
				t.Errorf("refresh failed: %v", err)
			}

			got = ident
			got.ConnectorData = nil

			if diff := pretty.Compare(test.want, got); diff != "" {
				t.Errorf("after refresh: %s", diff)
			}
		})
	}
}

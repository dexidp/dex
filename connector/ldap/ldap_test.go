package ldap

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
	"github.com/sirupsen/logrus"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/dexidp/dex/connector"
)

const envVar = "DEX_LDAP_TESTS"

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
	schema := `
dn: ou=People,dc=example,dc=org
objectClass: organizationalUnit
ou: People

dn: cn=jane,ou=People,dc=example,dc=org
objectClass: person
objectClass: inetOrgPerson
sn: doe
cn: jane
mail: janedoe@example.com
userpassword: foo

dn: cn=john,ou=People,dc=example,dc=org
objectClass: person
objectClass: inetOrgPerson
sn: doe
cn: john
mail: johndoe@example.com
userpassword: bar
`
	c := &Config{}
	c.UserSearch.BaseDN = "ou=People,dc=example,dc=org"
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
				UserID:        "cn=jane,ou=People,dc=example,dc=org",
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
				UserID:        "cn=john,ou=People,dc=example,dc=org",
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

	runTests(t, schema, connectLDAP, c, tests)
}

func TestQueryWithEmailSuffix(t *testing.T) {
	schema := `
dn: ou=People,dc=example,dc=org
objectClass: organizationalUnit
ou: People

dn: cn=jane,ou=People,dc=example,dc=org
objectClass: person
objectClass: inetOrgPerson
sn: doe
cn: jane
mail: janedoe@example.com
userpassword: foo

dn: cn=john,ou=People,dc=example,dc=org
objectClass: person
objectClass: inetOrgPerson
sn: doe
cn: john
userpassword: bar
`
	c := &Config{}
	c.UserSearch.BaseDN = "ou=People,dc=example,dc=org"
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
				UserID:        "cn=jane,ou=People,dc=example,dc=org",
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
				UserID:        "cn=john,ou=People,dc=example,dc=org",
				Username:      "john",
				Email:         "john@test.example.com",
				EmailVerified: true,
			},
		},
	}

	runTests(t, schema, connectLDAP, c, tests)
}

func TestUserFilter(t *testing.T) {
	schema := `
dn: ou=Seattle,dc=example,dc=org
objectClass: organizationalUnit
ou: Seattle

dn: ou=Portland,dc=example,dc=org
objectClass: organizationalUnit
ou: Portland

dn: ou=People,ou=Seattle,dc=example,dc=org
objectClass: organizationalUnit
ou: People

dn: ou=People,ou=Portland,dc=example,dc=org
objectClass: organizationalUnit
ou: People

dn: cn=jane,ou=People,ou=Seattle,dc=example,dc=org
objectClass: person
objectClass: inetOrgPerson
sn: doe
cn: jane
mail: janedoe@example.com
userpassword: foo

dn: cn=jane,ou=People,ou=Portland,dc=example,dc=org
objectClass: person
objectClass: inetOrgPerson
sn: doe
cn: jane
mail: janedoefromportland@example.com
userpassword: baz

dn: cn=john,ou=People,ou=Seattle,dc=example,dc=org
objectClass: person
objectClass: inetOrgPerson
sn: doe
cn: john
mail: johndoe@example.com
userpassword: bar
`
	c := &Config{}
	c.UserSearch.BaseDN = "dc=example,dc=org"
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
				UserID:        "cn=jane,ou=People,ou=Seattle,dc=example,dc=org",
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
				UserID:        "cn=john,ou=People,ou=Seattle,dc=example,dc=org",
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

	runTests(t, schema, connectLDAP, c, tests)
}

func TestGroupQuery(t *testing.T) {
	schema := `
dn: ou=People,dc=example,dc=org
objectClass: organizationalUnit
ou: People

dn: cn=jane,ou=People,dc=example,dc=org
objectClass: person
objectClass: inetOrgPerson
sn: doe
cn: jane
mail: janedoe@example.com
userpassword: foo

dn: cn=john,ou=People,dc=example,dc=org
objectClass: person
objectClass: inetOrgPerson
sn: doe
cn: john
mail: johndoe@example.com
userpassword: bar

# Group definitions.

dn: ou=Groups,dc=example,dc=org
objectClass: organizationalUnit
ou: Groups

dn: cn=admins,ou=Groups,dc=example,dc=org
objectClass: groupOfNames
cn: admins
member: cn=john,ou=People,dc=example,dc=org
member: cn=jane,ou=People,dc=example,dc=org

dn: cn=developers,ou=Groups,dc=example,dc=org
objectClass: groupOfNames
cn: developers
member: cn=jane,ou=People,dc=example,dc=org
`
	c := &Config{}
	c.UserSearch.BaseDN = "ou=People,dc=example,dc=org"
	c.UserSearch.NameAttr = "cn"
	c.UserSearch.EmailAttr = "mail"
	c.UserSearch.IDAttr = "DN"
	c.UserSearch.Username = "cn"
	c.GroupSearch.BaseDN = "ou=Groups,dc=example,dc=org"
	c.GroupSearch.UserAttr = "DN"
	c.GroupSearch.GroupAttr = "member"
	c.GroupSearch.NameAttr = "cn"

	tests := []subtest{
		{
			name:     "validpassword",
			username: "jane",
			password: "foo",
			groups:   true,
			want: connector.Identity{
				UserID:        "cn=jane,ou=People,dc=example,dc=org",
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
				UserID:        "cn=john,ou=People,dc=example,dc=org",
				Username:      "john",
				Email:         "johndoe@example.com",
				EmailVerified: true,
				Groups:        []string{"admins"},
			},
		},
	}

	runTests(t, schema, connectLDAP, c, tests)
}

func TestGroupsOnUserEntity(t *testing.T) {
	schema := `
dn: ou=People,dc=example,dc=org
objectClass: organizationalUnit
ou: People

# Groups are enumerated as part of the user entity instead of the members being
# a list on the group entity.

dn: cn=jane,ou=People,dc=example,dc=org
objectClass: person
objectClass: inetOrgPerson
sn: doe
cn: jane
mail: janedoe@example.com
userpassword: foo
departmentNumber: 1000
departmentNumber: 1001

dn: cn=john,ou=People,dc=example,dc=org
objectClass: person
objectClass: inetOrgPerson
sn: doe
cn: john
mail: johndoe@example.com
userpassword: bar
departmentNumber: 1000
departmentNumber: 1002

# Group definitions. Notice that they don't have any "member" field.

dn: ou=Groups,dc=example,dc=org
objectClass: organizationalUnit
ou: Groups

dn: cn=admins,ou=Groups,dc=example,dc=org
objectClass: posixGroup
cn: admins
gidNumber: 1000

dn: cn=developers,ou=Groups,dc=example,dc=org
objectClass: posixGroup
cn: developers
gidNumber: 1001

dn: cn=designers,ou=Groups,dc=example,dc=org
objectClass: posixGroup
cn: designers
gidNumber: 1002
`
	c := &Config{}
	c.UserSearch.BaseDN = "ou=People,dc=example,dc=org"
	c.UserSearch.NameAttr = "cn"
	c.UserSearch.EmailAttr = "mail"
	c.UserSearch.IDAttr = "DN"
	c.UserSearch.Username = "cn"
	c.GroupSearch.BaseDN = "ou=Groups,dc=example,dc=org"
	c.GroupSearch.UserAttr = "departmentNumber"
	c.GroupSearch.GroupAttr = "gidNumber"
	c.GroupSearch.NameAttr = "cn"
	tests := []subtest{
		{
			name:     "validpassword",
			username: "jane",
			password: "foo",
			groups:   true,
			want: connector.Identity{
				UserID:        "cn=jane,ou=People,dc=example,dc=org",
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
				UserID:        "cn=john,ou=People,dc=example,dc=org",
				Username:      "john",
				Email:         "johndoe@example.com",
				EmailVerified: true,
				Groups:        []string{"admins", "designers"},
			},
		},
	}
	runTests(t, schema, connectLDAP, c, tests)
}

func TestGroupFilter(t *testing.T) {
	schema := `
dn: ou=People,dc=example,dc=org
objectClass: organizationalUnit
ou: People

dn: cn=jane,ou=People,dc=example,dc=org
objectClass: person
objectClass: inetOrgPerson
sn: doe
cn: jane
mail: janedoe@example.com
userpassword: foo

dn: cn=john,ou=People,dc=example,dc=org
objectClass: person
objectClass: inetOrgPerson
sn: doe
cn: john
mail: johndoe@example.com
userpassword: bar

# Group definitions.

dn: ou=Seattle,dc=example,dc=org
objectClass: organizationalUnit
ou: Seattle

dn: ou=Portland,dc=example,dc=org
objectClass: organizationalUnit
ou: Portland

dn: ou=Groups,ou=Seattle,dc=example,dc=org
objectClass: organizationalUnit
ou: Groups

dn: ou=Groups,ou=Portland,dc=example,dc=org
objectClass: organizationalUnit
ou: Groups

dn: cn=qa,ou=Groups,ou=Portland,dc=example,dc=org
objectClass: groupOfNames
cn: qa
member: cn=john,ou=People,dc=example,dc=org

dn: cn=admins,ou=Groups,ou=Seattle,dc=example,dc=org
objectClass: groupOfNames
cn: admins
member: cn=john,ou=People,dc=example,dc=org
member: cn=jane,ou=People,dc=example,dc=org

dn: cn=developers,ou=Groups,ou=Seattle,dc=example,dc=org
objectClass: groupOfNames
cn: developers
member: cn=jane,ou=People,dc=example,dc=org
`
	c := &Config{}
	c.UserSearch.BaseDN = "ou=People,dc=example,dc=org"
	c.UserSearch.NameAttr = "cn"
	c.UserSearch.EmailAttr = "mail"
	c.UserSearch.IDAttr = "DN"
	c.UserSearch.Username = "cn"
	c.GroupSearch.BaseDN = "dc=example,dc=org"
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
				UserID:        "cn=jane,ou=People,dc=example,dc=org",
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
				UserID:        "cn=john,ou=People,dc=example,dc=org",
				Username:      "john",
				Email:         "johndoe@example.com",
				EmailVerified: true,
				Groups:        []string{"admins"},
			},
		},
	}

	runTests(t, schema, connectLDAP, c, tests)
}

func TestStartTLS(t *testing.T) {
	schema := `
dn: ou=People,dc=example,dc=org
objectClass: organizationalUnit
ou: People

dn: cn=jane,ou=People,dc=example,dc=org
objectClass: person
objectClass: inetOrgPerson
sn: doe
cn: jane
mail: janedoe@example.com
userpassword: foo
`
	c := &Config{}
	c.UserSearch.BaseDN = "ou=People,dc=example,dc=org"
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
				UserID:        "cn=jane,ou=People,dc=example,dc=org",
				Username:      "jane",
				Email:         "janedoe@example.com",
				EmailVerified: true,
			},
		},
	}
	runTests(t, schema, connectStartTLS, c, tests)
}

func TestInsecureSkipVerify(t *testing.T) {
	schema := `
dn: ou=People,dc=example,dc=org
objectClass: organizationalUnit
ou: People

dn: cn=jane,ou=People,dc=example,dc=org
objectClass: person
objectClass: inetOrgPerson
sn: doe
cn: jane
mail: janedoe@example.com
userpassword: foo
`
	c := &Config{}
	c.UserSearch.BaseDN = "ou=People,dc=example,dc=org"
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
				UserID:        "cn=jane,ou=People,dc=example,dc=org",
				Username:      "jane",
				Email:         "janedoe@example.com",
				EmailVerified: true,
			},
		},
	}
	runTests(t, schema, connectInsecureSkipVerify, c, tests)
}

func TestLDAPS(t *testing.T) {
	schema := `
dn: ou=People,dc=example,dc=org
objectClass: organizationalUnit
ou: People

dn: cn=jane,ou=People,dc=example,dc=org
objectClass: person
objectClass: inetOrgPerson
sn: doe
cn: jane
mail: janedoe@example.com
userpassword: foo
`
	c := &Config{}
	c.UserSearch.BaseDN = "ou=People,dc=example,dc=org"
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
				UserID:        "cn=jane,ou=People,dc=example,dc=org",
				Username:      "jane",
				Email:         "janedoe@example.com",
				EmailVerified: true,
			},
		},
	}
	runTests(t, schema, connectLDAPS, c, tests)
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

// runTests runs a set of tests against an LDAP schema. It does this by
// setting up an OpenLDAP server and injecting the provided scheme.
//
// The tests require Docker.
//
// The DEX_LDAP_TESTS must be set to "1"
func runTests(t *testing.T, schema string, connMethod connectionMethod, config *Config, tests []subtest) {
	if os.Getenv(envVar) != "1" {
		t.Skipf("%s not set. Skipping test (run 'export %s=1' to run tests)", envVar, envVar)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	schemaPath := filepath.Join(tempDir, "schema.ldif")
	if err := ioutil.WriteFile(schemaPath, []byte(schema), 0777); err != nil {
		t.Fatal(err)
	}

	req := testcontainers.ContainerRequest{
		Image:        "osixia/openldap:1.3.0",
		ExposedPorts: []string{"389/tcp", "636/tcp"},
		Cmd:          []string{"--copy-service"},
		Env: map[string]string{
			"LDAP_BASE_DN":           "dc=example,dc=org",
			"LDAP_TLS":               "true",
			"LDAP_TLS_VERIFY_CLIENT": "try",
		},
		BindMounts: map[string]string{
			filepath.Join(wd, "testdata", "certs"): "/container/service/slapd/assets/certs",
			schemaPath:                             "/container/service/slapd/assets/config/bootstrap/ldif/99-schema.ldif",
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("slapd starting").WithOccurrence(3).WithStartupTimeout(time.Minute),
			wait.ForListeningPort("389/tcp"),
			wait.ForListeningPort("636/tcp"),
		),
	}

	ctx := context.Background()

	slapd, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		if slapd != nil {
			logs, err := slapd.Logs(ctx)
			if err == nil {
				defer logs.Close()

				logLines, err := ioutil.ReadAll(logs)
				if err != nil {
					t.Log(string(logLines))
				}
			}
		}

		t.Fatal(err)
	}
	defer slapd.Terminate(ctx)

	ip, err := slapd.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := slapd.MappedPort(ctx, "389")
	if err != nil {
		t.Fatal(err)
	}
	tlsPort, err := slapd.MappedPort(ctx, "636")
	if err != nil {
		t.Fatal(err)
	}

	// Shallow copy.
	c := *config

	// We need to configure host parameters but don't want to overwrite user or
	// group search configuration.
	switch connMethod {
	case connectStartTLS:
		c.Host = fmt.Sprintf("%s:%s", ip, port.Port())
		c.RootCA = "testdata/certs/ca.crt"
		c.StartTLS = true
	case connectLDAPS:
		c.Host = fmt.Sprintf("%s:%s", ip, tlsPort.Port())
		c.RootCA = "testdata/certs/ca.crt"
	case connectInsecureSkipVerify:
		c.Host = fmt.Sprintf("%s:%s", ip, tlsPort.Port())
		c.InsecureSkipVerify = true
	case connectLDAP:
		c.Host = fmt.Sprintf("%s:%s", ip, port.Port())
		c.InsecureNoSSL = true
	}

	c.BindDN = "cn=admin,dc=example,dc=org"
	c.BindPW = "admin"

	l := &logrus.Logger{Out: ioutil.Discard, Formatter: &logrus.TextFormatter{}}

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

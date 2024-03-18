package sql

import (
	"context"
	"reflect"
	"testing"

	"github.com/dexidp/dex/connector"

	"github.com/kylelemons/godebug/pretty"
)

type DummyLogger struct{}

func (l DummyLogger) Debug(args ...interface{})              {}
func (l DummyLogger) Info(args ...interface{})               {}
func (l DummyLogger) Warn(args ...interface{})               {}
func (l DummyLogger) Error(args ...interface{})              {}
func (l DummyLogger) Debugf(fmt string, args ...interface{}) {}
func (l DummyLogger) Infof(fmt string, args ...interface{})  {}
func (l DummyLogger) Warnf(fmt string, args ...interface{})  {}
func (l DummyLogger) Errorf(fmt string, args ...interface{}) {}

func TestSQLite3(t *testing.T) {
	config := Config{
		Database:   "sqlite3",
		Connection: ":memory:",
		Prompt:     "",

		Login:       "SELECT COUNT(*) = 1 FROM auth WHERE username = :username AND password = :password",
		GetIdentity: `SELECT username as UserID, username as Username, username as preferredUsername, username as Email, true as EmailVerified FROM auth WHERE username = :username`,
		GetGroups:   "SELECT 'group-name'",
	}

	conn, err := config.Open("", DummyLogger{})
	if err != nil {
		t.Fatalf("Error connecting to database: %s", err)
	}

	sqlconn := conn.(*sqlConnector)
	defer sqlconn.Close()

	// Create user in database
	_, err = sqlconn.db.Exec("CREATE TABLE auth ( username, password )")
	if err != nil {
		t.Fatalf("Could not create dummy auth table: %s", err)
	}
	_, err = sqlconn.db.Exec("INSERT INTO auth VALUES (\"foo\", \"bar\")")
	if err != nil {
		t.Fatalf("Could not insert dummy user: %s", err)
	}

	ctx := context.Background()

	// Check login works
	id, valid, err := sqlconn.Login(ctx, connector.Scopes{OfflineAccess: true, Groups: true}, "foo", "bar")
	if !valid || err != nil {
		t.Errorf("Expected successful auth, got '%t': %e", valid, err)
	}

	expectedID := connector.Identity{
		UserID:            "foo",
		Username:          "foo",
		PreferredUsername: "foo",
		Email:             "foo",
		EmailVerified:     true,
		Groups:            []string{"group-name"},
		ConnectorData:     []byte("foo"),
	}
	if diff := pretty.Compare(id, expectedID); diff != "" {
		t.Errorf("Unexpected identity returned: %s", diff)
	}

	// Check refresh gives us back the same thing
	id, err = sqlconn.Refresh(ctx, connector.Scopes{OfflineAccess: true, Groups: true}, id)
	if err != nil {
		t.Errorf("Expected successful refresh, got %e", err)
	}
	if diff := pretty.Compare(id, expectedID); diff != "" {
		t.Errorf("Unexpected identity after refresh: %s", diff)
	}

	// Check we get tossed with wrong creds
	id, valid, err = sqlconn.Login(ctx, connector.Scopes{OfflineAccess: true, Groups: false}, "fail", "bad")
	if valid || err != nil {
		t.Errorf("Expected failed login and or no error, got '%t': %s", valid, err)
	}
	if !reflect.DeepEqual(id, connector.Identity{}) {
		t.Errorf("Got unexpected identity: %+v", id)
	}
}

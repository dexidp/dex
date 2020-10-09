package sql

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/bcrypt"

	_ "github.com/mattn/go-sqlite3"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"github.com/dexidp/dex/connector"
)

var db *sqlx.DB

type databaseBuilder struct {
	tempDir string
	db      *sqlx.DB
	ctx     context.Context

	DatabasePath string
}

type user struct {
	UserID    string
	Name      string
	GivenName string
	Email     string
	Password  string
}

type group struct {
	GroupID string
	Name    string
}

type userGroup struct {
	UserID  string
	GroupID string
}

func (db *databaseBuilder) createUser(username string, name string,
	password string) (string, error) {

	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	userID := "u-" + uuid.String()

	cryptPw, err := bcrypt.GenerateFromPassword([]byte(password),
		bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	_, err = db.db.NamedExecContext(db.ctx,
		"INSERT INTO Users VALUES (:userid,:name,:givenname,:email,:password)",
		user{
			UserID:    userID,
			Name:      username,
			GivenName: name,
			Email:     username + "@example.com",
			Password:  string(cryptPw),
		})
	if err != nil {
		return "", err
	}

	return userID, nil
}

func (db *databaseBuilder) createGroup(groupName string) (string, error) {

	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	groupID := "g-" + uuid.String()

	_, err = db.db.NamedExecContext(db.ctx,
		"INSERT INTO Groups VALUES (:groupid, :name)",
		group{
			GroupID: groupID,
			Name:    groupName,
		})
	if err != nil {
		return "", err
	}

	return groupID, err
}

func (db *databaseBuilder) addUserToGroup(userID string, groupID string) error {

	_, err := db.db.NamedExecContext(db.ctx,
		"INSERT INTO UserGroups VALUES (:userid, :groupid)",
		userGroup{
			UserID:  userID,
			GroupID: groupID,
		})

	return err
}

func (db *databaseBuilder) Build() error {
	_, err := db.db.ExecContext(db.ctx, `CREATE TABLE Users (
  userID TEXT PRIMARY KEY,
  name TEXT,
  givenName TEXT,
  email TEXT,
  password TEXT
)`)
	if err != nil {
		return err
	}

	_, err = db.db.ExecContext(db.ctx, `CREATE TABLE Groups (
  groupID TEXT PRIMARY KEY,
  name TEXT
)`)
	if err != nil {
		return err
	}

	_, err = db.db.ExecContext(db.ctx, `CREATE TABLE UserGroups (
  userID TEXT,
  groupID TEXT,
  FOREIGN KEY (userID) REFERENCES Users(userID),
  FOREIGN KEY (groupID) REFERENCES Groups(groupID)
)`)
	if err != nil {
		return err
	}

	// Create some users (names are all made-up)
	_, err = db.createUser("jbloggs", "Joe Bloggs", "alarmingLlama")
	if err != nil {
		return err
	}

	fred_smith, err := db.createUser("fsmith", "Fred Smith", "terribleCow")
	if err != nil {
		return err
	}

	george_wilson, err := db.createUser("gwilson", "George Wilson", "awfulRaven")
	if err != nil {
		return err
	}

	amy_jones, err := db.createUser("ajones", "Amy Jones", "scaryAnt")
	if err != nil {
		return err
	}

	jane_hall, err := db.createUser("jhall", "Jane Hall", "fireyGoat")
	if err != nil {
		return err
	}

	lily_evans, err := db.createUser("levans", "Lily Evans", "irritablePig")
	if err != nil {
		return err
	}

	// Create some groups
	red, err := db.createGroup("Red Team")
	if err != nil {
		return err
	}

	green, err := db.createGroup("Green Team")
	if err != nil {
		return err
	}

	blue, err := db.createGroup("Blue Team")
	if err != nil {
		return err
	}

	// Assign users to the groups
	if err = db.addUserToGroup(fred_smith, red); err != nil {
		return err
	}

	if err = db.addUserToGroup(amy_jones, red); err != nil {
		return err
	}

	if err = db.addUserToGroup(lily_evans, red); err != nil {
		return err
	}

	if err = db.addUserToGroup(george_wilson, blue); err != nil {
		return err
	}

	if err = db.addUserToGroup(jane_hall, blue); err != nil {
		return err
	}

	if err = db.addUserToGroup(lily_evans, blue); err != nil {
		return err
	}

	if err = db.addUserToGroup(george_wilson, green); err != nil {
		return err
	}

	if err = db.addUserToGroup(amy_jones, green); err != nil {
		return err
	}

	if err = db.addUserToGroup(lily_evans, green); err != nil {
		return err
	}

	return nil
}

func (db *databaseBuilder) Close() {
	db.db.Close()
	os.RemoveAll(db.tempDir)
}

func newDatabaseBuilder(ctx context.Context) *databaseBuilder {
	tempDir, err := ioutil.TempDir("", "dexsqltest")
	if err != nil {
		panic(err)
	}

	dbPath := filepath.Join(tempDir, "test.db")

	db, err := sqlx.Open("sqlite3", "file:"+dbPath)
	if err != nil {
		panic(err)
	}

	return &databaseBuilder{tempDir: tempDir, db: db, ctx: ctx,
		DatabasePath: dbPath}
}

var dbuilder *databaseBuilder
var cfg *Config
var conn *sqlConnector

func TestMain(m *testing.M) {
	ctx := context.Background()

	dbuilder := newDatabaseBuilder(ctx)
	err := dbuilder.Build()
	if err != nil {
		panic(err)
	}
	defer dbuilder.Close()

	cfg = &Config{
		Driver: "sqlite3",
		DSN:    "file:" + dbuilder.DatabasePath,
		UserQuery: UserQuery{
			QueryByName:    "SELECT * FROM Users WHERE name=:username OR email=:username",
			QueryByID:      "SELECT * FROM Users WHERE userID=:userid",
			IDColumn:       "userID",
			UsernameColumn: "name",
			EmailColumn:    "email",
			NameColumn:     "givenName",
			PasswordColumn: "password",
		},
		UserGroupsQuery: UserGroupsQuery{
			QueryByUserID: "SELECT Groups.name FROM Groups INNER JOIN UserGroups ON Groups.groupID=UserGroups.groupID WHERE UserGroups.userID=:userid",
			NameColumn:    "name",
		},
	}

	log := &logrus.Logger{Out: ioutil.Discard, Formatter: &logrus.TextFormatter{}}

	conn, err = cfg.openConnector(log)
	if err != nil {
		panic(err)
	}

	m.Run()
}

func TestSimpleLogin(t *testing.T) {
	ctx := context.Background()

	s := connector.Scopes{OfflineAccess: false, Groups: false}

	// A non-existent user should fail
	ident, validPass, err := conn.Login(ctx, s, "another", "monkey123")
	if err != nil {
		t.Errorf("non-existent user test failed: %v", err)
	}
	if validPass {
		t.Errorf("non-existent user should not have valid password!")
	}

	// As should a user that exists but with the wrong password
	ident, validPass, err = conn.Login(ctx, s, "jbloggs", "monkey123")
	if err != nil {
		t.Errorf("bad password test failed: %v", err)
	}
	if validPass {
		t.Errorf("bad password should not be valid!")
	}

	// Users with valid passwords should work
	ident, validPass, err = conn.Login(ctx, s, "jbloggs", "alarmingLlama")
	if err != nil {
		t.Errorf("valid password test failed: %v", err)
	}
	if !validPass {
		t.Errorf("valid password should be valid!")
	}
	if ident.PreferredUsername != "jbloggs" ||
		ident.Email != "jbloggs@example.com" ||
		ident.Username != "Joe Bloggs" {
		t.Errorf("bad identity: %v", ident)
	}
}

func mustLogin(t *testing.T, ctx context.Context, scopes connector.Scopes,
	username, password string) connector.Identity {

	ident, validPass, err := conn.Login(ctx, scopes, username, password)
	if err != nil {
		t.Errorf("mustLogin failed: %v", err)
	}
	if !validPass {
		t.Errorf("mustLogin failed with invalid password")
	}
	return ident
}

func contains(s []string, v string) bool {
	for _, a := range s {
		if a == v {
			return true
		}
	}
	return false
}

func TestGroups(t *testing.T) {
	ctx := context.Background()

	s := connector.Scopes{OfflineAccess: false, Groups: true}

	// jbloggs is not in any groups
	ident := mustLogin(t, ctx, s, "jbloggs", "alarmingLlama")
	if ident.Groups == nil || len(ident.Groups) != 0 {
		t.Errorf("groups should be empty")
	}

	// fsmith is in red
	ident = mustLogin(t, ctx, s, "fsmith", "terribleCow")
	if ident.Groups == nil || len(ident.Groups) != 1 ||
		ident.Groups[0] != "Red Team" {

		t.Errorf("fsmith should be in Red Team")
	}

	// gwilson is in green and blue
	ident = mustLogin(t, ctx, s, "gwilson", "awfulRaven")
	if ident.Groups == nil || len(ident.Groups) != 2 ||
		!contains(ident.Groups, "Blue Team") ||
		!contains(ident.Groups, "Green Team") {

		t.Errorf("gwilson should be in Blue Team and Green Team")
	}

	// levans is in red, green and blue
	ident = mustLogin(t, ctx, s, "levans", "irritablePig")
	if ident.Groups == nil || len(ident.Groups) != 3 ||
		!contains(ident.Groups, "Red Team") ||
		!contains(ident.Groups, "Green Team") ||
		!contains(ident.Groups, "Blue Team") {

		t.Errorf("levans should be in Red Team, Green Team and Blue Team")
	}
}

func TestRefresh(t *testing.T) {
	ctx := context.Background()

	// First with Groups turned off
	s := connector.Scopes{OfflineAccess: true, Groups: false}

	ident := mustLogin(t, ctx, s, "jhall", "fireyGoat")
	if ident.ConnectorData == nil {
		t.Errorf("there should be connector data")
	}

	newIdent, err := conn.Refresh(ctx, s, ident)
	if err != nil {
		t.Errorf("refresh failed: %v", err)
	}
	if newIdent.UserID != ident.UserID {
		t.Errorf("refresh fetched wrong user!")
	}
	if newIdent.Groups != nil {
		t.Errorf("refresh fetched groups when told not to")
	}

	// Now turn Groups on and check that Refresh gets groups
	s = connector.Scopes{OfflineAccess: true, Groups: true}

	newIdent2, err := conn.Refresh(ctx, s, newIdent)
	if err != nil {
		t.Errorf("refresh failed: %v", err)
	}
	if newIdent2.UserID != ident.UserID {
		t.Errorf("refresh fetched wrong user!")
	}
	if newIdent2.Groups == nil || len(newIdent2.Groups) != 1 ||
		newIdent2.Groups[0] != "Blue Team" {
		t.Errorf("refresh returned wrong groups")
	}
}

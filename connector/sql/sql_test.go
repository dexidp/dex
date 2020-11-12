package sql

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/al45tair/passlib"
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
	UserID     string
	Name       string
	FoldedName string
	GivenName  string
	Email      string
	Password   string
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

	// Deliberately use a low cost, to force passlib to update them
	cryptPw, err := passlib.Hash(password)
	if err != nil {
		return "", err
	}

	_, err = db.db.NamedExecContext(db.ctx,
		"INSERT INTO Users VALUES (:userid,:name,:foldedname,:givenname,:email,:password, 0, 0)",
		user{
			UserID:     userID,
			Name:       username,
			FoldedName: caseFolder.String(username),
			GivenName:  name,
			Email:      username + "@example.com",
			Password:   string(cryptPw),
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
  name_folded TEXT,
  givenName TEXT,
  email TEXT,
  password TEXT,
  successfulLogins INTEGER,
  failedLogins INTEGER
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

	_, err = db.createUser("FRANKSPENCER", "Frank Spencer", "terrifyingPorg")
	if err != nil {
		return err
	}

	_, err = db.createUser("MaiseyGroß", "Maisey Groß", "direAnteater")
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

var testdb *sqlx.DB
var cfg *Config
var conn *sqlConnector

func TestMain(m *testing.M) {
	ctx := context.Background()

	dbuilder := newDatabaseBuilder(ctx)
	err := dbuilder.Build()
	if err != nil {
		panic(err)
	}
	testdb = dbuilder.db
	defer dbuilder.Close()

	cfg = &Config{
		Driver: "sqlite3",
		DSN:    "file:" + dbuilder.DatabasePath,
		UserQuery: UserQuery{
			QueryByName:    "SELECT * FROM Users WHERE name=:username OR email=:username OR name=:username_lower OR name=:username_upper OR name_folded=:username_casefold",
			QueryByID:      "SELECT * FROM Users WHERE userID=:userid",
			UpdatePassword: "UPDATE Users SET password=:password WHERE userID=:userid",
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
		Hooks: Hooks{
			OnSuccess: "UPDATE Users SET successfulLogins=successfulLogins+1 WHERE userID=:userid",
			OnFailure: "UPDATE Users SET failedLogins=failedLogins+1 WHERE userID=:userid",
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

func TestLowercaseLogin(t *testing.T) {
	ctx := context.Background()

	s := connector.Scopes{OfflineAccess: false, Groups: false}

	ident, validPass, err := conn.Login(ctx, s, "JBLOGGS", "alarmingLlama")
	if err != nil {
		t.Errorf("lowercase test failed: %v", err)
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

func TestUppercaseLogin(t *testing.T) {
	ctx := context.Background()

	s := connector.Scopes{OfflineAccess: false, Groups: false}

	ident, validPass, err := conn.Login(ctx, s, "frankspencer", "terrifyingPorg")
	if err != nil {
		t.Errorf("uppercase test failed: %v", err)
	}
	if !validPass {
		t.Errorf("valid password should be valid!")
	}
	if ident.PreferredUsername != "FRANKSPENCER" ||
		ident.Email != "FRANKSPENCER@example.com" ||
		ident.Username != "Frank Spencer" {
		t.Errorf("bad identity: %v", ident)
	}
}

func TestCasefoldingLogin(t *testing.T) {
	ctx := context.Background()

	s := connector.Scopes{OfflineAccess: false, Groups: false}

	ident, validPass, err := conn.Login(ctx, s, "MAISEYgross", "direAnteater")
	if err != nil {
		t.Errorf("case folding test failed: %v", err)
	}
	if !validPass {
		t.Errorf("valid password should be valid!")
	}
	if ident.PreferredUsername != "MaiseyGroß" ||
		ident.Email != "MaiseyGroß@example.com" ||
		ident.Username != "Maisey Groß" {
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

// Check that auto-upgrading hashes works, by deliberately overwriting one
// of the passwords with a weakly hashed version and then ensuring that it
// has been updated after log-in.
func TestHashUpgrade(t *testing.T) {
	ctx := context.Background()

	weakHash, err := bcrypt.GenerateFromPassword([]byte("scaryAnt"), 5)
	if err != nil {
		t.Errorf("couldn't generate weak hash: %v", err)
	}

	_, err = testdb.NamedExecContext(ctx,
		"UPDATE Users SET password=:password WHERE name=:username",
		map[string]interface{}{
			"username": "ajones",
			"password": string(weakHash),
		})
	if err != nil {
		t.Errorf("couldn't substitute weak hash: %v", err)
	}

	// Make sure we can log-in; this should also replace the hash
	s := connector.Scopes{OfflineAccess: false, Groups: false}
	mustLogin(t, ctx, s, "ajones", "scaryAnt")

	rows, err := testdb.NamedQueryContext(ctx,
		"SELECT password FROM Users WHERE name=:username",
		map[string]interface{}{
			"username": "ajones",
		})
	if err != nil {
		t.Errorf("couldn't retrieve password hash: %v", err)
	}

	if !rows.Next() {
		rows.Close()
		t.Errorf("no password hash found")
	}

	row := map[string]interface{}{}

	err = rows.MapScan(row)
	rows.Close()
	if err != nil {
		t.Errorf("couldn't retrieve password row: %v", err)
	}

	if row["password"] == string(weakHash) {
		t.Errorf("password hash has not been upgraded: %q", row["password"])
	}
}

// Test the SQL login hooks by just making them count successful and failed
// logins and check that that happens.
//
// In a real system, you might use more complex hooks and possibly change the
// search queries to take account of things they do.
func TestLoginHooks(t *testing.T) {
	ctx := context.Background()

	s := connector.Scopes{OfflineAccess: false, Groups: false}

	// Reset the counters for jbloggs
	_, err := testdb.NamedExecContext(ctx,
		"UPDATE Users SET successfulLogins=0, failedLogins=0 WHERE name=:username",
		map[string]interface{}{
			"username": "jbloggs",
		})
	if err != nil {
		t.Errorf("couldn't reset counters for jbloggs: %v", err)
	}

	// This login should fail, and should increment failedLogins
	_, validPass, err := conn.Login(ctx, s, "jbloggs", "monkey123")
	if err != nil {
		t.Errorf("bad password test failed: %v", err)
	}
	if validPass {
		t.Errorf("bad password should not be valid!")
	}

	rows, err := testdb.NamedQueryContext(ctx,
		"SELECT failedLogins FROM Users WHERE name=:username",
		map[string]interface{}{
			"username": "jbloggs",
		})
	if err != nil {
		t.Errorf("couldn't retrieve failed login count: %v", err)
	}

	if !rows.Next() {
		rows.Close()
		t.Errorf("failed login count not found")
	}

	row := map[string]interface{}{}

	err = rows.MapScan(row)
	rows.Close()
	if err != nil {
		t.Errorf("couldn't retrieve failed login row: %v", err)
	}

	failedLogins := row["failedLogins"].(int64)
	if failedLogins != 1 {
		t.Errorf("failed login count should be 1, got %d", failedLogins)
	}

	// This login should succeed, and should increment successfulLogins
	_, validPass, err = conn.Login(ctx, s, "jbloggs", "alarmingLlama")
	if err != nil {
		t.Errorf("valid password test failed: %v", err)
	}
	if !validPass {
		t.Errorf("valid password should be valid!")
	}

	rows, err = testdb.NamedQueryContext(ctx,
		"SELECT successfulLogins FROM Users WHERE name=:username",
		map[string]interface{}{
			"username": "jbloggs",
		})
	if err != nil {
		t.Errorf("couldn't retrieve successful login count: %v", err)
	}

	if !rows.Next() {
		rows.Close()
		t.Errorf("successful login count not found")
	}

	row = map[string]interface{}{}

	err = rows.MapScan(row)
	rows.Close()
	if err != nil {
		t.Errorf("couldn't retrieve successful login row: %v", err)
	}

	successfulLogins := row["successfulLogins"].(int64)
	if successfulLogins != 1 {
		t.Errorf("successful login count should be 1, got %d", successfulLogins)
	}
}


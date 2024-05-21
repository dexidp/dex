package authproxy

import (
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/dexidp/dex/connector"
)

const (
	testEmail        = "testuser@example.com"
	testGroup1       = "group1"
	testGroup2       = "group2"
	testGroup3       = "group 3"
	testGroup4       = "group 4"
	testStaticGroup1 = "static1"
	testStaticGroup2 = "static 2"
	testUsername     = "testuser"
	testUserID       = "1234567890"
)

var logger = &logrus.Logger{Out: io.Discard, Formatter: &logrus.TextFormatter{}}

func TestUser(t *testing.T) {
	config := Config{
		UserHeader: "X-Remote-User",
	}
	conn := callback{userHeader: config.UserHeader, logger: logger, pathSuffix: "/test"}

	req, err := http.NewRequest("GET", "/", nil)
	expectNil(t, err)
	req.Header = map[string][]string{
		"X-Remote-User": {testUsername},
	}

	ident, err := conn.HandleCallback(connector.Scopes{OfflineAccess: true, Groups: true}, req)
	expectNil(t, err)

	// If not specified, the userID and email should fall back to the remote user
	expectEquals(t, ident.UserID, testUsername)
	expectEquals(t, ident.PreferredUsername, testUsername)
	expectEquals(t, ident.Username, testUsername)
	expectEquals(t, ident.Email, testUsername)
	expectEquals(t, len(ident.Groups), 0)
}

func TestExtraHeaders(t *testing.T) {
	config := Config{
		UserIDHeader: "X-Remote-User-Id",
		UserHeader:   "X-Remote-User",
		EmailHeader:  "X-Remote-User-Email",
	}
	conn := callback{
		userHeader:   config.UserHeader,
		userIDHeader: config.UserIDHeader,
		emailHeader:  config.EmailHeader,
		logger:       logger,
		pathSuffix:   "/test",
	}

	req, err := http.NewRequest("GET", "/", nil)
	expectNil(t, err)
	req.Header = map[string][]string{
		"X-Remote-User-Id":    {testUserID},
		"X-Remote-User":       {testUsername},
		"X-Remote-User-Email": {testEmail},
	}

	ident, err := conn.HandleCallback(connector.Scopes{OfflineAccess: true, Groups: true}, req)
	expectNil(t, err)

	expectEquals(t, ident.UserID, testUserID)
	expectEquals(t, ident.PreferredUsername, testUsername)
	expectEquals(t, ident.Username, testUsername)
	expectEquals(t, ident.Email, testEmail)
	expectEquals(t, len(ident.Groups), 0)
}

func TestSingleGroup(t *testing.T) {
	config := Config{
		UserHeader:  "X-Remote-User",
		GroupHeader: "X-Remote-Group",
	}

	conn := callback{userHeader: config.UserHeader, groupHeader: config.GroupHeader, logger: logger, pathSuffix: "/test"}

	req, err := http.NewRequest("GET", "/", nil)
	expectNil(t, err)
	req.Header = map[string][]string{
		"X-Remote-User":  {testEmail},
		"X-Remote-Group": {testGroup1},
	}

	ident, err := conn.HandleCallback(connector.Scopes{OfflineAccess: true, Groups: true}, req)
	expectNil(t, err)

	expectEquals(t, ident.UserID, testEmail)
	expectEquals(t, len(ident.Groups), 1)
	expectEquals(t, ident.Groups[0], testGroup1)
}

func TestMultipleGroup(t *testing.T) {
	config := Config{
		UserHeader:  "X-Remote-User",
		GroupHeader: "X-Remote-Group",
	}

	conn := callback{userHeader: config.UserHeader, groupHeader: config.GroupHeader, logger: logger, pathSuffix: "/test"}

	req, err := http.NewRequest("GET", "/", nil)
	expectNil(t, err)
	req.Header = map[string][]string{
		"X-Remote-User":  {testEmail},
		"X-Remote-Group": {testGroup1 + ", " + testGroup2 + ", " + testGroup3 + ", " + testGroup4},
	}

	ident, err := conn.HandleCallback(connector.Scopes{OfflineAccess: true, Groups: true}, req)
	expectNil(t, err)

	expectEquals(t, ident.UserID, testEmail)
	expectEquals(t, len(ident.Groups), 4)
	expectEquals(t, ident.Groups[0], testGroup1)
	expectEquals(t, ident.Groups[1], testGroup2)
	expectEquals(t, ident.Groups[2], testGroup3)
	expectEquals(t, ident.Groups[3], testGroup4)
}

func TestStaticGroup(t *testing.T) {
	config := Config{
		UserHeader:  "X-Remote-User",
		GroupHeader: "X-Remote-Group",
		Groups:      []string{"static1", "static 2"},
	}

	conn := callback{userHeader: config.UserHeader, groupHeader: config.GroupHeader, groups: config.Groups, logger: logger, pathSuffix: "/test"}

	req, err := http.NewRequest("GET", "/", nil)
	expectNil(t, err)
	req.Header = map[string][]string{
		"X-Remote-User":  {testEmail},
		"X-Remote-Group": {testGroup1 + ", " + testGroup2 + ", " + testGroup3 + ", " + testGroup4},
	}

	ident, err := conn.HandleCallback(connector.Scopes{OfflineAccess: true, Groups: true}, req)
	expectNil(t, err)

	expectEquals(t, ident.UserID, testEmail)
	expectEquals(t, len(ident.Groups), 6)
	expectEquals(t, ident.Groups[0], testGroup1)
	expectEquals(t, ident.Groups[1], testGroup2)
	expectEquals(t, ident.Groups[2], testGroup3)
	expectEquals(t, ident.Groups[3], testGroup4)
	expectEquals(t, ident.Groups[4], testStaticGroup1)
	expectEquals(t, ident.Groups[5], testStaticGroup2)
}

func expectNil(t *testing.T, a interface{}) {
	if a != nil {
		t.Errorf("Expected %+v to equal nil", a)
	}
}

func expectEquals(t *testing.T, a interface{}, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Expected %+v to equal %+v", a, b)
	}
}

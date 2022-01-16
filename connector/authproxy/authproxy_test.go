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
	testUserid = "test-user"
	testEmail  = "testuser@example.com"
	testGroup1 = "allUsers"
	testGroup2 = "admins"
)

var logger = &logrus.Logger{Out: io.Discard, Formatter: &logrus.TextFormatter{}}

func TestExtractUserFromHeaders(t *testing.T) {
	config := Config{
		UserHeader: "X-Remote-User",
	}

	conn := callback{config: config, logger: logger, pathSuffix: "/test"}

	req, err := http.NewRequest("GET", "/", nil)
	expectNil(t, err)
	req.Header = map[string][]string{
		"X-Remote-User": {testEmail},
	}

	ident, err := conn.HandleCallback(connector.Scopes{OfflineAccess: true, Groups: true}, req)
	expectNil(t, err)

	expectEquals(t, ident.UserID, testEmail)
	expectEquals(t, ident.Email, testEmail)
	expectEquals(t, len(ident.Groups), 0)
}

func TestExtractEmailFromHeaders(t *testing.T) {
	config := Config{
		UserHeader:  "X-Remote-User",
		EmailHeader: "X-Email",
	}

	conn := callback{config: config, logger: logger, pathSuffix: "/test"}

	req, err := http.NewRequest("GET", "/", nil)
	expectNil(t, err)
	req.Header = map[string][]string{
		"X-Remote-User": {testUserid},
		"X-Email":       {testEmail},
	}

	ident, err := conn.HandleCallback(connector.Scopes{OfflineAccess: true, Groups: true}, req)
	expectNil(t, err)

	expectEquals(t, ident.UserID, testUserid)
	expectEquals(t, ident.Email, testEmail)
	expectEquals(t, len(ident.Groups), 0)
}

func TestExtractGroupsWithoutScopes(t *testing.T) {
	config := Config{
		UserHeader:  "X-Remote-User",
		GroupHeader: "X-Remote-Group",
	}

	conn := callback{config: config, logger: logger, pathSuffix: "/test"}

	req, err := http.NewRequest("GET", "/", nil)
	expectNil(t, err)
	req.Header = map[string][]string{
		"X-Remote-User":  {testEmail},
		"X-Remote-Group": {testGroup1},
	}

	ident, err := conn.HandleCallback(connector.Scopes{OfflineAccess: true, Groups: false}, req)
	expectNil(t, err)

	expectEquals(t, ident.UserID, testEmail)
	expectEquals(t, len(ident.Groups), 0)
}

func TestExtractGroupsFromSingleHeader(t *testing.T) {
	config := Config{
		UserHeader:     "X-Remote-User",
		GroupHeader:    "X-Remote-Group",
		GroupSeparator: ",",
	}

	conn := callback{config: config, logger: logger, pathSuffix: "/test"}

	req, err := http.NewRequest("GET", "/", nil)
	expectNil(t, err)
	req.Header = map[string][]string{
		"X-Remote-User":  {testEmail},
		"X-Remote-Group": {testGroup1 + "," + testGroup2},
	}

	ident, err := conn.HandleCallback(connector.Scopes{OfflineAccess: true, Groups: true}, req)
	expectNil(t, err)

	expectEquals(t, ident.UserID, testEmail)
	expectEquals(t, len(ident.Groups), 2)
	expectEquals(t, ident.Groups[0], testGroup1)
	expectEquals(t, ident.Groups[1], testGroup2)
}

func TestExtractGroupsFromMultiHeaders(t *testing.T) {
	config := Config{
		UserHeader:  "X-Remote-User",
		GroupHeader: "X-Remote-Group",
	}

	conn := callback{config: config, logger: logger, pathSuffix: "/test"}

	req, err := http.NewRequest("GET", "/", nil)
	expectNil(t, err)
	req.Header = map[string][]string{
		"X-Remote-User":  {testEmail},
		"X-Remote-Group": {testGroup1, testGroup2},
	}

	ident, err := conn.HandleCallback(connector.Scopes{OfflineAccess: true, Groups: true}, req)
	expectNil(t, err)

	expectEquals(t, ident.UserID, testEmail)
	expectEquals(t, len(ident.Groups), 2)
	expectEquals(t, ident.Groups[0], testGroup1)
	expectEquals(t, ident.Groups[1], testGroup2)
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

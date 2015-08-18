// +build acceptance

// The acceptance test package includes utilities supporting acceptance tests in *_test.go
// files.  To execute these acceptance tests, you must invoke them using the acceptance
// build tag, like so:
//
// $ go test -tags acceptance github.com/mailgun/mailgun-go
//
// Note that some API calls may potentially cost the user money!  By default, such tests
// do NOT run.  However, you will then not be testing the full capability of Mailgun.
// To run them, you'll also need to specify the spendMoney build tag:
//
// $ go test -tags "acceptance spendMoney" github.com/mailgun/mailgun-go
package acceptance

import (
	"os"
	"testing"
)

// Many tests require configuration settings unique to the user, passed in via
// environment variables.  If these variables aren't set, we need to fail the test early.
func reqEnv(t *testing.T, variableName string) string {
	value := os.Getenv(variableName)
	if value == "" {
		t.Fatalf("Expected environment variable %s to be set", variableName)
	}
	return value
}

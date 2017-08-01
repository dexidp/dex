package github

import (
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/coreos/dex/connector"
	"github.com/sirupsen/logrus"
)

func TestGithubConnector_LoginURL(t *testing.T) {
	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: &logrus.TextFormatter{DisableColors: true},
		Level:     logrus.DebugLevel,
	}

	tests := []struct {
		scopes connector.Scopes
		org    string

		wantedScopes string
	}{
		{
			connector.Scopes{}, "",
			scopeEmail,
		},
		{
			connector.Scopes{Groups: true}, "",
			scopeEmail + " " + scopeOrgs,
		},
		{
			connector.Scopes{Groups: true, OfflineAccess: true}, "",
			scopeEmail + " " + scopeOrgs,
		},
		{
			connector.Scopes{}, "test-org",
			scopeEmail,
		},
		{
			connector.Scopes{Groups: true}, "test-org",
			scopeEmail + " " + scopeOrgs,
		},
		{
			connector.Scopes{Groups: true, OfflineAccess: true}, "test-org",
			scopeEmail + " " + scopeOrgs,
		},
	}

	callback := "https://dex.example.com/callback"
	state := "secret"

	for _, test := range tests {
		config := &Config{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			RedirectURI:  "https://dex.example.com/callback",
			Org:          test.org,
		}

		conn, err := config.Open(logger)
		if err != nil {
			t.Errorf("failed to open connector: %v", err)
			continue
		}

		loginURL, err := conn.(connector.CallbackConnector).LoginURL(test.scopes, callback, state)
		if err != nil {
			t.Errorf("failed to get login URL: %v", err)
			continue
		}

		actual, err := url.Parse(loginURL)
		if err != nil {
			t.Errorf("failed to parse login URL: %v", err)
			continue
		}

		wanted, _ := url.Parse("https://github.com/login/oauth/authorize")
		wantedQuery := &url.Values{}
		wantedQuery.Set("client_id", config.ClientID)
		wantedQuery.Set("response_type", "code")
		wantedQuery.Set("state", "secret")
		wantedQuery.Set("scope", test.wantedScopes)
		wanted.RawQuery = wantedQuery.Encode()

		if !reflect.DeepEqual(actual, wanted) {
			t.Errorf("Github GetLogin failed\nwanted %v\ngot    %v", wanted, actual)
		}
	}
}

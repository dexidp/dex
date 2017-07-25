package oidc

import (
	"github.com/coreos/dex/connector"
	"github.com/sirupsen/logrus"
	"net/url"
	"os"
	"reflect"
	"testing"
)

func TestKnownBrokenAuthHeaderProvider(t *testing.T) {
	tests := []struct {
		issuerURL string
		expect    bool
	}{
		{"https://dev.oktapreview.com", true},
		{"https://dev.okta.com", true},
		{"https://okta.com", true},
		{"https://dev.oktaaccounts.com", false},
		{"https://accounts.google.com", false},
	}

	for _, tc := range tests {
		got := knownBrokenAuthHeaderProvider(tc.issuerURL)
		if got != tc.expect {
			t.Errorf("knownBrokenAuthHeaderProvider(%q), want=%t, got=%t", tc.issuerURL, tc.expect, got)
		}
	}
}

func TestOidcConnector_LoginURL(t *testing.T) {
	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: &logrus.TextFormatter{DisableColors: true},
		Level:     logrus.DebugLevel,
	}

	tests := []struct {
		scopes        connector.Scopes
		hostedDomains []string

		wantScopes  string
		wantHdParam string
	}{
		{
			connector.Scopes{}, []string{"example.com"},
			"openid profile email", "example.com",
		},
		{
			connector.Scopes{}, []string{"mydomain.org", "example.com"},
			"openid profile email", "*",
		},
		{
			connector.Scopes{}, []string{},
			"openid profile email", "",
		},
		{
			connector.Scopes{OfflineAccess: true}, []string{},
			"openid profile email", "",
		},
	}

	callback := "https://dex.example.com/callback"
	state := "secret"

	for _, test := range tests {
		config := &Config{
			Issuer:        "https://accounts.google.com",
			ClientID:      "client-id",
			ClientSecret:  "client-secret",
			RedirectURI:   "https://dex.example.com/callback",
			HostedDomains: test.hostedDomains,
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

		wanted, _ := url.Parse("https://accounts.google.com/o/oauth2/v2/auth")
		wantedQuery := &url.Values{}
		wantedQuery.Set("client_id", config.ClientID)
		wantedQuery.Set("redirect_uri", config.RedirectURI)
		wantedQuery.Set("response_type", "code")
		wantedQuery.Set("state", "secret")
		wantedQuery.Set("scope", test.wantScopes)
		if test.wantHdParam != "" {
			wantedQuery.Set("hd", test.wantHdParam)
		}
		wanted.RawQuery = wantedQuery.Encode()

		if !reflect.DeepEqual(actual, wanted) {
			t.Errorf("Wanted %v, got %v", wanted, actual)
		}
	}
}

//func TestOidcConnector_HandleCallback(t *testing.T) {
//	logger := &logrus.Logger{
//		Out:       os.Stderr,
//		Formatter: &logrus.TextFormatter{DisableColors: true},
//		Level:     logrus.DebugLevel,
//	}
//
//	tests := []struct {
//
//	}
//}

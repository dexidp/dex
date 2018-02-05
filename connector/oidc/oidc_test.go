package oidc

import (
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

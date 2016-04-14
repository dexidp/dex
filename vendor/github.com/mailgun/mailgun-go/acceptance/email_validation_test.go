// +build acceptance

package acceptance

import (
	"github.com/mailgun/mailgun-go"
	"testing"
)

func TestEmailValidation(t *testing.T) {
	domain := reqEnv(t, "MG_DOMAIN")
	apiKey := reqEnv(t, "MG_PUBLIC_API_KEY")
	mg := mailgun.NewMailgun(domain, "", apiKey)
	ev, err := mg.ValidateEmail("foo@mailgun.com")
	if err != nil {
		t.Fatal(err)
	}
	if ev.IsValid != true {
		t.Fatal("Expected a valid e-mail address")
	}
	if ev.Parts.DisplayName != "" {
		t.Fatal("No display name should exist")
	}
	if ev.Parts.LocalPart != "foo" {
		t.Fatal("Expected local part of foo; got ", ev.Parts.LocalPart)
	}
	if ev.Parts.Domain != "mailgun.com" {
		t.Fatal("Expected mailgun.com domain; got ", ev.Parts.Domain)
	}
}

func TestParseAddresses(t *testing.T) {
	domain := reqEnv(t, "MG_DOMAIN")
	apiKey := reqEnv(t, "MG_PUBLIC_API_KEY")
	mg := mailgun.NewMailgun(domain, "", apiKey)
	addressesThatParsed, unparsableAddresses, err := mg.ParseAddresses("Alice <alice@example.com>", "bob@example.com", "example.com")
	if err != nil {
		t.Fatal(err)
	}
	hittest := map[string]bool{
		"Alice <alice@example.com>": true,
		"bob@example.com":           true,
	}
	for _, a := range addressesThatParsed {
		if !hittest[a] {
			t.Fatalf("Expected %s to be parsable", a)
		}
	}
	if len(unparsableAddresses) != 1 {
		t.Fatalf("Expected 1 address to be unparsable; got %d", len(unparsableAddresses))
	}
}

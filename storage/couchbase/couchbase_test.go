package couchbase

import (
	"fmt"
	"testing"
)

func TestGenerateKeyID(t *testing.T) {
	got := keyID(connectorKey, "id-connector")
	wanted := "dex-connector-id-connector"
	if got != wanted {
		t.Fatalf("Expected key ID to be %q, got %q", wanted, got)
	}
}

func TestGeneratekeyEmail(t *testing.T) {
	got := keyEmail(passwordKey, "User@user.com")
	wanted := "dex-password-user@user.com"
	if got != wanted {
		t.Fatalf("Expected keyEmail to be %q, got %q", wanted, got)
	}
}

func TestGeneratekeySession(t *testing.T) {
	got := keySession(offlineSessionKey, "userID", "connID")
	wanted := "dex-offlinesession-userid-connid"
	if got != wanted {
		t.Fatalf("Expected keySession to be %q, got %q", wanted, got)
	}
}

func TestAlreadyExistsCheck(t *testing.T) {
	err := fmt.Errorf("key already exists")
	got := alreadyExistsCheck(err)
	if !got {
		t.Fatalf("Expected error that key already exists")
	}
}

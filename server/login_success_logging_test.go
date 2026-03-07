package server

import (
	"testing"

	"github.com/dexidp/dex/storage"
)

func logArgsMap(t *testing.T, args []any) map[string]any {
	t.Helper()

	if len(args)%2 != 0 {
		t.Fatalf("expected keyvals to have even length, got %d", len(args))
	}

	m := make(map[string]any, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			t.Fatalf("expected key at index %d to be string, got %T", i, args[i])
		}
		m[key] = args[i+1]
	}
	return m
}

func TestLoginSuccessfulLogArgs(t *testing.T) {
	claims := storage.Claims{
		UserID:            "user-id",
		Username:          "username",
		PreferredUsername: "preferred-username",
		Email:             "user@example.com",
		EmailVerified:     false,
		Groups:            []string{"group-1", "group-2"},
	}

	t.Run("enabled", func(t *testing.T) {
		args := loginSuccessfulLogArgs("connector-id", claims, true)
		if len(args)%2 != 0 {
			t.Fatalf("expected keyvals to have even length, got %d", len(args))
		}

		keyvals := logArgsMap(t, args)
		for _, key := range []string{"username", "preferred_username", "email", "groups"} {
			if _, ok := keyvals[key]; !ok {
				t.Fatalf("expected key %q to be present", key)
			}
		}
		email, ok := keyvals["email"].(string)
		if !ok {
			t.Fatalf("expected email to be string, got %T", keyvals["email"])
		}
		if want := claims.Email + " (unverified)"; email != want {
			t.Fatalf("expected email %q, got %q", want, email)
		}
	})

	t.Run("disabled", func(t *testing.T) {
		args := loginSuccessfulLogArgs("connector-id", claims, false)
		if len(args)%2 != 0 {
			t.Fatalf("expected keyvals to have even length, got %d", len(args))
		}

		keyvals := logArgsMap(t, args)
		if _, ok := keyvals["connector_id"]; !ok {
			t.Fatalf("expected key %q to be present", "connector_id")
		}
		if userID, ok := keyvals["user_id"]; !ok {
			t.Fatalf("expected key %q to be present", "user_id")
		} else if userID != claims.UserID {
			t.Fatalf("expected user_id %q, got %v", claims.UserID, userID)
		}
		for _, key := range []string{"username", "preferred_username", "email", "groups"} {
			if _, ok := keyvals[key]; ok {
				t.Fatalf("expected key %q to be absent", key)
			}
		}
	})
}

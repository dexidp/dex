package conformance

import (
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/dexidp/dex/storage"
)

// RunTransactionTests runs a test suite aimed a verifying the transaction
// guarantees of the storage interface. Atomic updates, deletes, etc. The
// storage returned by newStorage will be closed at the end of each test run.
//
// This call is separate from RunTests because some storage perform extremely
// poorly under deadlocks, such as SQLite3, while others may be working towards
// conformance.
func RunTransactionTests(t *testing.T, newStorage func() storage.Storage) {
	runTests(t, newStorage, []subTest{
		{"AuthRequestConcurrentUpdate", testAuthRequestConcurrentUpdate},
		{"ClientConcurrentUpdate", testClientConcurrentUpdate},
		{"PasswordConcurrentUpdate", testPasswordConcurrentUpdate},
		{"KeysConcurrentUpdate", testKeysConcurrentUpdate},
	})
}

func testClientConcurrentUpdate(t *testing.T, s storage.Storage) {
	c := storage.Client{
		ID:           storage.NewID(),
		Secret:       "foobar",
		RedirectURIs: []string{"foo://bar.com/", "https://auth.example.com"},
		Name:         "dex client",
		LogoURL:      "https://goo.gl/JIyzIC",
	}

	if err := s.CreateClient(c); err != nil {
		t.Fatalf("create client: %v", err)
	}

	var err1, err2 error

	err1 = s.UpdateClient(c.ID, func(old storage.Client) (storage.Client, error) {
		old.Secret = "new secret 1"
		err2 = s.UpdateClient(c.ID, func(old storage.Client) (storage.Client, error) {
			old.Secret = "new secret 2"
			return old, nil
		})
		return old, nil
	})

	if (err1 == nil) == (err2 == nil) {
		t.Errorf("update client:\nupdate1: %v\nupdate2: %v\n", err1, err2)
	}
}

func testAuthRequestConcurrentUpdate(t *testing.T, s storage.Storage) {
	a := storage.AuthRequest{
		ID:                  storage.NewID(),
		ClientID:            "foobar",
		ResponseTypes:       []string{"code"},
		Scopes:              []string{"openid", "email"},
		RedirectURI:         "https://localhost:80/callback",
		Nonce:               "foo",
		State:               "bar",
		ForceApprovalPrompt: true,
		LoggedIn:            true,
		Expiry:              neverExpire,
		ConnectorID:         "ldap",
		Claims: storage.Claims{
			UserID:        "1",
			Username:      "jane",
			Email:         "jane.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
	}

	if err := s.CreateAuthRequest(a); err != nil {
		t.Fatalf("failed creating auth request: %v", err)
	}

	var err1, err2 error

	err1 = s.UpdateAuthRequest(a.ID, func(old storage.AuthRequest) (storage.AuthRequest, error) {
		old.State = "state 1"
		err2 = s.UpdateAuthRequest(a.ID, func(old storage.AuthRequest) (storage.AuthRequest, error) {
			old.State = "state 2"
			return old, nil
		})
		return old, nil
	})

	if (err1 == nil) == (err2 == nil) {
		t.Errorf("update auth request:\nupdate1: %v\nupdate2: %v\n", err1, err2)
	}
}

func testPasswordConcurrentUpdate(t *testing.T, s storage.Storage) {
	// Use bcrypt.MinCost to keep the tests short.
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	password := storage.Password{
		Email:    "jane@example.com",
		Hash:     passwordHash,
		Username: "jane",
		UserID:   "foobar",
	}
	if err := s.CreatePassword(password); err != nil {
		t.Fatalf("create password token: %v", err)
	}

	var err1, err2 error

	err1 = s.UpdatePassword(password.Email, func(old storage.Password) (storage.Password, error) {
		old.Username = "user 1"
		err2 = s.UpdatePassword(password.Email, func(old storage.Password) (storage.Password, error) {
			old.Username = "user 2"
			return old, nil
		})
		return old, nil
	})

	if (err1 == nil) == (err2 == nil) {
		t.Errorf("update password: concurrent updates both returned no error")
	}
}

func testKeysConcurrentUpdate(t *testing.T, s storage.Storage) {
	// Test twice. Once for a create, once for an update.
	for i := 0; i < 2; i++ {
		n := time.Now().UTC().Round(time.Second)
		keys1 := storage.Keys{
			SigningKey:    jsonWebKeys[0].Private,
			SigningKeyPub: jsonWebKeys[0].Public,
			NextRotation:  n,
		}

		keys2 := storage.Keys{
			SigningKey:    jsonWebKeys[2].Private,
			SigningKeyPub: jsonWebKeys[2].Public,
			NextRotation:  n.Add(time.Hour),
			VerificationKeys: []storage.VerificationKey{
				{
					PublicKey: jsonWebKeys[0].Public,
					Expiry:    n.Add(time.Hour),
				},
				{
					PublicKey: jsonWebKeys[1].Public,
					Expiry:    n.Add(time.Hour * 2),
				},
			},
		}

		var err1, err2 error

		err1 = s.UpdateKeys(func(old storage.Keys) (storage.Keys, error) {
			err2 = s.UpdateKeys(func(old storage.Keys) (storage.Keys, error) {
				return keys1, nil
			})
			return keys2, nil
		})

		if (err1 == nil) == (err2 == nil) {
			t.Errorf("update keys: concurrent updates both returned no error")
		}
	}
}

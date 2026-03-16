package conformance

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
func RunTransactionTests(t *testing.T, newStorage func(t *testing.T) storage.Storage) {
	runTests(t, newStorage, []subTest{
		{"AuthRequestConcurrentUpdate", testAuthRequestConcurrentUpdate},
		{"ClientConcurrentUpdate", testClientConcurrentUpdate},
		{"PasswordConcurrentUpdate", testPasswordConcurrentUpdate},
		{"KeysConcurrentUpdate", testKeysConcurrentUpdate},
	})
}

// RunConcurrencyTests runs tests that verify storage implementations handle
// high-contention parallel updates correctly. Unlike RunTransactionTests,
// these tests use real goroutine-based parallelism rather than nested calls,
// and are safe to run on all storage backends (including those with non-reentrant locks).
func RunConcurrencyTests(t *testing.T, newStorage func(t *testing.T) storage.Storage) {
	runTests(t, newStorage, []subTest{
		{"RefreshTokenParallelUpdate", testRefreshTokenParallelUpdate},
	})
}

func testClientConcurrentUpdate(t *testing.T, s storage.Storage) {
	ctx := t.Context()
	c := storage.Client{
		ID:           storage.NewID(),
		Secret:       "foobar",
		RedirectURIs: []string{"foo://bar.com/", "https://auth.example.com"},
		Name:         "dex client",
		LogoURL:      "https://goo.gl/JIyzIC",
	}

	if err := s.CreateClient(ctx, c); err != nil {
		t.Fatalf("create client: %v", err)
	}

	var err1, err2 error

	err1 = s.UpdateClient(ctx, c.ID, func(old storage.Client) (storage.Client, error) {
		old.Secret = "new secret 1"
		err2 = s.UpdateClient(ctx, c.ID, func(old storage.Client) (storage.Client, error) {
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
	ctx := t.Context()
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
		ConnectorData:       []byte(`{"some":"data"}`),
		Claims: storage.Claims{
			UserID:        "1",
			Username:      "jane",
			Email:         "jane.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
		HMACKey: []byte("hmac_key"),
	}

	if err := s.CreateAuthRequest(ctx, a); err != nil {
		t.Fatalf("failed creating auth request: %v", err)
	}

	var err1, err2 error

	err1 = s.UpdateAuthRequest(ctx, a.ID, func(old storage.AuthRequest) (storage.AuthRequest, error) {
		old.State = "state 1"
		err2 = s.UpdateAuthRequest(ctx, a.ID, func(old storage.AuthRequest) (storage.AuthRequest, error) {
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
	ctx := t.Context()
	// Use bcrypt.MinCost to keep the tests short.
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	password := storage.Password{
		Email:             "jane@example.com",
		Hash:              passwordHash,
		Username:          "jane",
		Name:              "Jane Doe",
		PreferredUsername: "jane-public",
		EmailVerified:     boolPtr(true),
		UserID:            "foobar",
		Groups:            []string{"team-a"},
	}
	if err := s.CreatePassword(ctx, password); err != nil {
		t.Fatalf("create password token: %v", err)
	}

	var err1, err2 error

	err1 = s.UpdatePassword(ctx, password.Email, func(old storage.Password) (storage.Password, error) {
		old.Username = "user 1"
		err2 = s.UpdatePassword(ctx, password.Email, func(old storage.Password) (storage.Password, error) {
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
			SigningKey:    jsonWebKeys[i].Private,
			SigningKeyPub: jsonWebKeys[i].Public,
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

		ctx := context.TODO()
		err1 = s.UpdateKeys(ctx, func(old storage.Keys) (storage.Keys, error) {
			err2 = s.UpdateKeys(ctx, func(old storage.Keys) (storage.Keys, error) {
				return keys1, nil
			})
			return keys2, nil
		})

		if (err1 == nil) == (err2 == nil) {
			t.Errorf("update keys: concurrent updates both returned no error")
		}
	}
}

// testRefreshTokenParallelUpdate tests that many parallel updates to the same
// refresh token are serialized correctly by the storage and no updates are lost.
//
// Each goroutine atomically increments a counter stored in the Token field.
// After all goroutines finish, the counter must equal the number of successful updates.
// A mismatch indicates lost updates due to broken atomicity.
func testRefreshTokenParallelUpdate(t *testing.T, s storage.Storage) {
	ctx := t.Context()

	id := storage.NewID()
	refresh := storage.RefreshToken{
		ID:          id,
		Token:       "0",
		Nonce:       "foo",
		ClientID:    "client_id",
		ConnectorID: "connector_id",
		Scopes:      []string{"openid"},
		CreatedAt:   time.Now().UTC().Round(time.Millisecond),
		LastUsed:    time.Now().UTC().Round(time.Millisecond),
		Claims: storage.Claims{
			UserID:   "1",
			Username: "jane",
			Email:    "jane@example.com",
		},
	}

	require.NoError(t, s.CreateRefresh(ctx, refresh))

	const numWorkers = 100

	type updateResult struct {
		err      error
		newToken string // token value written by this worker's updater
	}

	var wg sync.WaitGroup
	results := make([]updateResult, numWorkers)

	for i := range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i].err = s.UpdateRefreshToken(ctx, id, func(old storage.RefreshToken) (storage.RefreshToken, error) {
				counter, _ := strconv.Atoi(old.Token)
				old.Token = strconv.Itoa(counter + 1)
				results[i].newToken = old.Token
				return old, nil
			})
		}()
	}

	wg.Wait()

	errCounts := map[string]int{}
	var successes int
	writtenTokens := map[string]int{}
	for _, r := range results {
		if r.err == nil {
			successes++
			writtenTokens[r.newToken]++
		} else {
			errCounts[r.err.Error()]++
		}
	}

	for msg, count := range errCounts {
		t.Logf("error (x%d): %s", count, msg)
	}

	stored, err := s.GetRefresh(ctx, id)
	require.NoError(t, err)

	counter, err := strconv.Atoi(stored.Token)
	require.NoError(t, err)

	t.Logf("parallel refresh token updates: %d/%d succeeded, final counter: %d", successes, numWorkers, counter)

	if successes < numWorkers {
		t.Errorf("not all updates succeeded: %d/%d (some failed under contention)", successes, numWorkers)
	}

	if counter != successes {
		t.Errorf("lost updates detected: %d successful updates but counter is %d", successes, counter)
	}

	// Each successful updater must have seen a unique counter value.
	// Duplicates would mean two updaters read the same state — a sign of broken atomicity.
	for token, count := range writtenTokens {
		if count > 1 {
			t.Errorf("token %q was written by %d updaters — concurrent updaters saw the same state", token, count)
		}
	}

	// Successful updaters must have produced a contiguous sequence 1..N.
	// A gap would mean an updater saw stale state even though the write succeeded.
	for i := 1; i <= successes; i++ {
		if writtenTokens[strconv.Itoa(i)] != 1 {
			t.Errorf("expected token %q to be written exactly once, got %d", strconv.Itoa(i), writtenTokens[strconv.Itoa(i)])
		}
	}

	// The token stored in the database must match the highest value written.
	// This confirms that the last successful update is the one persisted.
	if stored.Token != strconv.Itoa(successes) {
		t.Errorf("stored token %q does not match expected final value %q", stored.Token, strconv.Itoa(successes))
	}
}

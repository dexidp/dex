// +build go1.7

package conformance

import (
	"testing"

	"github.com/coreos/dex/storage"
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
		{"ClientConcurrentUpdate", testClientConcurrentUpdate},
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

	t.Logf("update1: %v", err1)
	t.Logf("update2: %v", err2)

	if err1 == nil && err2 == nil {
		t.Errorf("update client: concurrent updates both returned no error")
	}
}

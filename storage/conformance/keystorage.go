package conformance

import (
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"

	"github.com/dexidp/dex/storage"
)

type keySubTest struct {
	name string
	run  func(t *testing.T, s storage.KeyStorage)
}

func runKeyTests(t *testing.T, newStorage func() storage.Storage, tests []keySubTest) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := newStorage()
			defer s.Close()

			ks, ok := s.(storage.KeyStorage)
			if !ok {
				t.Fatal("storage is not KeyStorage")
				return
			}
			test.run(t, ks)
		})
	}
}

func RunKeyTests(t *testing.T, newStorage func() storage.Storage) {
	runKeyTests(t, newStorage, []keySubTest{
		{"KeysCRUD", testKeysCRUD},
	})
}

func testKeysCRUD(t *testing.T, s storage.KeyStorage) {
	updateAndCompare := func(k storage.Keys) {
		err := s.UpdateKeys(func(oldKeys storage.Keys) (storage.Keys, error) {
			return k, nil
		})
		if err != nil {
			t.Errorf("failed to update keys: %v", err)
			return
		}

		if got, err := s.GetKeys(); err != nil {
			t.Errorf("failed to get keys: %v", err)
		} else {
			got.NextRotation = got.NextRotation.UTC()
			if diff := pretty.Compare(k, got); diff != "" {
				t.Errorf("got keys did not equal expected: %s", diff)
			}
		}
	}

	// Postgres isn't as accurate with nano seconds as we'd like
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

	updateAndCompare(keys1)
	updateAndCompare(keys2)
}

func RunKeyTransactionTests(t *testing.T, newStorage func() storage.Storage) {
	runKeyTests(t, newStorage, []keySubTest{
		{"KeysConcurrentUpdate", testKeysConcurrentUpdate},
	})
}

func testKeysConcurrentUpdate(t *testing.T, s storage.KeyStorage) {
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

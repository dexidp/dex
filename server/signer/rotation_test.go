package signer

import (
	"context"
	"log/slog"
	"sort"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func signingKeyID(t *testing.T, s storage.Storage) string {
	keys, err := s.GetKeys(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	return keys.SigningKey.KeyID
}

func verificationKeyIDs(t *testing.T, s storage.Storage) (ids []string) {
	keys, err := s.GetKeys(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range keys.VerificationKeys {
		ids = append(ids, key.PublicKey.KeyID)
	}
	return ids
}

// slicesEq compare two string slices without modifying the ordering
// of the slices.
func slicesEq(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}

	cp := func(s []string) []string {
		c := make([]string, len(s))
		copy(c, s)
		return c
	}

	cp1 := cp(s1)
	cp2 := cp(s2)
	sort.Strings(cp1)
	sort.Strings(cp2)

	for i, el := range cp1 {
		if el != cp2[i] {
			return false
		}
	}
	return true
}

func TestKeyRotator(t *testing.T) {
	now := time.Now()

	delta := time.Millisecond
	rotationFrequency := time.Second * 5
	validFor := time.Second * 21

	// Only the last 5 verification keys are expected to be kept around.
	maxVerificationKeys := 5

	l := slog.New(slog.DiscardHandler)
	strategy, err := rotationStrategyForAlgorithm(rotationFrequency, validFor, jose.RS256)
	if err != nil {
		t.Fatal(err)
	}

	r := &keyRotator{
		Storage:  memory.New(l),
		strategy: strategy,
		now:      func() time.Time { return now },
		logger:   l,
	}

	var expVerificationKeys []string

	for i := 0; i < 10; i++ {
		now = now.Add(rotationFrequency + delta)
		if err := r.rotate(false); err != nil {
			t.Fatal(err)
		}

		got := verificationKeyIDs(t, r.Storage)

		if !slicesEq(expVerificationKeys, got) {
			t.Errorf("after %d rotation, expected verification keys %q, got %q", i+1, expVerificationKeys, got)
		}

		expVerificationKeys = append(expVerificationKeys, signingKeyID(t, r.Storage))
		if n := len(expVerificationKeys); n > maxVerificationKeys {
			expVerificationKeys = expVerificationKeys[n-maxVerificationKeys:]
		}
	}
}

func TestKeyRotatorAlgorithmCheck(t *testing.T) {
	now := time.Now()
	l := slog.New(slog.DiscardHandler)

	newRotator := func(s storage.Storage, alg jose.SignatureAlgorithm) *keyRotator {
		strategy, err := rotationStrategyForAlgorithm(time.Hour, time.Hour, alg)
		if err != nil {
			t.Fatal(err)
		}
		return &keyRotator{
			Storage:  s,
			strategy: strategy,
			now:      func() time.Time { return now },
			logger:   l,
		}
	}

	s := memory.New(l)
	if err := newRotator(s, jose.RS256).rotate(false); err != nil {
		t.Fatal(err)
	}
	rsaKeyID := signingKeyID(t, s)

	// A key that matches the configured algorithm is kept.
	if err := newRotator(s, jose.RS256).rotate(true); err != nil {
		t.Fatal(err)
	}
	if got := signingKeyID(t, s); got != rsaKeyID {
		t.Fatalf("expected signing key %q to be kept, got %q", rsaKeyID, got)
	}

	// A key with another algorithm is kept until it expires unless the
	// algorithm is checked.
	es := newRotator(s, jose.ES256)
	if err := es.rotate(false); err != nil {
		t.Fatal(err)
	}
	if got := signingKeyID(t, s); got != rsaKeyID {
		t.Fatalf("expected signing key %q to be kept, got %q", rsaKeyID, got)
	}

	if err := es.rotate(true); err != nil {
		t.Fatal(err)
	}
	keys, err := s.GetKeys(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	if keys.SigningKey.KeyID == rsaKeyID || keys.SigningKey.Algorithm != string(jose.ES256) {
		t.Fatalf("expected a new ES256 signing key, got %q (%s)", keys.SigningKey.KeyID, keys.SigningKey.Algorithm)
	}
	if got := verificationKeyIDs(t, s); !slicesEq(got, []string{rsaKeyID}) {
		t.Fatalf("expected the RS256 key %q to be kept for verification, got %q", rsaKeyID, got)
	}
}

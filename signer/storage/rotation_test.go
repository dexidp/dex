package storage

import (
	"os"
	"sort"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func signingKeyID(t *testing.T, s storage.KeyStorage) string {
	keys, err := s.GetKeys()
	if err != nil {
		t.Fatal(err)
	}
	return keys.SigningKey.KeyID
}

func verificationKeyIDs(t *testing.T, s storage.KeyStorage) (ids []string) {
	keys, err := s.GetKeys()
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

	l := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: &logrus.TextFormatter{DisableColors: true},
		Level:     logrus.DebugLevel,
	}

	s := &Signer{
		storage:          memory.New(l).(storage.KeyStorage),
		logger:           logger,
		now:              func() time.Time { return now },
		rotationStrategy: DefaultRotationStrategy(rotationFrequency, validFor),
	}

	var expVerificationKeys []string

	for i := 0; i < 10; i++ {
		now = now.Add(rotationFrequency + delta)
		if err := s.RotateKey(); err != nil {
			t.Fatal(err)
		}

		got := verificationKeyIDs(t, s.storage)

		if !slicesEq(expVerificationKeys, got) {
			t.Errorf("after %d rotation, expected verification keys %q, got %q", i+1, expVerificationKeys, got)
		}

		expVerificationKeys = append(expVerificationKeys, signingKeyID(t, s.storage))
		if n := len(expVerificationKeys); n > maxVerificationKeys {
			expVerificationKeys = expVerificationKeys[n-maxVerificationKeys:]
		}
	}
}

package server

import (
	"context"
	"io"
	"log/slog"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

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

	l := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))

	r := &keyRotator{
		Storage:  memory.New(l),
		strategy: defaultRotationStrategy(rotationFrequency, validFor),
		now:      func() time.Time { return now },
		logger:   l,
	}

	var expVerificationKeys []string

	for i := 0; i < 10; i++ {
		now = now.Add(rotationFrequency + delta)
		if err := r.rotate(); err != nil {
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

func TestRefreshTokenPolicy(t *testing.T) {
	lastTime := time.Now()
	l := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))

	r, err := NewRefreshTokenPolicy(l, true, "1m", "1m", "1m")
	require.NoError(t, err)

	t.Run("Allowed", func(t *testing.T) {
		r.now = func() time.Time { return lastTime }
		require.Equal(t, true, r.AllowedToReuse(lastTime))
		require.Equal(t, false, r.ExpiredBecauseUnused(lastTime))
		require.Equal(t, false, r.CompletelyExpired(lastTime))
	})

	t.Run("Expired", func(t *testing.T) {
		r.now = func() time.Time { return lastTime.Add(2 * time.Minute) }
		require.Equal(t, false, r.AllowedToReuse(lastTime))
		require.Equal(t, true, r.ExpiredBecauseUnused(lastTime))
		require.Equal(t, true, r.CompletelyExpired(lastTime))
	})
}

package tokens

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStrategy(t *testing.T) {
	lastTime := time.Now()

	t.Run("Allowed", func(t *testing.T) {
		s := NewRefreshStrategy(false, time.Minute, time.Minute, time.Minute, func() time.Time { return lastTime })
		require.True(t, s.AllowedToReuse(lastTime))
		require.False(t, s.ExpiredBecauseUnused(lastTime))
		require.False(t, s.CompletelyExpired(lastTime))
	})

	t.Run("Expired", func(t *testing.T) {
		s := NewRefreshStrategy(false, time.Minute, time.Minute, time.Minute, func() time.Time { return lastTime.Add(2 * time.Minute) })
		require.False(t, s.AllowedToReuse(lastTime))
		require.True(t, s.ExpiredBecauseUnused(lastTime))
		require.True(t, s.CompletelyExpired(lastTime))
	})

	t.Run("disabled intervals never expire", func(t *testing.T) {
		s := NewRefreshStrategy(true, 0, 0, 0, nil)
		require.True(t, s.RotationEnabled())
		require.False(t, s.CompletelyExpired(lastTime.Add(-100*time.Hour)))
		require.False(t, s.ExpiredBecauseUnused(lastTime.Add(-100*time.Hour)))
		require.False(t, s.AllowedToReuse(lastTime))
	})
}

func TestNewRefreshTokenPolicyRejectsNegativeDurations(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	for _, tc := range []struct{ valid, absolute, reuse string }{
		{valid: "-1h"},
		{absolute: "-1h"},
		{reuse: "-1s"},
	} {
		_, err := NewRefreshTokenPolicy(logger, false, tc.valid, tc.absolute, tc.reuse)
		require.Error(t, err)
		require.Contains(t, err.Error(), "must not be negative")
	}
}

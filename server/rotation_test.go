package server

import (
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestRefreshTokenPolicy(t *testing.T) {
	lastTime := time.Now()
	l := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: &logrus.TextFormatter{DisableColors: true},
		Level:     logrus.DebugLevel,
	}

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

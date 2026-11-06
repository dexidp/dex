package mfa

import (
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/require"
)

func totpOpts() totp.ValidateOpts {
	return totp.ValidateOpts{Period: totpPeriod, Skew: totpSkew, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1}
}

// TestValidateTOTPCode covers replay protection: a code is single-use per
// time-step, while a code from a later step is still accepted.
func TestValidateTOTPCode(t *testing.T) {
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "dex", AccountName: "user@example.com"})
	require.NoError(t, err)
	secret := key.Secret()

	now := time.Unix(1700000000, 0)
	code, err := totp.GenerateCodeCustom(secret, now, totpOpts())
	require.NoError(t, err)

	// First use is accepted and reports the matched counter.
	ok, counter := validateTOTPCode(secret, code, now, 0)
	require.True(t, ok)
	require.Equal(t, now.Unix()/totpPeriod, counter)

	// Replaying the same code with that counter recorded is rejected.
	ok, _ = validateTOTPCode(secret, code, now, counter)
	require.False(t, ok, "replayed code must be rejected")

	// A wrong code is rejected.
	ok, _ = validateTOTPCode(secret, "000000", now, 0)
	require.False(t, ok)

	// A code from the next step is accepted and advances the counter.
	next := now.Add(totpPeriod * time.Second)
	nextCode, err := totp.GenerateCodeCustom(secret, next, totpOpts())
	require.NoError(t, err)
	ok, nextCounter := validateTOTPCode(secret, nextCode, next, counter)
	require.True(t, ok)
	require.Greater(t, nextCounter, counter)
}

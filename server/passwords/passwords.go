// Package passwords holds the shared password-hashing policy used by both the
// gRPC API (when a password is created) and the local passwordDB connector (when
// a static password is used to log in).
package passwords

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// recCost is the recommended bcrypt cost, which balances hash strength and
	// efficiency.
	recCost = 12

	// upBoundCost is a sane upper bound on bcrypt cost determined by benchmarking:
	// high enough to ensure secure encryption, low enough to not put unnecessary
	// load on a dex server.
	upBoundCost = 16
)

// CheckCost returns an error if the bcrypt hash's cost is below the minimum or
// above the accepted upper bound.
func CheckCost(hash []byte) error {
	actual, err := bcrypt.Cost(hash)
	if err != nil {
		return fmt.Errorf("parsing bcrypt hash: %v", err)
	}
	if actual < bcrypt.DefaultCost {
		return fmt.Errorf("given hash cost = %d does not meet minimum cost requirement = %d", actual, bcrypt.DefaultCost)
	}
	if actual > upBoundCost {
		return fmt.Errorf("given hash cost = %d is above upper bound cost = %d, recommended cost = %d", actual, upBoundCost, recCost)
	}
	return nil
}

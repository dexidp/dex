package refreshtest

import (
	"fmt"

	"github.com/coreos/dex/db"
	"github.com/coreos/dex/refresh"
)

// NewTestRefreshTokenRepo returns a test repo whose tokens monotonically increase.
// The tokens are in the form { refresh-1, refresh-2 ... refresh-n}.
func NewTestRefreshTokenRepo() refresh.RefreshTokenRepo {
	var tokenIdx int
	tokenGenerator := func() ([]byte, error) {
		tokenIdx++
		return []byte(fmt.Sprintf("refresh-%d", tokenIdx)), nil
	}
	return db.NewRefreshTokenRepoWithGenerator(db.NewMemDB(), tokenGenerator)
}

package refreshtest

import (
	"fmt"

	"github.com/coreos/dex/refresh"
)

// NewTestRefreshTokenRepo returns a test repo whose tokens monotonically increase.
// The tokens are in the form { refresh-1, refresh-2 ... refresh-n}.
func NewTestRefreshTokenRepo() (refresh.RefreshTokenRepo, error) {
	var tokenIdx int
	tokenGenerator := func() (string, error) {
		tokenIdx++
		return fmt.Sprintf("refresh-%d", tokenIdx), nil
	}
	return refresh.NewRefreshTokenRepoWithTokenGenerator(tokenGenerator), nil
}

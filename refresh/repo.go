package refresh

import (
	"crypto/rand"
	"errors"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/scope"
)

const (
	DefaultRefreshTokenPayloadLength = 64
	TokenDelimer                     = "/"
)

var (
	ErrorInvalidUserID   = errors.New("invalid user ID")
	ErrorInvalidClientID = errors.New("invalid client ID")

	ErrorInvalidToken = errors.New("invalid token")
)

type RefreshTokenGenerator func() ([]byte, error)

func (g RefreshTokenGenerator) Generate() ([]byte, error) {
	return g()
}

func DefaultRefreshTokenGenerator() ([]byte, error) {
	// TODO(yifan) Remove this duplicated token generate function.
	b := make([]byte, DefaultRefreshTokenPayloadLength)
	n, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	if n != DefaultRefreshTokenPayloadLength {
		return nil, errors.New("unable to read enough random bytes")
	}
	return b, nil
}

type RefreshTokenRepo interface {
	// Create generates and returns a new refresh token for the given client-user pair.
	// The scopes will be stored with the refresh token, and used to verify
	// against future OIDC refresh requests' scopes.
	// On success the token will be returned.
	Create(userID, clientID, connectorID string, scope []string) (string, error)

	// Verify verifies that a token belongs to the client.
	// It returns the user ID to which the token belongs, and the scopes stored
	// with token.
	Verify(clientID, token string) (userID, connectorID string, scope scope.Scopes, err error)

	// Revoke deletes the refresh token if the token belongs to the given userID.
	Revoke(userID, token string) error

	// Revoke old refresh token and generates a new one
	RenewRefreshToken(clientID, userID, oldToken string) (newRefreshToken string, err error)

	// RevokeTokensForClient revokes all tokens issued for the userID for the provided client.
	RevokeTokensForClient(userID, clientID string) error

	// ClientsWithRefreshTokens returns a list of all clients the user has an outstanding client with.
	ClientsWithRefreshTokens(userID string) ([]client.Client, error)
}

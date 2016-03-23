package refresh

import (
	"crypto/rand"
	"errors"

	"github.com/coreos/go-oidc/oidc"
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
	// On success the token will be return.
	Create(userID, clientID string) (string, error)

	// Verify verifies that a token belongs to the client, and returns the corresponding user ID.
	// Note that this assumes the client validation is currently done in the application layer,
	Verify(clientID, token string) (string, error)

	// Revoke deletes the refresh token if the token belongs to the given userID.
	Revoke(userID, token string) error

	// RevokeTokensForClient revokes all tokens issued for the userID for the provided client.
	RevokeTokensForClient(userID, clientID string) error

	// ClientsWithRefreshTokens returns a list of all clients the user has an outstanding client with.
	ClientsWithRefreshTokens(userID string) ([]oidc.ClientIdentity, error)
}

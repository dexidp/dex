package refresh

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"strconv"
	"strings"
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
	} else if n != DefaultRefreshTokenPayloadLength {
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
}

type refreshToken struct {
	payload  []byte
	userID   string
	clientID string
}

type memRefreshTokenRepo struct {
	store          map[int]refreshToken
	tokenGenerator RefreshTokenGenerator
}

// buildToken combines the token ID and token payload to create a new token.
func buildToken(tokenID int, tokenPayload []byte) string {
	return fmt.Sprintf("%d%s%s", tokenID, TokenDelimer, tokenPayload)
}

// parseToken parses a token and returns the token ID and token payload.
func parseToken(token string) (int, []byte, error) {
	parts := strings.SplitN(token, TokenDelimer, 2)
	if len(parts) != 2 {
		return -1, nil, ErrorInvalidToken
	}
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		return -1, nil, ErrorInvalidToken
	}
	return id, []byte(parts[1]), nil
}

// NewRefreshTokenRepo returns an in-memory RefreshTokenRepo useful for development.
func NewRefreshTokenRepo() RefreshTokenRepo {
	return NewRefreshTokenRepoWithTokenGenerator(DefaultRefreshTokenGenerator)
}

func NewRefreshTokenRepoWithTokenGenerator(tokenGenerator RefreshTokenGenerator) RefreshTokenRepo {
	repo := &memRefreshTokenRepo{}
	repo.store = make(map[int]refreshToken)
	repo.tokenGenerator = tokenGenerator
	return repo
}

func (r *memRefreshTokenRepo) Create(userID, clientID string) (string, error) {
	// Validate userID.
	if userID == "" {
		return "", ErrorInvalidUserID
	}

	// Validate clientID.
	if clientID == "" {
		return "", ErrorInvalidClientID
	}

	// Generate and store token.
	tokenPayload, err := r.tokenGenerator.Generate()
	if err != nil {
		return "", err
	}

	tokenID := len(r.store) // Should only be used in single threaded tests.

	// No limits on the number of tokens per user/client for this in-memory repo.
	r.store[tokenID] = refreshToken{
		payload:  tokenPayload,
		userID:   userID,
		clientID: clientID,
	}
	return buildToken(tokenID, tokenPayload), nil
}

func (r *memRefreshTokenRepo) Verify(clientID, token string) (string, error) {
	tokenID, tokenPayload, err := parseToken(token)
	if err != nil {
		return "", err
	}

	record, ok := r.store[tokenID]
	if !ok {
		return "", ErrorInvalidToken
	}

	if !bytes.Equal(record.payload, tokenPayload) {
		return "", ErrorInvalidToken
	}

	if record.clientID != clientID {
		return "", ErrorInvalidClientID
	}

	return record.userID, nil
}

func (r *memRefreshTokenRepo) Revoke(userID, token string) error {
	tokenID, tokenPayload, err := parseToken(token)
	if err != nil {
		return err
	}

	record, ok := r.store[tokenID]
	if !ok {
		return ErrorInvalidToken
	}

	if !bytes.Equal(record.payload, tokenPayload) {
		return ErrorInvalidToken
	}

	if record.userID != userID {
		return ErrorInvalidUserID
	}

	delete(r.store, tokenID)
	return nil
}

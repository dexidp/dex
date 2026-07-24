package tokens

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

// Errors a refresh-token lookup can reject a token with. Callers map them to
// their own response: the refresh grant to OAuth2 errors, introspection to an
// inactive-token result.
var (
	ErrRefreshTokenInvalid              = errors.New("refresh token is invalid or has already been claimed")
	ErrRefreshTokenExpired              = errors.New("refresh token expired")
	ErrRefreshTokenClaimedByOtherClient = errors.New("refresh token claimed by another client")
)

// LookupRefreshToken validates a refresh token against storage: it exists,
// belongs to clientID when one is given, has not been claimed twice, and has not
// expired per the expiry policy of the connector it was issued through. It
// returns one of the Err* sentinels for a rejected token, or a wrapped storage
// error for infrastructure failures. Both the refresh grant and token
// introspection use it, so the validation lives in one place.
func LookupRefreshToken(ctx context.Context, s storage.Storage, expiry *Expiry, logger *slog.Logger, clientID *string, token *internal.RefreshToken) (*storage.RefreshToken, error) {
	refresh, err := s.GetRefresh(ctx, token.RefreshId)
	if err != nil {
		if err != storage.ErrNotFound {
			logger.ErrorContext(ctx, "failed to get refresh token", "err", err)
			return nil, fmt.Errorf("get refresh token: %w", err)
		}
		return nil, ErrRefreshTokenInvalid
	}
	strategy := expiry.RefreshStrategy(refresh.ConnectorID)

	// Only check the client when one was provided (introspection does not).
	// https://datatracker.ietf.org/doc/html/rfc6749#section-5.2
	if clientID != nil && refresh.ClientID != *clientID {
		logger.ErrorContext(ctx, "trying to claim token for different client", "client_id", *clientID, "refresh_client_id", refresh.ClientID)
		return nil, ErrRefreshTokenClaimedByOtherClient
	}

	// Constant-time comparison of the token secret: introspection reaches this
	// path unauthenticated, so a byte-by-byte '!=' would leak the secret via
	// timing. Matches the client-secret/nonce comparisons elsewhere in the server.
	if subtle.ConstantTimeCompare([]byte(refresh.Token), []byte(token.Token)) != 1 {
		switch {
		case !strategy.AllowedToReuse(refresh.LastUsed):
			fallthrough
		case subtle.ConstantTimeCompare([]byte(refresh.ObsoleteToken), []byte(token.Token)) != 1:
			fallthrough
		case refresh.ObsoleteToken == "":
			logger.ErrorContext(ctx, "refresh token claimed twice", "token_id", refresh.ID)
			return nil, ErrRefreshTokenInvalid
		}
	}

	if strategy.CompletelyExpired(refresh.CreatedAt) {
		logger.ErrorContext(ctx, "refresh token expired", "token_id", refresh.ID)
		return nil, ErrRefreshTokenExpired
	}
	if strategy.ExpiredBecauseUnused(refresh.LastUsed) {
		logger.ErrorContext(ctx, "refresh token expired due to inactivity", "token_id", refresh.ID)
		return nil, ErrRefreshTokenExpired
	}

	return &refresh, nil
}

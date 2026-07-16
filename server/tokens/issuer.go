package tokens

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/google/uuid"

	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/storage"
)

// Issuer turns an Authorization into signed tokens. It mints access and ID
// tokens (SignAccessToken/SignIDToken) and owns a RefreshStore for refresh-token
// persistence. The low-level cryptographic signer is an injected dependency.
type Issuer struct {
	storage          storage.Storage
	signer           signer.Signer
	issuerURL        url.URL
	idTokensValidFor time.Duration
	now              func() time.Time
	logger           *slog.Logger

	// Refresh persists and rotates refresh tokens.
	Refresh *RefreshStore
}

// NewIssuer wires an issuer from the shared dependencies.
func NewIssuer(storage storage.Storage, sig signer.Signer, issuerURL url.URL, idTokensValidFor time.Duration, now func() time.Time, logger *slog.Logger) *Issuer {
	return &Issuer{
		storage:          storage,
		signer:           sig,
		issuerURL:        issuerURL,
		idTokensValidFor: idTokensValidFor,
		now:              now,
		logger:           logger,
		Refresh:          NewRefreshStore(storage, now, logger),
	}
}

// Issue mints the standard token set for the authorization: an access token,
// an ID token when the openid scope was requested, and a refresh token when
// withRefresh is true. Whether a refresh token is wanted (connector support,
// grant-type policy, offline_access scope) is a grant decision, so the caller
// passes it in. This is the single mint every standard OAuth2 grant uses.
func (i *Issuer) Issue(ctx context.Context, auth Authorization, withRefresh bool) (TokenSet, error) {
	accessToken, expiry, err := i.SignAccessToken(ctx, auth)
	if err != nil {
		return TokenSet{}, fmt.Errorf("create access token: %w", err)
	}

	ts := TokenSet{AccessToken: accessToken, Expiry: expiry}

	if HasOpenID(auth.Scopes) {
		ts.IDToken, ts.Expiry, err = i.SignIDToken(ctx, auth, accessToken, "")
		if err != nil {
			return TokenSet{}, fmt.Errorf("create id token: %w", err)
		}
	}

	if withRefresh {
		ts.RefreshToken, err = i.Refresh.Create(ctx, auth)
		if err != nil {
			return TokenSet{}, fmt.Errorf("create refresh token: %w", err)
		}
	}
	return ts, nil
}

// IssueResponse mints the standard token set with Issue and renders it as an
// OAuth2 token response, so a grant handler need not carry a clock of its own.
func (i *Issuer) IssueResponse(ctx context.Context, auth Authorization, withRefresh bool) (Response, error) {
	ts, err := i.Issue(ctx, auth, withRefresh)
	if err != nil {
		return Response{}, err
	}
	return ts.Response(i.now()), nil
}

// SignAccessToken mints an opaque-looking JWT access token. Dex's access token is
// an ID token bound to a random value, so each access token is unique.
func (i *Issuer) SignAccessToken(ctx context.Context, auth Authorization) (string, time.Time, error) {
	return i.SignIDToken(ctx, auth, storage.NewID(), "")
}

// SignIDToken mints an ID token. accessToken/code, when set, contribute
// at_hash/c_hash.
func (i *Issuer) SignIDToken(ctx context.Context, auth Authorization, accessToken, code string) (string, time.Time, error) {
	issuedAt := i.now()
	expiry := issuedAt.Add(i.idTokensValidFor)

	subjectString, err := GenSubject(auth.Claims.UserID, auth.ConnectorID)
	if err != nil {
		i.logger.ErrorContext(ctx, "failed to marshal offline session ID", "err", err)
		return "", expiry, fmt.Errorf("failed to marshal offline session ID: %v", err)
	}

	tok := IDTokenClaims{
		Issuer:   i.issuerURL.String(),
		Subject:  subjectString,
		Nonce:    auth.Nonce,
		Expiry:   expiry.Unix(),
		IssuedAt: issuedAt.Unix(),
		JWTID:    uuid.New().String(),
	}

	if !auth.AuthTime.IsZero() {
		tok.AuthTime = auth.AuthTime.Unix()
	}

	signingAlg, err := i.signer.Algorithm(ctx)
	if err != nil {
		i.logger.ErrorContext(ctx, "failed to get signing algorithm", "err", err)
		return "", expiry, fmt.Errorf("failed to get signing algorithm: %v", err)
	}

	if accessToken != "" {
		atHash, err := AccessTokenHash(signingAlg, accessToken)
		if err != nil {
			i.logger.ErrorContext(ctx, "error computing at_hash", "err", err)
			return "", expiry, fmt.Errorf("error computing at_hash: %v", err)
		}
		tok.AccessTokenHash = atHash
	}

	if code != "" {
		cHash, err := AccessTokenHash(signingAlg, code)
		if err != nil {
			i.logger.ErrorContext(ctx, "error computing c_hash", "err", err)
			return "", expiry, fmt.Errorf("error computing c_hash: %v", err)
		}
		tok.CodeHash = cHash
	}

	clientID := auth.Client.ID
	for _, scope := range auth.Scopes {
		switch {
		case scope == ScopeEmail:
			tok.Email = auth.Claims.Email
			tok.EmailVerified = &auth.Claims.EmailVerified
		case scope == ScopeGroups:
			tok.Groups = auth.Claims.Groups
		case scope == ScopeProfile:
			tok.Name = auth.Claims.Username
			tok.PreferredUsername = auth.Claims.PreferredUsername
		case scope == ScopeFederatedID:
			tok.FederatedIDClaims = &FederatedIDClaims{
				ConnectorID: auth.ConnectorID,
				UserID:      auth.Claims.UserID,
			}
		default:
			peerID, ok := ParseCrossClientScope(scope)
			if !ok {
				continue
			}
			isTrusted, err := i.crossClientTrusted(ctx, clientID, peerID)
			if err != nil {
				return "", expiry, err
			}
			if !isTrusted {
				return "", expiry, fmt.Errorf("peer (%s) does not trust client", peerID)
			}
		}
	}

	tok.Audience = GetAudience(clientID, auth.Scopes)
	if len(tok.Audience) > 1 {
		tok.AuthorizingParty = clientID
	}

	payload, err := json.Marshal(tok)
	if err != nil {
		return "", expiry, fmt.Errorf("could not serialize claims: %v", err)
	}

	idToken, err := i.signer.Sign(ctx, payload)
	if err != nil {
		return "", expiry, fmt.Errorf("failed to sign payload: %v", err)
	}
	return idToken, expiry, nil
}

// crossClientTrusted reports whether peerID's client trusts clientID as a peer.
func (i *Issuer) crossClientTrusted(ctx context.Context, clientID, peerID string) (bool, error) {
	if peerID == clientID {
		return true, nil
	}
	peer, err := i.storage.GetClient(ctx, peerID)
	if err != nil {
		if err != storage.ErrNotFound {
			i.logger.ErrorContext(ctx, "failed to get client", "err", err)
			return false, err
		}
		return false, nil
	}
	for _, id := range peer.TrustedPeers {
		if id == clientID {
			return true, nil
		}
	}
	return false, nil
}

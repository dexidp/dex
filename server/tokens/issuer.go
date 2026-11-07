package tokens

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
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

	// Sessions resolves whether the browser session a token was issued under
	// is still alive. Nil when sessions are disabled.
	Sessions SessionChecker
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
// withRefresh is true. code is the authorization code bound into the ID token's
// c_hash, empty for grants without one. Whether a refresh token is wanted
// (connector support, grant-type policy, offline_access scope) is a grant
// decision, so the caller passes it in. This is the single mint every standard
// OAuth2 grant uses.
func (i *Issuer) Issue(ctx context.Context, auth Authorization, code string, withRefresh bool) (TokenSet, error) {
	accessToken, expiry, err := i.SignAccessToken(ctx, auth)
	if err != nil {
		return TokenSet{}, fmt.Errorf("create access token: %w", err)
	}

	ts := TokenSet{AccessToken: accessToken, Expiry: expiry}

	if HasOpenID(auth.Scopes) {
		ts.IDToken, ts.Expiry, err = i.SignIDToken(ctx, auth, accessToken, code)
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
func (i *Issuer) IssueResponse(ctx context.Context, auth Authorization, code string, withRefresh bool) (Response, error) {
	ts, err := i.Issue(ctx, auth, code, withRefresh)
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
	if !auth.ExpiresAt.IsZero() && auth.ExpiresAt.Before(expiry) {
		expiry = auth.ExpiresAt
	}

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

	tok.SessionID = auth.SessionID
	tok.Act = auth.Act
	tok.Scope = auth.ScopeClaim

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

	tok.Audience = withAudiences(clientID, auth.Audience, GetAudience(clientID, auth.Scopes))
	if auth.Act != nil {
		tok.ClientID = clientID
	}
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

// withAudiences puts the extra audiences an exchange named in front of aud,
// leaving out the client's own ID and duplicates, so the client stays last as
// GetAudience places it.
func withAudiences(clientID string, extra []string, aud Audience) Audience {
	if len(extra) == 0 {
		return aud
	}
	var out Audience
	for _, a := range extra {
		if a != clientID && !out.contains(a) {
			out = append(out, a)
		}
	}
	for _, a := range aud {
		if !out.contains(a) {
			out = append(out, a)
		}
	}
	return out
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

// ErrInvalidToken reports a token this issuer did not sign, that has expired,
// or whose issuing client cannot be resolved.
var ErrInvalidToken = errors.New("invalid token")

// ErrSessionEnded reports a token bound to a browser session that has ended.
var ErrSessionEnded = errors.New("token session has ended")

// SessionChecker reports whether a browser session is still alive.
type SessionChecker interface {
	Alive(ctx context.Context, sessionID string) bool
}

// VerifiedToken is a token this issuer signed, verified and still live.
type VerifiedToken struct {
	Token  *oidc.IDToken
	Claims map[string]any
	// Client is the client the token was issued to, resolved from its aud and
	// azp, not the client presenting it.
	Client storage.Client
	// SessionID is the token's sid claim, empty when it carries none.
	SessionID string
}

// VerifyToken verifies a token this issuer signed (an ID or access token),
// resolves the client it was issued to, and applies that client's session
// binding: a token carrying sid, issued to a client whose tokens end with the
// session (RefreshTokenLifetime session), is refused once that session ends
// (RFC 7662 §4 reports such a token inactive). It returns ErrInvalidToken or
// ErrSessionEnded for those refusals and other errors for storage failures.
func (i *Issuer) VerifyToken(ctx context.Context, raw string) (*VerifiedToken, error) {
	verifier := oidc.NewVerifier(i.issuerURL.String(), &signer.KeySet{Signer: i.signer}, &oidc.Config{SkipClientIDCheck: true, Now: i.now})
	token, err := verifier.Verify(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	var claims map[string]any
	if err := token.Claims(&claims); err != nil {
		return nil, fmt.Errorf("%w: decode claims: %v", ErrInvalidToken, err)
	}
	azp, _ := claims["azp"].(string)
	clientID, err := GetClientID(token.Audience, azp)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	client, err := i.storage.GetClient(ctx, clientID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, fmt.Errorf("%w: issuing client %q not found", ErrInvalidToken, clientID)
		}
		return nil, fmt.Errorf("get client %q: %w", clientID, err)
	}
	sid, _ := claims["sid"].(string)
	if sid != "" && client.RefreshBoundToSession() && (i.Sessions == nil || !i.Sessions.Alive(ctx, sid)) {
		return nil, ErrSessionEnded
	}
	return &VerifiedToken{Token: token, Claims: claims, Client: client, SessionID: sid}, nil
}

// Now is the issuer's clock, so callers report expiries on the same clock the
// tokens were signed with.
func (i *Issuer) Now() time.Time {
	return i.now()
}

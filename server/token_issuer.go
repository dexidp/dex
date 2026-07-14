package server

// token_issuer.go is the token-issuance core. Every grant reduces to producing an
// Authorization (an authenticated subject's permission for a client); the issuer
// turns that uniformly into a TokenSet. Signing (tokenSigner) and refresh-token
// persistence (refreshTokens) are the two collaborators; writing the HTTP response
// is a separate transport concern (writeTokenResponse).

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/storage"
)

// Authorization is the result of any grant: an authenticated subject and the
// permission it grants a client. The issuer turns an Authorization into a TokenSet.
type Authorization struct {
	Client        storage.Client
	Claims        storage.Claims
	Scopes        []string
	ConnectorID   string
	Nonce         string
	AuthTime      time.Time
	ConnectorData []byte
}

// TokenSet is what the /token endpoint returns for an Authorization.
type TokenSet struct {
	AccessToken  string
	IDToken      string
	RefreshToken string
	Expiry       time.Time
}

// tokenIssuer turns an Authorization into a TokenSet. It composes the signer and
// the refresh-token store; it holds no HTTP or grant-specific logic.
type tokenIssuer struct {
	signer  *tokenSigner
	refresh *refreshTokens
}

func newTokenIssuer(s *Server) *tokenIssuer {
	return &tokenIssuer{
		signer: &tokenSigner{
			storage:          s.storage,
			signer:           s.signer,
			issuerURL:        s.issuerURL,
			idTokensValidFor: s.idTokensValidFor,
			now:              s.now,
			logger:           s.logger,
		},
		refresh: &refreshTokens{
			storage: s.storage,
			now:     s.now,
			logger:  s.logger,
		},
	}
}

// Issue mints an access and ID token for the authorization, and a refresh token
// when withRefresh is true. Whether a refresh token is wanted (connector support,
// grant-type policy, offline_access scope) is a grant decision, so the caller
// passes it in.
func (i *tokenIssuer) Issue(ctx context.Context, auth Authorization, withRefresh bool) (TokenSet, error) {
	accessToken, _, err := i.signer.signAccessToken(ctx, auth)
	if err != nil {
		return TokenSet{}, fmt.Errorf("create access token: %w", err)
	}

	idToken, expiry, err := i.signer.signIDToken(ctx, auth, accessToken, "")
	if err != nil {
		return TokenSet{}, fmt.Errorf("create id token: %w", err)
	}

	ts := TokenSet{AccessToken: accessToken, IDToken: idToken, Expiry: expiry}

	if withRefresh {
		ts.RefreshToken, err = i.refresh.create(ctx, auth)
		if err != nil {
			return TokenSet{}, fmt.Errorf("create refresh token: %w", err)
		}
	}
	return ts, nil
}

// tokenSigner mints signed access and ID tokens. It owns the OIDC claim assembly,
// including cross-client audience resolution.
type tokenSigner struct {
	storage          storage.Storage
	signer           signer.Signer
	issuerURL        url.URL
	idTokensValidFor time.Duration
	now              func() time.Time
	logger           *slog.Logger
}

// signAccessToken mints an opaque-looking JWT access token. Dex's access token is an
// ID token bound to a random value, so each access token is unique.
func (t *tokenSigner) signAccessToken(ctx context.Context, auth Authorization) (string, time.Time, error) {
	return t.signIDToken(ctx, auth, storage.NewID(), "")
}

// signIDToken mints an ID token. accessToken/code, when set, contribute at_hash/c_hash.
func (t *tokenSigner) signIDToken(ctx context.Context, auth Authorization, accessToken, code string) (string, time.Time, error) {
	issuedAt := t.now()
	expiry := issuedAt.Add(t.idTokensValidFor)

	subjectString, err := genSubject(auth.Claims.UserID, auth.ConnectorID)
	if err != nil {
		t.logger.ErrorContext(ctx, "failed to marshal offline session ID", "err", err)
		return "", expiry, fmt.Errorf("failed to marshal offline session ID: %v", err)
	}

	tok := idTokenClaims{
		Issuer:   t.issuerURL.String(),
		Subject:  subjectString,
		Nonce:    auth.Nonce,
		Expiry:   expiry.Unix(),
		IssuedAt: issuedAt.Unix(),
		JWTID:    uuid.New().String(),
	}

	if !auth.AuthTime.IsZero() {
		tok.AuthTime = auth.AuthTime.Unix()
	}

	signingAlg, err := t.signer.Algorithm(ctx)
	if err != nil {
		t.logger.ErrorContext(ctx, "failed to get signing algorithm", "err", err)
		return "", expiry, fmt.Errorf("failed to get signing algorithm: %v", err)
	}

	if accessToken != "" {
		atHash, err := accessTokenHash(signingAlg, accessToken)
		if err != nil {
			t.logger.ErrorContext(ctx, "error computing at_hash", "err", err)
			return "", expiry, fmt.Errorf("error computing at_hash: %v", err)
		}
		tok.AccessTokenHash = atHash
	}

	if code != "" {
		cHash, err := accessTokenHash(signingAlg, code)
		if err != nil {
			t.logger.ErrorContext(ctx, "error computing c_hash", "err", err)
			return "", expiry, fmt.Errorf("error computing c_hash: %v", err)
		}
		tok.CodeHash = cHash
	}

	clientID := auth.Client.ID
	for _, scope := range auth.Scopes {
		switch {
		case scope == scopeEmail:
			tok.Email = auth.Claims.Email
			tok.EmailVerified = &auth.Claims.EmailVerified
		case scope == scopeGroups:
			tok.Groups = auth.Claims.Groups
		case scope == scopeProfile:
			tok.Name = auth.Claims.Username
			tok.PreferredUsername = auth.Claims.PreferredUsername
		case scope == scopeFederatedID:
			tok.FederatedIDClaims = &federatedIDClaims{
				ConnectorID: auth.ConnectorID,
				UserID:      auth.Claims.UserID,
			}
		default:
			peerID, ok := parseCrossClientScope(scope)
			if !ok {
				continue
			}
			isTrusted, err := t.crossClientTrusted(ctx, clientID, peerID)
			if err != nil {
				return "", expiry, err
			}
			if !isTrusted {
				return "", expiry, fmt.Errorf("peer (%s) does not trust client", peerID)
			}
		}
	}

	tok.Audience = getAudience(clientID, auth.Scopes)
	if len(tok.Audience) > 1 {
		tok.AuthorizingParty = clientID
	}

	payload, err := json.Marshal(tok)
	if err != nil {
		return "", expiry, fmt.Errorf("could not serialize claims: %v", err)
	}

	idToken, err := t.signer.Sign(ctx, payload)
	if err != nil {
		return "", expiry, fmt.Errorf("failed to sign payload: %v", err)
	}
	return idToken, expiry, nil
}

// crossClientTrusted reports whether peerID's client trusts clientID as a peer.
func (t *tokenSigner) crossClientTrusted(ctx context.Context, clientID, peerID string) (bool, error) {
	if peerID == clientID {
		return true, nil
	}
	peer, err := t.storage.GetClient(ctx, peerID)
	if err != nil {
		if err != storage.ErrNotFound {
			t.logger.ErrorContext(ctx, "failed to get client", "err", err)
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

// refreshTokens creates and rotates refresh tokens, keeping the offline-session
// references in sync. It is the single owner of refresh-token persistence.
type refreshTokens struct {
	storage storage.Storage
	now     func() time.Time
	logger  *slog.Logger
}

// create issues a new refresh token for the authorization and records it in the
// user's offline session. On any offline-session failure the refresh token is
// rolled back so no orphaned token is left behind.
func (rt *refreshTokens) create(ctx context.Context, auth Authorization) (string, error) {
	refresh := storage.RefreshToken{
		ID:            storage.NewID(),
		Token:         storage.NewID(),
		ClientID:      auth.Client.ID,
		ConnectorID:   auth.ConnectorID,
		Scopes:        auth.Scopes,
		Claims:        auth.Claims,
		Nonce:         auth.Nonce,
		ConnectorData: auth.ConnectorData,
		CreatedAt:     rt.now(),
		LastUsed:      rt.now(),
	}

	rawToken, err := internal.Marshal(&internal.RefreshToken{RefreshId: refresh.ID, Token: refresh.Token})
	if err != nil {
		return "", fmt.Errorf("marshal refresh token: %w", err)
	}

	if err := rt.storage.CreateRefresh(ctx, refresh); err != nil {
		return "", fmt.Errorf("create refresh token: %w", err)
	}

	// Roll back the just-created refresh token if wiring it into the offline
	// session fails, so we never leave an orphaned token.
	rollback := func() {
		if err := rt.storage.DeleteRefresh(ctx, refresh.ID); err != nil && err != storage.ErrNotFound {
			rt.logger.ErrorContext(ctx, "failed to roll back refresh token", "err", err)
		}
	}

	tokenRef := storage.RefreshTokenRef{
		ID:        refresh.ID,
		ClientID:  refresh.ClientID,
		CreatedAt: refresh.CreatedAt,
		LastUsed:  refresh.LastUsed,
	}

	session, err := rt.storage.GetOfflineSessions(ctx, refresh.Claims.UserID, refresh.ConnectorID)
	switch {
	case err != nil && err != storage.ErrNotFound:
		rollback()
		return "", fmt.Errorf("get offline session: %w", err)
	case err != nil: // ErrNotFound: create a fresh offline session.
		offlineSessions := storage.OfflineSessions{
			UserID:        refresh.Claims.UserID,
			ConnID:        refresh.ConnectorID,
			Refresh:       map[string]*storage.RefreshTokenRef{tokenRef.ClientID: &tokenRef},
			ConnectorData: auth.ConnectorData,
		}
		if err := rt.storage.CreateOfflineSessions(ctx, offlineSessions); err != nil {
			rollback()
			return "", fmt.Errorf("create offline session: %w", err)
		}
	default:
		// Replace any existing refresh token for this client.
		if oldRef, ok := session.Refresh[tokenRef.ClientID]; ok {
			if err := rt.storage.DeleteRefresh(ctx, oldRef.ID); err != nil && err != storage.ErrNotFound {
				rollback()
				return "", fmt.Errorf("delete previous refresh token: %w", err)
			}
		}
		if err := rt.storage.UpdateOfflineSessions(ctx, session.UserID, session.ConnID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
			old.Refresh[tokenRef.ClientID] = &tokenRef
			if len(auth.ConnectorData) > 0 {
				old.ConnectorData = auth.ConnectorData
			}
			return old, nil
		}); err != nil {
			rollback()
			return "", fmt.Errorf("update offline session: %w", err)
		}
	}

	return rawToken, nil
}

// writeTokenResponse writes a TokenSet as an OAuth2 token response.
func writeTokenResponse(w http.ResponseWriter, ts TokenSet, now time.Time) error {
	resp := accessTokenResponse{
		AccessToken:  ts.AccessToken,
		TokenType:    "bearer",
		ExpiresIn:    int(ts.Expiry.Sub(now).Seconds()),
		RefreshToken: ts.RefreshToken,
		IDToken:      ts.IDToken,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal token response: %w", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	// Token response must include cache headers, https://tools.ietf.org/html/rfc6749#section-5.1
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Write(data)
	return nil
}

package introspection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// Introspection contains an access token's session data as specified by
// [IETF RFC 7662](https://tools.ietf.org/html/rfc7662)
type Introspection struct {
	// Boolean indicator of whether or not the presented token
	// is currently active.  The specifics of a token's "active" state
	// will vary depending on the implementation of the authorization
	// server and the information it keeps about its tokens, but a "true"
	// value return for the "active" property will generally indicate
	// that a given token has been issued by this authorization server,
	// has not been revoked by the resource owner, and is within its
	// given time window of validity (e.g., after its issuance time and
	// before its expiration time).
	Active bool `json:"active"`

	// JSON string containing a space-separated list of
	// scopes associated with this token.
	Scope string `json:"scope,omitempty"`

	// Client identifier for the OAuth 2.0 client that
	// requested this token.
	ClientID string `json:"client_id"`

	// Subject of the token, as defined in JWT [RFC7519].
	// Usually a machine-readable identifier of the resource owner who
	// authorized this token.
	Subject string `json:"sub"`

	// Integer timestamp, measured in the number of seconds
	// since January 1 1970 UTC, indicating when this token will expire.
	Expiry int64 `json:"exp"`

	// Integer timestamp, measured in the number of seconds
	// since January 1 1970 UTC, indicating when this token was
	// originally issued.
	IssuedAt int64 `json:"iat"`

	// Integer timestamp, measured in the number of seconds
	// since January 1 1970 UTC, indicating when this token is not to be
	// used before.
	NotBefore int64 `json:"nbf"`

	// Human-readable identifier for the resource owner who
	// authorized this token.
	Username string `json:"username,omitempty"`

	// Service-specific string identifier or list of string
	// identifiers representing the intended audience for this token, as
	// defined in JWT
	Audience tokens.Audience `json:"aud"`

	// String representing the issuer of this token, as
	// defined in JWT
	Issuer string `json:"iss"`

	// String identifier for the token, as defined in JWT [RFC7519].
	JwtTokenID string `json:"jti,omitempty"`

	// TokenType is the introspected token's type, typically `bearer`.
	TokenType string `json:"token_type"`

	// TokenUse is the introspected token's use, for example `access_token` or `refresh_token`.
	TokenUse string `json:"token_use"`

	// Extra is arbitrary data set from the token claims.
	Extra IntrospectionExtra `json:"ext,omitempty"`
}

type IntrospectionExtra struct {
	AuthorizingParty string `json:"azp,omitempty"`

	Email         string `json:"email,omitempty"`
	EmailVerified *bool  `json:"email_verified,omitempty"`

	Groups []string `json:"groups,omitempty"`

	Name              string `json:"name,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`

	FederatedIDClaims *tokens.FederatedIDClaims `json:"federated_claims,omitempty"`
}

type TokenTypeEnum int

const (
	AccessToken TokenTypeEnum = iota
	RefreshToken
)

func (t TokenTypeEnum) String() string {
	switch t {
	case AccessToken:
		return "access_token"
	case RefreshToken:
		return "refresh_token"
	default:
		return fmt.Sprintf("TokenTypeEnum(%d)", t)
	}
}

type introspectionError struct {
	typ  string
	code int
	desc string
}

func (e *introspectionError) Error() string {
	return fmt.Sprintf("introspection error: status %d, %q %s", e.code, e.typ, e.desc)
}

func (e *introspectionError) Is(tgt error) bool {
	target, ok := tgt.(*introspectionError)
	if !ok {
		return false
	}

	return e.typ == target.typ &&
		e.code == target.code &&
		e.desc == target.desc
}

func newIntrospectInactiveTokenError() *introspectionError {
	return &introspectionError{typ: oauth2.InactiveToken, desc: "", code: http.StatusUnauthorized}
}

func newIntrospectInternalServerError() *introspectionError {
	return &introspectionError{typ: oauth2.ServerError, desc: "", code: http.StatusInternalServerError}
}

func newIntrospectBadRequestError(desc string) *introspectionError {
	return &introspectionError{typ: oauth2.InvalidRequest, desc: desc, code: http.StatusBadRequest}
}

// Handler serves the OAuth2 token introspection endpoint. It validates refresh
// tokens with tokens.LookupRefreshToken, the same lookup the refresh grant uses.
type Handler struct {
	Issuer        string
	Signer        signer.Signer
	Storage       storage.Storage
	Logger        *slog.Logger
	RefreshPolicy *tokens.RefreshStrategy
}

// Mount registers the introspection route.
func (h *Handler) Mount(m router.Mux) {
	m.HandleCORS("/token/introspect", h.handle, http.MethodPost)
}

func (h *Handler) guessTokenType(ctx context.Context, token string) (TokenTypeEnum, error) {
	// We skip every checks, we only want to know if it's a valid JWT
	verifierConfig := oidc.Config{
		SkipClientIDCheck: true,
		SkipExpiryCheck:   true,
		SkipIssuerCheck:   true,

		// We skip signature checks to avoid database calls;
		InsecureSkipSignatureCheck: true,
	}

	verifier := oidc.NewVerifier(h.Issuer, nil, &verifierConfig)
	if _, err := verifier.Verify(ctx, token); err != nil {
		// If it's not an access token, let's assume it's a refresh token;
		return RefreshToken, nil
	}

	// If it's a valid JWT, it's an access token.
	return AccessToken, nil
}

func (h *Handler) getTokenFromRequest(r *http.Request) (string, TokenTypeEnum, error) {
	if err := r.ParseForm(); err != nil {
		return "", 0, newIntrospectBadRequestError("Unable to parse HTTP body, make sure to send a properly formatted form request body.")
	} else if len(r.PostForm) == 0 {
		return "", 0, newIntrospectBadRequestError("The POST body can not be empty.")
	} else if !r.PostForm.Has("token") {
		return "", 0, newIntrospectBadRequestError("The POST body doesn't contain 'token' parameter.")
	}

	token := r.PostForm.Get("token")
	tokenType, err := h.guessTokenType(r.Context(), token)
	if err != nil {
		h.Logger.ErrorContext(r.Context(), "failed to guess token type", "err", err)
		return "", 0, newIntrospectInternalServerError()
	}

	requestTokenType := r.PostForm.Get("token_type_hint")
	if requestTokenType != "" {
		if tokenType.String() != requestTokenType {
			h.Logger.Warn("token type hint doesn't match token type", "request_token_type", requestTokenType, "token_type", tokenType)
		}
	}

	return token, tokenType, nil
}

func (h *Handler) introspectRefreshToken(ctx context.Context, token string) (*Introspection, error) {
	rToken := new(internal.RefreshToken)
	if err := internal.Unmarshal(token, rToken); err != nil {
		// For backward compatibility, assume the refresh_token is a raw refresh token ID
		// if it fails to decode.
		//
		// Because refresh_token values that aren't unmarshable were generated by servers
		// that don't have a Token value, we'll still reject any attempts to claim a
		// refresh_token twice.
		rToken = &internal.RefreshToken{RefreshId: token, Token: ""}
	}

	refresh, err := tokens.LookupRefreshToken(ctx, h.Storage, h.RefreshPolicy, h.Logger, nil, rToken)
	if err != nil {
		// A rejected token (unknown, revoked or expired) is reported as inactive;
		// only an infrastructure failure is a server error.
		if errors.Is(err, tokens.ErrRefreshTokenInvalid) ||
			errors.Is(err, tokens.ErrRefreshTokenExpired) ||
			errors.Is(err, tokens.ErrRefreshTokenClaimedByOtherClient) {
			return nil, newIntrospectInactiveTokenError()
		}
		h.Logger.ErrorContext(ctx, "failed to get refresh token", "err", err)
		return nil, newIntrospectInternalServerError()
	}

	subjectString, sErr := tokens.GenSubject(refresh.Claims.UserID, refresh.ConnectorID)
	if sErr != nil {
		h.Logger.ErrorContext(ctx, "failed to marshal offline session ID", "err", sErr)
		return nil, newIntrospectInternalServerError()
	}

	return &Introspection{
		Active:    true,
		ClientID:  refresh.ClientID,
		IssuedAt:  refresh.CreatedAt.Unix(),
		NotBefore: refresh.CreatedAt.Unix(),
		Expiry:    refresh.CreatedAt.Add(h.RefreshPolicy.AbsoluteLifetime()).Unix(),
		Subject:   subjectString,
		Username:  refresh.Claims.PreferredUsername,
		// Refresh-token introspection does not resolve scopes, so the audience is
		// the token's own client only.
		Audience: tokens.GetAudience(refresh.ClientID, nil),
		Issuer:   h.Issuer,

		Extra: IntrospectionExtra{
			Email:             refresh.Claims.Email,
			EmailVerified:     &refresh.Claims.EmailVerified,
			Groups:            refresh.Claims.Groups,
			Name:              refresh.Claims.Username,
			PreferredUsername: refresh.Claims.PreferredUsername,
		},
		TokenType: "Bearer",
		TokenUse:  "refresh_token",
	}, nil
}

func (h *Handler) introspectAccessToken(ctx context.Context, token string) (*Introspection, error) {
	verifier := oidc.NewVerifier(h.Issuer, &signer.KeySet{Signer: h.Signer}, &oidc.Config{SkipClientIDCheck: true})
	idToken, err := verifier.Verify(ctx, token)
	if err != nil {
		return nil, newIntrospectInactiveTokenError()
	}

	var claims IntrospectionExtra
	if err := idToken.Claims(&claims); err != nil {
		h.Logger.ErrorContext(ctx, "error while fetching token claims", "err", err.Error())
		return nil, newIntrospectInternalServerError()
	}

	clientID, err := tokens.GetClientID(idToken.Audience, claims.AuthorizingParty)
	if err != nil {
		h.Logger.ErrorContext(ctx, "error while fetching client_id from token:", "err", err.Error())
		return nil, newIntrospectInternalServerError()
	}

	client, err := h.Storage.GetClient(ctx, clientID)
	if err != nil {
		h.Logger.ErrorContext(ctx, "error while fetching client from storage", "err", err.Error())
		return nil, newIntrospectInternalServerError()
	}

	return &Introspection{
		Active:    true,
		ClientID:  client.ID,
		IssuedAt:  idToken.IssuedAt.Unix(),
		NotBefore: idToken.IssuedAt.Unix(),
		Expiry:    idToken.Expiry.Unix(),
		Subject:   idToken.Subject,
		Username:  claims.PreferredUsername,
		Audience:  idToken.Audience,
		Issuer:    h.Issuer,

		Extra:     claims,
		TokenType: "Bearer",
		TokenUse:  "access_token",
	}, nil
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var introspect *Introspection
	token, tokenType, err := h.getTokenFromRequest(r)
	if err == nil {
		switch tokenType {
		case AccessToken:
			introspect, err = h.introspectAccessToken(ctx, token)
		case RefreshToken:
			introspect, err = h.introspectRefreshToken(ctx, token)
		default:
			// Token type is neither handled token types.
			h.Logger.ErrorContext(ctx, "unknown token type", "token_type", tokenType)
			introspectInactiveErr(w)
			return
		}
	}

	if err != nil {
		if intErr, ok := err.(*introspectionError); ok {
			h.introspectErrHelper(w, intErr.typ, intErr.desc, intErr.code)
		} else {
			h.Logger.ErrorContext(ctx, "an unknown error occurred", "err", err.Error())
			h.introspectErrHelper(w, oauth2.ServerError, "An unknown error occurred", http.StatusInternalServerError)
		}

		return
	}

	rawJSON, jsonErr := json.Marshal(introspect)
	if jsonErr != nil {
		h.Logger.ErrorContext(ctx, "failed to marshal introspection response", "err", jsonErr)
		h.introspectErrHelper(w, oauth2.ServerError, "", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(rawJSON)
}

func (h *Handler) introspectErrHelper(w http.ResponseWriter, typ, description string, statusCode int) {
	if typ == oauth2.InactiveToken {
		introspectInactiveErr(w)
		return
	}

	if err := oauth2.WriteError(w, typ, description, statusCode); err != nil {
		h.Logger.Error("introspect error response", "err", err)
	}
}

func introspectInactiveErr(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(struct {
		Active bool `json:"active"`
	}{Active: false})
}

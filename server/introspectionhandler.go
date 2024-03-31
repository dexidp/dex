package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/dexidp/dex/server/internal"
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
	Audience audience `json:"aud"`

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

	FederatedIDClaims *federatedIDClaims `json:"federated_claims,omitempty"`
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
	return &introspectionError{typ: errInactiveToken, desc: "", code: http.StatusUnauthorized}
}

func newIntrospectInternalServerError() *introspectionError {
	return &introspectionError{typ: errServerError, desc: "", code: http.StatusInternalServerError}
}

func newIntrospectBadRequestError(desc string) *introspectionError {
	return &introspectionError{typ: errInvalidRequest, desc: desc, code: http.StatusBadRequest}
}

func (s *Server) guessTokenType(ctx context.Context, token string) (TokenTypeEnum, error) {
	// We skip every checks, we only want to know if it's a valid JWT
	verifierConfig := oidc.Config{
		SkipClientIDCheck: true,
		SkipExpiryCheck:   true,
		SkipIssuerCheck:   true,

		// We skip signature checks to avoid database calls;
		InsecureSkipSignatureCheck: true,
	}

	verifier := oidc.NewVerifier(s.issuerURL.String(), nil, &verifierConfig)
	if _, err := verifier.Verify(ctx, token); err != nil {
		// If it's not an access token, let's assume it's a refresh token;
		return RefreshToken, nil
	}

	// If it's a valid JWT, it's an access token.
	return AccessToken, nil
}

func (s *Server) getTokenFromRequest(r *http.Request) (string, TokenTypeEnum, error) {
	if r.Method != "POST" {
		return "", 0, newIntrospectBadRequestError(fmt.Sprintf("HTTP method is \"%s\", expected \"POST\".", r.Method))
	} else if err := r.ParseForm(); err != nil {
		return "", 0, newIntrospectBadRequestError("Unable to parse HTTP body, make sure to send a properly formatted form request body.")
	} else if r.PostForm == nil || len(r.PostForm) == 0 {
		return "", 0, newIntrospectBadRequestError("The POST body can not be empty.")
	} else if !r.PostForm.Has("token") {
		return "", 0, newIntrospectBadRequestError("The POST body doesn't contain 'token' parameter.")
	}

	token := r.PostForm.Get("token")
	tokenType, err := s.guessTokenType(r.Context(), token)
	if err != nil {
		s.logger.Error(err)
		return "", 0, newIntrospectInternalServerError()
	}

	requestTokenType := r.PostForm.Get("token_type_hint")
	if requestTokenType != "" {
		if tokenType.String() != requestTokenType {
			s.logger.Warnf("Token type hint doesn't match token type: %s != %s", requestTokenType, tokenType)
		}
	}

	return token, tokenType, nil
}

func (s *Server) introspectRefreshToken(_ context.Context, token string) (*Introspection, error) {
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

	rCtx, err := s.getRefreshTokenFromStorage(nil, rToken)
	if err != nil {
		if errors.Is(err, invalidErr) || errors.Is(err, expiredErr) {
			return nil, newIntrospectInactiveTokenError()
		}

		s.logger.Errorf("failed to get refresh token: %v", err)
		return nil, newIntrospectInternalServerError()
	}

	subjectString, sErr := genSubject(rCtx.storageToken.Claims.UserID, rCtx.storageToken.ConnectorID)
	if sErr != nil {
		s.logger.Errorf("failed to marshal offline session ID: %v", err)
		return nil, newIntrospectInternalServerError()
	}

	return &Introspection{
		Active:    true,
		ClientID:  rCtx.storageToken.ClientID,
		IssuedAt:  rCtx.storageToken.CreatedAt.Unix(),
		NotBefore: rCtx.storageToken.CreatedAt.Unix(),
		Expiry:    rCtx.storageToken.CreatedAt.Add(s.refreshTokenPolicy.absoluteLifetime).Unix(),
		Subject:   subjectString,
		Username:  rCtx.storageToken.Claims.PreferredUsername,
		Audience:  getAudience(rCtx.storageToken.ClientID, rCtx.scopes),
		Issuer:    s.issuerURL.String(),

		Extra: IntrospectionExtra{
			Email:             rCtx.storageToken.Claims.Email,
			EmailVerified:     &rCtx.storageToken.Claims.EmailVerified,
			Groups:            rCtx.storageToken.Claims.Groups,
			Name:              rCtx.storageToken.Claims.Username,
			PreferredUsername: rCtx.storageToken.Claims.PreferredUsername,
		},
		TokenType: "Bearer",
		TokenUse:  "refresh_token",
	}, nil
}

func (s *Server) introspectAccessToken(ctx context.Context, token string) (*Introspection, error) {
	verifier := oidc.NewVerifier(s.issuerURL.String(), &storageKeySet{s.storage}, &oidc.Config{SkipClientIDCheck: true})
	idToken, err := verifier.Verify(ctx, token)
	if err != nil {
		return nil, newIntrospectInactiveTokenError()
	}

	var claims IntrospectionExtra
	if err := idToken.Claims(&claims); err != nil {
		s.logger.Errorf("Error while fetching token claims: %s", err.Error())
		return nil, newIntrospectInternalServerError()
	}

	clientID, err := getClientID(idToken.Audience, claims.AuthorizingParty)
	if err != nil {
		s.logger.Error("Error while fetching client_id from token: %s", err.Error())
		return nil, newIntrospectInternalServerError()
	}

	client, err := s.storage.GetClient(clientID)
	if err != nil {
		s.logger.Error("Error while fetching client from storage: %s", err.Error())
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
		Issuer:    s.issuerURL.String(),

		Extra:     claims,
		TokenType: "Bearer",
		TokenUse:  "access_token",
	}, nil
}

func (s *Server) handleIntrospect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var introspect *Introspection
	token, tokenType, err := s.getTokenFromRequest(r)
	if err == nil {
		switch tokenType {
		case AccessToken:
			introspect, err = s.introspectAccessToken(ctx, token)
		case RefreshToken:
			introspect, err = s.introspectRefreshToken(ctx, token)
		default:
			// Token type is neither handled token types.
			s.logger.Errorf("Unknown token type: %s", tokenType)
			introspectInactiveErr(w)
			return
		}
	}

	if err != nil {
		if intErr, ok := err.(*introspectionError); ok {
			s.introspectErrHelper(w, intErr.typ, intErr.desc, intErr.code)
		} else {
			s.logger.Errorf("An unknown error occurred: %s", err.Error())
			s.introspectErrHelper(w, errServerError, "An unknown error occurred", http.StatusInternalServerError)
		}

		return
	}

	rawJSON, jsonErr := json.Marshal(introspect)
	if jsonErr != nil {
		s.introspectErrHelper(w, errServerError, jsonErr.Error(), 500)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(rawJSON)
}

func (s *Server) introspectErrHelper(w http.ResponseWriter, typ string, description string, statusCode int) {
	if typ == errInactiveToken {
		introspectInactiveErr(w)
		return
	}

	if err := tokenErr(w, typ, description, statusCode); err != nil {
		s.logger.Errorf("introspect error response: %v", err)
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

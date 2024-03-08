package server

type IntrospectionExtra struct {
	AuthorizingParty string `json:"azp,omitempty"`

	Email         string `json:"email,omitempty"`
	EmailVerified *bool  `json:"email_verified,omitempty"`

	Groups []string `json:"groups,omitempty"`

	Name              string `json:"name,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`

	FederatedIDClaims *federatedIDClaims `json:"federated_claims,omitempty"`
}

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

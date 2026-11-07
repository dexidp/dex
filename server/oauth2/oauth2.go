package oauth2

// Error codes returned in OAuth2/OIDC error responses (RFC 6749 §5.2, the OIDC
// Core spec, and RFC 7662 for token introspection).
const (
	InvalidRequest          = "invalid_request"
	UnauthorizedClient      = "unauthorized_client"
	AccessDenied            = "access_denied"
	UnsupportedResponseType = "unsupported_response_type"
	RequestNotSupported     = "request_not_supported"
	InvalidScope            = "invalid_scope"
	ServerError             = "server_error"
	TemporarilyUnavailable  = "temporarily_unavailable"
	UnsupportedGrantType    = "unsupported_grant_type"
	InvalidGrant            = "invalid_grant"
	InvalidClient           = "invalid_client"
	InactiveToken           = "inactive_token"
	LoginRequired           = "login_required"
	InteractionRequired     = "interaction_required"
	ConsentRequired         = "consent_required"
)

// Grant types supported at the token endpoint.
const (
	GrantTypeAuthorizationCode = "authorization_code"
	GrantTypeRefreshToken      = "refresh_token"
	GrantTypeImplicit          = "implicit"
	GrantTypePassword          = "password"
	GrantTypeDeviceCode        = "urn:ietf:params:oauth:grant-type:device_code"
	GrantTypeTokenExchange     = "urn:ietf:params:oauth:grant-type:token-exchange"
	GrantTypeClientCredentials = "client_credentials"
)

// Token-exchange token types (RFC 8693 §3).
const (
	TokenTypeAccess  = "urn:ietf:params:oauth:token-type:access_token"
	TokenTypeRefresh = "urn:ietf:params:oauth:token-type:refresh_token"
	TokenTypeID      = "urn:ietf:params:oauth:token-type:id_token"
	TokenTypeSAML1   = "urn:ietf:params:oauth:token-type:saml1"
	TokenTypeSAML2   = "urn:ietf:params:oauth:token-type:saml2"
	TokenTypeJWT     = "urn:ietf:params:oauth:token-type:jwt"
)

// Response types for the authorization endpoint.
const (
	ResponseTypeCode             = "code"                // "Regular" flow
	ResponseTypeToken            = "token"               // Implicit flow for frontend apps.
	ResponseTypeIDToken          = "id_token"            // ID Token in url fragment
	ResponseTypeCodeToken        = "code token"          // "Regular" flow + Implicit flow
	ResponseTypeCodeIDToken      = "code id_token"       // "Regular" flow + ID Token
	ResponseTypeIDTokenToken     = "id_token token"      // ID Token + Implicit flow
	ResponseTypeCodeIDTokenToken = "code id_token token" // "Regular" flow + ID Token + Implicit flow
)

// Device authorization grant poll statuses returned at the token endpoint.
const (
	DeviceTokenPending  = "authorization_pending"
	DeviceTokenComplete = "complete"
	DeviceTokenSlowDown = "slow_down"
	DeviceTokenExpired  = "expired_token"
)

// Special redirect URIs for native and device clients.
const (
	// RedirectURIOOB is the out-of-band redirect for clients without a browser.
	RedirectURIOOB = "urn:ietf:wg:oauth:2.0:oob"
	// DeviceCallbackURI is the callback path used by the device authorization grant.
	DeviceCallbackURI = "/device/callback"
)

// PKCE code challenge methods (RFC 7636 §4.2).
const (
	PKCEMethodPlain = "plain"
	PKCEMethodS256  = "S256"
)

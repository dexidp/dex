package cel

import (
	"github.com/google/cel-go/cel"

	"github.com/dexidp/dex/connector"
)

// VariableDeclaration declares a named variable and its CEL type
// that will be available in expressions.
type VariableDeclaration struct {
	Name string
	Type *cel.Type
}

// IdentityVal is the CEL native type for the identity variable.
// Fields are typed so that the CEL compiler rejects unknown field access
// (e.g. identity.emial) at config load time rather than at evaluation time.
type IdentityVal struct {
	UserID            string   `cel:"user_id"`
	Username          string   `cel:"username"`
	PreferredUsername string   `cel:"preferred_username"`
	Email             string   `cel:"email"`
	EmailVerified     bool     `cel:"email_verified"`
	Groups            []string `cel:"groups"`
}

// RequestVal is the CEL native type for the request variable.
type RequestVal struct {
	ClientID    string   `cel:"client_id"`
	ConnectorID string   `cel:"connector_id"`
	Scopes      []string `cel:"scopes"`
	RedirectURI string   `cel:"redirect_uri"`
}

// identityTypeName is the CEL type name for IdentityVal.
// Derived by ext.NativeTypes as simplePkgAlias(pkgPath) + "." + structName.
const identityTypeName = "cel.IdentityVal"

// requestTypeName is the CEL type name for RequestVal.
const requestTypeName = "cel.RequestVal"

// IdentityVariables provides the 'identity' variable with typed fields.
//
//	identity.user_id            — string
//	identity.username           — string
//	identity.preferred_username — string
//	identity.email              — string
//	identity.email_verified     — bool
//	identity.groups             — list(string)
func IdentityVariables() []VariableDeclaration {
	return []VariableDeclaration{
		{Name: "identity", Type: cel.ObjectType(identityTypeName)},
	}
}

// RequestVariables provides the 'request' variable with typed fields.
//
//	request.client_id     — string
//	request.connector_id  — string
//	request.scopes        — list(string)
//	request.redirect_uri  — string
func RequestVariables() []VariableDeclaration {
	return []VariableDeclaration{
		{Name: "request", Type: cel.ObjectType(requestTypeName)},
	}
}

// ClaimsVariable provides a 'claims' map for raw upstream claims.
// Claims remain map(string, dyn) because their shape is genuinely
// unknown — they carry arbitrary upstream IdP data.
//
//	claims — map(string, dyn)
func ClaimsVariable() []VariableDeclaration {
	return []VariableDeclaration{
		{Name: "claims", Type: cel.MapType(cel.StringType, cel.DynType)},
	}
}

// IdentityFromConnector converts a connector.Identity to a CEL-compatible IdentityVal.
func IdentityFromConnector(id connector.Identity) IdentityVal {
	return IdentityVal{
		UserID:            id.UserID,
		Username:          id.Username,
		PreferredUsername: id.PreferredUsername,
		Email:             id.Email,
		EmailVerified:     id.EmailVerified,
		Groups:            id.Groups,
	}
}

// RequestContext represents the authentication/token request context
// available as the 'request' variable in CEL expressions.
type RequestContext struct {
	ClientID    string
	ConnectorID string
	Scopes      []string
	RedirectURI string
}

// RequestFromContext converts a RequestContext to a CEL-compatible RequestVal.
func RequestFromContext(rc RequestContext) RequestVal {
	return RequestVal{
		ClientID:    rc.ClientID,
		ConnectorID: rc.ConnectorID,
		Scopes:      rc.Scopes,
		RedirectURI: rc.RedirectURI,
	}
}

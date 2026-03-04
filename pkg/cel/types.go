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

// IdentityVariables provides the 'identity' variable with user claims.
//
//	identity.user_id            — string
//	identity.username           — string
//	identity.preferred_username — string
//	identity.email              — string
//	identity.email_verified     — bool
//	identity.groups             — list(string)
func IdentityVariables() []VariableDeclaration {
	return []VariableDeclaration{
		{Name: "identity", Type: cel.MapType(cel.StringType, cel.DynType)},
	}
}

// RequestVariables provides the 'request' variable with request context.
//
//	request.client_id     — string
//	request.connector_id  — string
//	request.scopes        — list(string)
//	request.redirect_uri  — string
func RequestVariables() []VariableDeclaration {
	return []VariableDeclaration{
		{Name: "request", Type: cel.MapType(cel.StringType, cel.DynType)},
	}
}

// ClaimsVariable provides a 'claims' map for raw upstream claims.
//
//	claims — map(string, dyn)
func ClaimsVariable() []VariableDeclaration {
	return []VariableDeclaration{
		{Name: "claims", Type: cel.MapType(cel.StringType, cel.DynType)},
	}
}

// IdentityFromConnector converts a connector.Identity to a CEL-compatible map.
func IdentityFromConnector(id connector.Identity) map[string]any {
	return map[string]any{
		"user_id":            id.UserID,
		"username":           id.Username,
		"preferred_username": id.PreferredUsername,
		"email":              id.Email,
		"email_verified":     id.EmailVerified,
		"groups":             id.Groups,
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

// RequestFromContext converts a RequestContext to a CEL-compatible map.
func RequestFromContext(rc RequestContext) map[string]any {
	return map[string]any{
		"client_id":    rc.ClientID,
		"connector_id": rc.ConnectorID,
		"scopes":       rc.Scopes,
		"redirect_uri": rc.RedirectURI,
	}
}

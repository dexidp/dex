package keystone

// Config holds the configuration parameters for Keystone connector.
// Keystone should expose API v3
type Config struct {
	Domain             string `json:"domain"`
	Host               string `json:"keystoneHost"`
	AdminUsername      string `json:"keystoneUsername"`
	AdminPassword      string `json:"keystonePassword"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify"`
	CustomerName       string `json:"customerName"`
}

// FederationConfig holds the configuration parameters for Keystone federation connector.
// This connector supports SSO authentication via Shibboleth and SAML.
type FederationConfig struct {
	// Domain is domain ID, typically "default"
	Domain string `json:"domain"`
	// Host is Keystone host URL, e.g. https://keystone.pf9.com:5000
	Host string `json:"keystoneHost"`
	// AdminUsername is Keystone admin username
	AdminUsername string `json:"keystoneUsername"`
	// AdminPassword is Keystone admin password
	AdminPassword string `json:"keystonePassword"`
	// CustomerName is customer name to be used in group names
	CustomerName string `json:"customerName"`
	// ShibbolethLoginPath is Shibboleth SSO login endpoint path, typically '/sso/{IdP}/Shibboleth.sso/Login'
	ShibbolethLoginPath string `json:"shibbolethLoginPath,omitempty"`
	// FederationAuthPath is OS-FEDERATION identity providers auth path, typically '/keystone/v3/OS-FEDERATION/identity_providers/{IdP}/protocols/saml2/auth'
	FederationAuthPath string `json:"federationAuthPath,omitempty"`
	// TimeoutSeconds is the timeout for HTTP requests in seconds
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`
}

// userKeystone represents a Keystone user
type userKeystone struct {
	Domain       domainKeystone `json:"domain"`
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	OSFederation *struct {
		Groups           []keystoneGroup `json:"groups"`
		IdentityProvider struct {
			ID string `json:"id"`
		} `json:"identity_provider"`
		Protocol struct {
			ID string `json:"id"`
		} `json:"protocol"`
	} `json:"OS-FEDERATION"`
}

// domainKeystone represents a Keystone domain
type domainKeystone struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// loginRequestData represents a login request with unscoped authorization
type loginRequestData struct {
	auth `json:"auth"`
}

// auth represents the authentication part of a login request
type auth struct {
	Identity identity `json:"identity"`
}

// identity represents the identity part of an authentication request
type identity struct {
	Methods  []string `json:"methods"`
	Password password `json:"password"`
}

// password represents the password authentication method
type password struct {
	User user `json:"user"`
}

// user represents a user for authentication
type user struct {
	Name     string         `json:"name"`
	Domain   domainKeystone `json:"domain"`
	Password string         `json:"password"`
}

// tokenInfo represents information about a token
type tokenInfo struct {
	User  userKeystone `json:"user"`
	Roles []role       `json:"roles"`
}

// tokenResponse represents a response containing a token
type tokenResponse struct {
	Token tokenInfo `json:"token"`
}

// keystoneGroup represents a Keystone group
type keystoneGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// groupsResponse represents a response containing groups
type groupsResponse struct {
	Groups []keystoneGroup `json:"groups"`
}

// userResponse represents a response containing user information
type userResponse struct {
	User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		ID    string `json:"id"`
	} `json:"user"`
}

// role represents a Keystone role
type role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DomainID    string `json:"domain_id"`
	Description string `json:"description"`
}

// project represents a Keystone project
type project struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DomainID    string `json:"domain_id"`
	Description string `json:"description"`
}

// identifierContainer represents an object with an ID
type identifierContainer struct {
	ID string `json:"id"`
}

// projectScope represents a project scope for authorization
type projectScope struct {
	Project identifierContainer `json:"project"`
}

// roleAssignment represents a role assignment
type roleAssignment struct {
	Scope projectScope        `json:"scope"`
	User  identifierContainer `json:"user"`
	Role  identifierContainer `json:"role"`
}

// connectorData represents data stored with the connector
type connectorData struct {
	Token string `json:"token"`
}

// getRoleAssignmentsOptions represents options for getting role assignments
type getRoleAssignmentsOptions struct {
	userID  string
	groupID string
}

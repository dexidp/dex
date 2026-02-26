package featureflags

var (
	// EntEnabled enables experimental ent-based engine for the database storages.
	// https://entgo.io/
	EntEnabled = newFlag("ent_enabled", false)

	// ExpandEnv can enable or disable env expansion in the config which can be useful in environments where, e.g.,
	// $ sign is a part of the password for LDAP user.
	ExpandEnv = newFlag("expand_env", true)

	// APIConnectorsCRUD allows CRUD operations on connectors through the gRPC API
	APIConnectorsCRUD = newFlag("api_connectors_crud", false)

	// ContinueOnConnectorFailure allows the server to start even if some connectors fail to initialize.
	ContinueOnConnectorFailure = newFlag("continue_on_connector_failure", true)

	// ConfigDisallowUnknownFields enables to forbid unknown fields in the config while unmarshaling.
	ConfigDisallowUnknownFields = newFlag("config_disallow_unknown_fields", false)

  // ClientCredentialGrantEnabledByDefault enables the client_credentials grant type by default
	// without requiring explicit configuration in oauth2.grantTypes.
	ClientCredentialGrantEnabledByDefault = newFlag("client_credential_grant_enabled_by_default", false)
)
